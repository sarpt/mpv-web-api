package api

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

const (
	replaySseStateArg = "replay"
	sseChannelArg     = "channel"
)

// SSEObservers represents client observers that are currently connected to this instance of api server
type SSEObservers struct {
	Items map[string]chan interface{}
	Lock  *sync.RWMutex
}

type getSseHandler = func(res http.ResponseWriter, req *http.Request)
type sseReplayHandler = func(res http.ResponseWriter, flusher http.Flusher) error
type sseChangeHandler = func(res http.ResponseWriter, flusher http.Flusher, change interface{}) error

// SSEChannel is used to construct channel on which subscribers can listen to on a mutexed on a single SSE keep-alive connection
type SSEChannel struct {
	Observers     SSEObservers
	Variant       SSEChannelVariant
	ChangeHandler sseChangeHandler
	ReplayHandler sseReplayHandler
}

// SseHandlerConfig is used to control creation of SSE handler for Server
type SseHandlerConfig struct {
	Channels map[SSEChannelVariant]SSEChannel
}

var (
	sseEventEnd = []byte("\n\n")

	errResponseJSONCreationFailed = errors.New("could not create JSON for response")
	errClientWritingFailed        = errors.New("could not write to the client")
	errConvertToFlusherFailed     = errors.New("could not instantiate http sse flusher")
	errIncorrectChangesType       = errors.New("changes of incorrect type provided to the change handler")
)

func sseFlusher(res http.ResponseWriter) (http.Flusher, error) {
	flusher, ok := res.(http.Flusher)
	if !ok {
		return flusher, errConvertToFlusherFailed
	}

	res.Header().Set("Connection", "keep-alive")
	res.Header().Set("Content-Type", "text/event-stream")
	res.Header().Set("Access-Control-Allow-Origin", "*")

	return flusher, nil
}

func replaySseState(req *http.Request) bool {
	replay, ok := req.URL.Query()[replaySseStateArg]

	return ok && len(replay) > 0 && replay[0] == "true"
}

func formatSseEvent(eventName string, data []byte) []byte {
	var out []byte

	out = append(out, []byte(fmt.Sprintf("event:%s\n", eventName))...)

	dataEntries := bytes.Split(data, []byte("\n"))
	for _, dataEntry := range dataEntries {
		out = append(out, []byte(fmt.Sprintf("data:%s\n", dataEntry))...)
	}

	out = append(out, sseEventEnd...)
	return out
}

func (s *Server) createGetSseHandler(cfg SseHandlerConfig) getSseHandler {
	return func(res http.ResponseWriter, req *http.Request) {
		flusher, err := sseFlusher(res)
		if err != nil {
			res.WriteHeader(400)
			return
		}

		wg := &sync.WaitGroup{}

		channelVariants := req.URL.Query()[sseChannelArg]
		for _, reqChannel := range channelVariants {
			channelVariant := SSEChannelVariant(reqChannel)

			channel, ok := cfg.Channels[channelVariant]
			if !ok {
				continue
			}

			wg.Add(1)
			go s.observeChannelVariant(res, req, flusher, channel, wg)
		}

		wg.Wait()
	}
}

func (s *Server) observeChannelVariant(res http.ResponseWriter, req *http.Request, flusher http.Flusher, channel SSEChannel, wg *sync.WaitGroup) {
	defer wg.Done()

	remoteAddr := req.RemoteAddr
	// Buffer of 1 in case connection is closed after playbackObservers fan-out dispatcher already acquired read lock (blocking the write lock).
	// The dispatcher will expect for the select below to receive the message but the Context().Done() already waits to acquire a write lock acquired by the dispatcher in order to send on a channel.
	// So the buffer of 1 ensures that one message will be buffered, dispatcher will not be blocked, and write lock will be obtained.
	// When the write lock is obtained to remove from the set, even if a new playback will be received, read lock will wait until Context().Done() finishes.
	changes := make(chan interface{}, 1)
	channel.Observers.Lock.Lock()
	channel.Observers.Items[remoteAddr] = changes
	channel.Observers.Lock.Unlock()

	s.status.addObservingAddress(req.RemoteAddr, channel.Variant)
	s.outLog.Printf("added %s observer with addr %s\n", channel.Variant, remoteAddr)

	if replaySseState(req) {
		err := channel.ReplayHandler(res, flusher)
		if err != nil {
			s.errLog.Println(err.Error())
		}
	}

	for {
		select {
		case change, ok := <-changes:
			if !ok {
				return
			}

			err := channel.ChangeHandler(res, flusher, change)
			if err != nil {
				s.errLog.Println(err.Error())
			}
		case <-req.Context().Done():
			channel.Observers.Lock.Lock()
			delete(channel.Observers.Items, remoteAddr)
			channel.Observers.Lock.Unlock()

			s.status.removeObservingAddress(req.RemoteAddr, channel.Variant)
			s.outLog.Printf("removing %s observer with addr %s\n", channel.Variant, remoteAddr)

			return
		}
	}
}
