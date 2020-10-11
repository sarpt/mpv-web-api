package api

import (
	"encoding/json"
	"net/http"
	"sync"
)

// StatusObserverVariant specifies type of observer (movies, playback, etc.)
type StatusObserverVariant string

// StatusChangeVariant specifies what type of change to server status occurs
type StatusChangeVariant string

const (
	statusObserverVariant StatusObserverVariant = "status"

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
	ObservingAddresses map[string][]StatusObserverVariant
	lock               *sync.RWMutex
}

func (s *Status) addObservingAddress(remoteAddr string, observerVariant StatusObserverVariant) {
	var observers []StatusObserverVariant
	var ok bool

	s.lock.Lock()
	defer s.lock.Unlock()

	observers, ok = s.ObservingAddresses[remoteAddr]
	if !ok {
		observers = []StatusObserverVariant{}
	}

	s.ObservingAddresses[remoteAddr] = append(observers, observerVariant)
}

func (s *Status) removeObservingAddress(remoteAddr string, observerVariant StatusObserverVariant) {
	var observers []StatusObserverVariant
	var ok bool

	s.lock.Lock()
	defer s.lock.Unlock()

	observers, ok = s.ObservingAddresses[remoteAddr]
	if !ok {
		return
	}

	filteredObservers := []StatusObserverVariant{}
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
}

func (s *Status) observingAddresses() map[string][]StatusObserverVariant {
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
		ObserverVariant: statusObserverVariant,
		Observers:       s.statusChangesObservers,
		ChangeHandler:   s.createStatusChangeHandler(),
		ReplayHandler:   s.createStatusReplayHandler(),
	}

	return s.createGetSseHandler(cfg)
}

func (s Server) addObservingAddressToStatus(remoteAddr string, observerVariant StatusObserverVariant) {
	s.status.addObservingAddress(remoteAddr, observerVariant)
	s.statusChanges <- StatusChange{
		Variant: clientObserverAdded,
		Status:  s.status.safeCopy(),
	}
}

func (s Server) removeObservingAddressFromStatus(remoteAddr string, observerVariant StatusObserverVariant) {
	s.status.removeObservingAddress(remoteAddr, observerVariant)
	s.statusChanges <- StatusChange{
		Variant: clientObserverRemoved,
		Status:  s.status.safeCopy(),
	}
}

// watchStatusChanges reads all statusChanges done by path/event handlers.
func (s Server) watchStatusChanges() {
	for {
		changes, ok := <-s.statusChanges
		if !ok {
			return
		}

		s.statusChangesObservers.Lock.RLock()
		for _, observer := range s.statusChangesObservers.Items {
			observer <- changes
		}
		s.statusChangesObservers.Lock.RUnlock()
	}
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
