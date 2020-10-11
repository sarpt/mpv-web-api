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

func (s *Status) getObservingAddresses() map[string][]StatusObserverVariant {
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

func (s *Server) getSseStatusHandler(res http.ResponseWriter, req *http.Request) {
	flusher, err := sseFlusher(res)
	if err != nil {
		res.WriteHeader(400)
		return
	}

	statusChanges := make(chan StatusChange, 1)
	s.addStatusChangeObserver(req.RemoteAddr, statusChanges)

	if replaySseState(req) {
		err := sendStatus(statusReplay, s.status, res, flusher)

		if err != nil {
			s.errLog.Println(err.Error())
		}
	}

	for {
		select {
		case change, ok := <-statusChanges:
			if !ok {
				return
			}

			err := sendStatus(change.Variant, &change.Status, res, flusher)
			if err != nil {
				s.errLog.Println(err.Error())
			}
		case <-req.Context().Done():
			s.removeStatusChangeObserver(req.RemoteAddr)
			return
		}
	}
}

func (s *Server) addStatusChangeObserver(remoteAddr string, changes chan StatusChange) {
	s.statusChangesObserversLock.Lock()
	s.statusChangesObservers[remoteAddr] = changes
	s.statusChangesObserversLock.Unlock()

	s.addObservingAddressToStatus(remoteAddr, statusObserverVariant)
	s.outLog.Printf("added /sse/status observer with addr %s\n", remoteAddr)
}

func (s *Server) removeStatusChangeObserver(remoteAddr string) {
	s.moviesChangesObserversLock.Lock()
	delete(s.moviesChangesObservers, remoteAddr)
	s.moviesChangesObserversLock.Unlock()

	s.removeObservingAddressFromStatus(remoteAddr, statusObserverVariant)
	s.outLog.Printf("removing /sse/movies observer with addr %s\n", remoteAddr)
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

		s.statusChangesObserversLock.RLock()
		for _, observer := range s.statusChangesObservers {
			observer <- changes
		}
		s.statusChangesObserversLock.RUnlock()
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
