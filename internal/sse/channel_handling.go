package sse

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/sarpt/mpv-web-api/internal/state"
)

var (
	sseEventEnd = []byte("\n\n")

	errResponseJSONCreationFailed = errors.New("could not create JSON for response")
	errClientWritingFailed        = errors.New("could not write to the client")
	errConvertToFlusherFailed     = errors.New("could not instantiate http sse flusher")
	errIncorrectChangesType       = errors.New("changes of incorrect type provided to the change handler")
)

const (
	replaySseStateArg = "replay"
	sseChannelArg     = "channel"

	// ObserverAdded informs about new observer being added to the SSE server.
	ObserverAdded ObserverChangeVariant = "observer-added"

	// ObserverRemoved informs about new observer being removed from the SSE server.
	ObserverRemoved ObserverChangeVariant = "observer-removed"
)

// observers represents client observers that are currently connected to this instance of api server
type observers struct {
	items map[string]chan interface{}
	lock  *sync.RWMutex
}

// ObserverChangeVariant specifies what change to the state the specified observers change specifies (addition, removal, etc.).
type ObserverChangeVariant string

// ObserversChange informs about changes to the SSE observers list.
type ObserversChange struct {
	ChangeVariant  ObserverChangeVariant
	RemoteAddr     string
	ChannelVariant state.SSEChannelVariant
}

type getSseHandler = func(res http.ResponseWriter, req *http.Request)
type sseReplayHandler = func(res ResponseWriter) error
type sseChangeHandler = func(res ResponseWriter, change interface{}) error

// channel is used to construct channel on which subscribers can listen to on a mutexed on a single SSE keep-alive connection
type channel struct {
	observers     observers
	variant       state.SSEChannelVariant
	changeHandler sseChangeHandler
	replayHandler sseReplayHandler
}

// handlerConfig is used to control creation of SSE handler for Server
type handlerConfig struct {
	Channels map[state.SSEChannelVariant]channel
}

func (s *Server) createGetSseHandler(cfg handlerConfig) getSseHandler {
	return func(res http.ResponseWriter, req *http.Request) {
		sseResWriter, err := sseResponseWriter(res)
		if err != nil {
			res.WriteHeader(400)
			return
		}

		wg := &sync.WaitGroup{}

		channelVariants := req.URL.Query()[sseChannelArg]
		for _, reqChannel := range channelVariants {
			channelVariant := state.SSEChannelVariant(reqChannel)

			channel, ok := cfg.Channels[channelVariant]
			if !ok {
				continue
			}

			wg.Add(1)
			go s.observeChannelVariant(sseResWriter, req, channel, wg)
		}

		wg.Wait()
		s.outLog.Printf("all sse channels closed for %s", req.RemoteAddr)
	}
}

func (s *Server) observeChannelVariant(res ResponseWriter, req *http.Request, sseChannel channel, wg *sync.WaitGroup) {
	defer wg.Done()

	remoteAddr := req.RemoteAddr
	// Buffer of 1 in case connection is closed after playbackObservers fan-out dispatcher already acquired read lock (blocking the write lock).
	// The dispatcher will expect for the select below to receive the message but the Context().Done() already waits to acquire a write lock acquired by the dispatcher in order to send on a channel.
	// So the buffer of 1 ensures that one message will be buffered, dispatcher will not be blocked, and write lock will be obtained.
	// When the write lock is obtained to remove from the set, even if a new playback will be received, read lock will wait until Context().Done() finishes.
	// TODO: create a separate goroutine that does not block select in waiting for the Context().Done().
	// When that goroutine ends, it should emit on done channel that is present in select, which is used for finishing the "for" looping.
	// By waiting in the separate goroutine, the Select below will not choose at random which chan should be handled, eliminating possiblity that Context().Done()
	// is chosen during looping in "for" in fanout dispatcher which has already acquired lock. Context().Done() will still have to wait for unlock, but will not block
	// next iteration of "for" in fanout dispatcher waiting for Select case to (randomly) decide it's turn.
	changes := make(chan interface{}, 1)
	sseChannel.observers.lock.Lock()
	sseChannel.observers.items[remoteAddr] = changes
	sseChannel.observers.lock.Unlock()

	if s.observersChange != nil {
		s.observersChange <- ObserversChange{
			ChangeVariant:  ObserverAdded,
			RemoteAddr:     req.RemoteAddr,
			ChannelVariant: sseChannel.variant,
		}
	}
	s.outLog.Printf("added %s observer with addr %s\n", sseChannel.variant, remoteAddr)

	if replaySseState(req) {
		err := sseChannel.replayHandler(res)
		if err != nil {
			s.errLog.Println(fmt.Sprintf("could not replay data on sse: %s", err.Error()))
		}
	}

	for {
		select {
		case change, closed := <-changes:
			if !closed {
				s.outLog.Printf("sse observation on channel %s done for %s due to changes channel being closed\n", sseChannel.variant, remoteAddr)

				return
			}

			err := sseChannel.changeHandler(res, change)
			if err != nil {
				s.errLog.Println(err.Error())
			}
		case <-req.Context().Done():
			sseChannel.observers.lock.Lock()
			delete(sseChannel.observers.items, remoteAddr)
			sseChannel.observers.lock.Unlock()

			if s.observersChange != nil {
				s.observersChange <- ObserversChange{
					ChangeVariant:  ObserverRemoved,
					RemoteAddr:     req.RemoteAddr,
					ChannelVariant: sseChannel.variant,
				}
			}
			s.outLog.Printf("removing %s observer with addr %s\n", sseChannel.variant, remoteAddr)

			return
		}
	}
}

func sseResponseWriter(res http.ResponseWriter) (ResponseWriter, error) {
	flusher, ok := res.(http.Flusher)
	if !ok {
		return ResponseWriter{}, errConvertToFlusherFailed
	}

	res.Header().Set("Connection", "keep-alive")
	res.Header().Set("Content-Type", "text/event-stream")
	res.Header().Set("Access-Control-Allow-Origin", "*")

	sseFlusher := ResponseWriter{
		res:     res,
		flusher: flusher,
		lock:    &sync.Mutex{},
	}
	return sseFlusher, nil
}

func replaySseState(req *http.Request) bool {
	replay, ok := req.URL.Query()[replaySseStateArg]

	return ok && len(replay) > 0 && replay[0] == "true"
}

func formatSseEvent(channel state.SSEChannelVariant, eventName string, data []byte) []byte {
	var out []byte

	channelEvent := fmt.Sprintf("%s.%s", channel, eventName)
	out = append(out, []byte(fmt.Sprintf("event:%s\n", channelEvent))...)

	dataEntries := bytes.Split(data, []byte("\n"))
	for _, dataEntry := range dataEntries {
		out = append(out, []byte(fmt.Sprintf("data:%s\n", dataEntry))...)
	}

	out = append(out, sseEventEnd...)
	return out
}

// distributeChangesToChannelObservers is a fan-out dispatcher, which notifies all playback observers (subscribers from SSE etc.) when a playbackChange occurs.
func distributeChangesToChannelObservers(channelChanges <-chan interface{}, channelObservers observers) {
	for {
		change, ok := <-channelChanges
		if !ok {
			return
		}

		channelObservers.lock.RLock()
		for _, observer := range channelObservers.items {
			observer <- change
		}
		channelObservers.lock.RUnlock()
	}
}
