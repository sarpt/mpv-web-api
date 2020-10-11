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
)

// SSEObservers represents client observers that are currently connected to this instance of api server
type SSEObservers struct {
	Items map[string]chan interface{}
	Lock  *sync.RWMutex
}

type getSseHandler = func(res http.ResponseWriter, req *http.Request)
type sseReplayHandler = func(res http.ResponseWriter, flusher http.Flusher) error
type sseChangeHandler = func(res http.ResponseWriter, flusher http.Flusher, change interface{}) error

// SseHandlerConfig is used to control creation of SSE handler for Server
type SseHandlerConfig struct {
	ObserverVariant StatusObserverVariant
	Observers       SSEObservers
	ChangeHandler   sseChangeHandler
	ReplayHandler   sseReplayHandler
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

		// Buffer of 1 in case connection is closed after playbackObservers fan-out dispatcher already acquired read lock (blocking the write lock).
		// The dispatcher will expect for the select below to receive the message but the Context().Done() already waits to acquire a write lock.
		// So the buffer of 1 ensures that one message will be buffered, dispatcher will not be blocked, and write lock will be obtained.
		// When the write lock is obtained to remove from the set, even if a new playback will be received, read lock will wait until Context().Done() finishes.
		changes := make(chan interface{}, 1)
		cfg.Observers.Lock.Lock()
		cfg.Observers.Items[req.RemoteAddr] = changes
		cfg.Observers.Lock.Unlock()

		s.addObservingAddressToStatus(req.RemoteAddr, cfg.ObserverVariant)
		s.outLog.Printf("added %s observer with addr %s\n", cfg.ObserverVariant, req.RemoteAddr)

		if replaySseState(req) {
			err := cfg.ReplayHandler(res, flusher)
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

				err := cfg.ChangeHandler(res, flusher, change)
				if err != nil {
					s.errLog.Println(err.Error())
				}
			case <-req.Context().Done():
				cfg.Observers.Lock.Lock()
				delete(cfg.Observers.Items, req.RemoteAddr)
				cfg.Observers.Lock.Unlock()

				s.removeObservingAddressFromStatus(req.RemoteAddr, cfg.ObserverVariant)
				s.outLog.Printf("removing %s observer with addr %s\n", cfg.ObserverVariant, req.RemoteAddr)

				return
			}
		}
	}
}
