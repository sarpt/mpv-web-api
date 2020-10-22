package api

import (
	"encoding/json"
	"net/http"
	"sync"
)

// SSEObserverVariant specifies type of observer (movies, playback, etc.)
type SSEObserverVariant string

// StatusChangeVariant specifies what type of change to server status occurs
type StatusChangeVariant string

const (
	statusObserverVariant SSEObserverVariant = "status"

	statusReplay          StatusChangeVariant = "replay"
	clientObserverAdded   StatusChangeVariant = "client-observer-added"
	clientObserverRemoved StatusChangeVariant = "client-observer-removed"
	mpvProcessChanged     StatusChangeVariant = "mpv-process-changed"
)

// StatusChange holds information about changes to the server misc status
type StatusChange struct {
	Variant StatusChangeVariant
	Status  Status
}

// Status holds information about server misc status
type Status struct {
	ObservingAddresses map[string][]SSEObserverVariant
	lock               *sync.RWMutex    `json:"-"`
	Changes            chan interface{} `json:"-"`
}

func (s *Status) addObservingAddress(remoteAddr string, observerVariant SSEObserverVariant) {
	var observers []SSEObserverVariant
	var ok bool

	s.lock.Lock()
	defer s.lock.Unlock()

	observers, ok = s.ObservingAddresses[remoteAddr]
	if !ok {
		observers = []SSEObserverVariant{}
	}

	s.ObservingAddresses[remoteAddr] = append(observers, observerVariant)
	s.Changes <- StatusChange{
		Variant: clientObserverAdded,
		Status:  *s,
	}
}

func (s *Status) removeObservingAddress(remoteAddr string, observerVariant SSEObserverVariant) {
	var observers []SSEObserverVariant
	var ok bool

	s.lock.Lock()
	defer s.lock.Unlock()

	observers, ok = s.ObservingAddresses[remoteAddr]
	if !ok {
		return
	}

	filteredObservers := []SSEObserverVariant{}
	for _, observer := range observers {
		if observer != observerVariant {
			filteredObservers = append(filteredObservers, observer)
		}
	}

	if len(filteredObservers) == 0 {
		delete(s.ObservingAddresses, remoteAddr)
	} else {
		s.ObservingAddresses[remoteAddr] = filteredObservers
	}

	s.Changes <- StatusChange{
		Variant: clientObserverRemoved,
		Status:  *s,
	}
}

func (s *Status) observingAddresses() map[string][]SSEObserverVariant {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.ObservingAddresses
}

func (s *Status) jsonMarshal() ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return json.Marshal(s)
}

// concurrent-safe copy
func (s *Status) safeCopy() Status {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return *s
}

func (s *Server) createStatusReplayHandler() sseReplayHandler {
	return func(res http.ResponseWriter, flusher http.Flusher) error {
		return sendStatus(statusReplay, s.status, res, flusher)
	}
}

func (s *Server) createStatusChangeHandler() sseChangeHandler {
	return func(res http.ResponseWriter, flusher http.Flusher, changes interface{}) error {
		statusChange, ok := changes.(StatusChange)
		if !ok {
			return errIncorrectChangesType
		}

		return sendStatus(statusChange.Variant, &statusChange.Status, res, flusher)
	}
}

func (s *Server) createGetSseStatusHandler() getSseHandler {
	cfg := SseHandlerConfig{
		Observers:     s.statusSSEObservers,
		ChangeHandler: s.createStatusChangeHandler(),
		ReplayHandler: s.createStatusReplayHandler(),
	}

	return s.createGetSseHandler(cfg)
}

func sendStatus(variant StatusChangeVariant, status *Status, res http.ResponseWriter, flusher http.Flusher) error {
	out, err := status.jsonMarshal()
	if err != nil {
		return errResponseJSONCreationFailed
	}

	_, err = res.Write(formatSseEvent(string(variant), out))
	if err != nil {
		return errClientWritingFailed
	}

	flusher.Flush()
	return nil
}
