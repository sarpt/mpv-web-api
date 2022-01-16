package sse

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/state"
)

var (
	sseEventEnd = []byte("\n\n")

	errResponseJSONCreationFailed = errors.New("could not create JSON for response")
	errClientWritingFailed        = errors.New("could not write to the client")
	errConvertToFlusherFailed     = errors.New("could not instantiate http sse flusher")
)

const (
	replaySseStateArg = "replay"
	sseChannelArg     = "channel"

	// ObserverAdded informs about new observer being added to the SSE server.
	ObserverAdded ObserverChangeVariant = "observer-added"

	// ObserverRemoved informs about new observer being removed from the SSE server.
	ObserverRemoved ObserverChangeVariant = "observer-removed"
)

// ObserverChangeVariant specifies what change to the state the specified observers change specifies (addition, removal, etc.).
type ObserverChangeVariant string

// ObserversChange informs about changes to the SSE observers list.
type ObserversChange struct {
	ChangeVariant  ObserverChangeVariant
	RemoteAddr     string
	ChannelVariant state.SSEChannelVariant
}

type getSseHandler = func(res http.ResponseWriter, req *http.Request)

// handlerConfig is used to control creation of SSE handler for Server
type handlerConfig struct {
	Channels map[state.SSEChannelVariant]channel
}

func (s *Server) createSseRegisterHandler(cfg handlerConfig) getSseHandler {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			res.WriteHeader(404)
			return
		}

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
	sseChannel.AddObserver(remoteAddr)

	if s.observersChanges != nil {
		s.observersChanges <- ObserversChange{
			ChangeVariant:  ObserverAdded,
			RemoteAddr:     remoteAddr,
			ChannelVariant: sseChannel.Variant(),
		}
	}
	s.outLog.Printf("added %s observer with addr %s\n", sseChannel.Variant(), remoteAddr)

	if replaySseState(req) {
		err := sseChannel.Replay(res)
		if err != nil {
			s.errLog.Println(fmt.Sprintf("could not replay data on sse: %s", err.Error()))
		}
	}

	connectionDone := make(chan bool)
	channelDone := make(chan bool)
	channelErrors := make(chan error)
	go s.waitForConnectionClosure(req, connectionDone, sseChannel)
	go sseChannel.ServeObserver(remoteAddr, res, channelDone, channelErrors)

	for {
		select {
		case err := <-channelErrors:
			s.errLog.Printf("error occured on channel '%s' for remote address %s: %s\n", sseChannel.Variant(), remoteAddr, err)
		case <-channelDone:
			s.outLog.Printf("sse observation on channel '%s' done for remote address %s due to changes channel being closed\n", sseChannel.Variant(), remoteAddr)

			return
		case <-s.ctx.Done():
			s.outLog.Printf("sse observation on channel '%s' done for remote address %s due to server being stopped\n", sseChannel.Variant(), remoteAddr)

			return
		case <-connectionDone:
			return
		}
	}
}

// waitForConnectionClosure handles waiting for the sse SSE connection closure, handling the mutex and observers management afterwards.
// It should be run in a separate goroutine in-case change to the list of sse observers triggered by dispatcher is not handled due to Context().Done()
// being (randomly) selected first - since after the disconnect the lock should be obtained for the list of observers, it may happen that
// the dispatcher already acquired this lock and will deadlock below code. Running this method in a goroutine ensures that dispatcher will manage to go through
// the loop of channel observers and unlock the mutex.
func (s *Server) waitForConnectionClosure(req *http.Request, done chan<- bool, sseChannel channel) {
	<-req.Context().Done()
	sseChannel.RemoveObserver(req.RemoteAddr)

	if s.observersChanges != nil {
		s.observersChanges <- ObserversChange{
			ChangeVariant:  ObserverRemoved,
			RemoteAddr:     req.RemoteAddr,
			ChannelVariant: sseChannel.Variant(),
		}
	}
	s.outLog.Printf("removing %s observer with addr %s\n", sseChannel.Variant(), req.RemoteAddr)

	done <- true
	close(done)
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
