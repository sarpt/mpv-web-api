package api

import (
	"encoding/json"
	"sync"
)

// StatusChangeVariant specifies what type of change to server status occurs
type StatusChangeVariant string

const (
	statusSSEChannelVariant SSEChannelVariant = "status"

	statusReplay          StatusChangeVariant = "replay"
	clientObserverAdded   StatusChangeVariant = "client-observer-added"
	clientObserverRemoved StatusChangeVariant = "client-observer-removed"
	mpvProcessChanged     StatusChangeVariant = "mpv-process-changed"
)

// statusJSON is a status information in JSON form
type statusJSON struct {
	ObservingAddresses map[string][]SSEChannelVariant `json:"ObservingAddresses"`
}

// StatusChange holds information about changes to the server misc status
type StatusChange struct {
	Variant StatusChangeVariant
}

// Status holds information about server misc status
type Status struct {
	observingAddresses map[string][]SSEChannelVariant
	lock               *sync.RWMutex
	Changes            chan interface{}
}

func (s *Status) addObservingAddress(remoteAddr string, observerVariant SSEChannelVariant) {
	var observers []SSEChannelVariant
	var ok bool

	s.lock.Lock()
	observers, ok = s.observingAddresses[remoteAddr]
	if !ok {
		observers = []SSEChannelVariant{}
	}

	s.observingAddresses[remoteAddr] = append(observers, observerVariant)
	s.lock.Unlock()

	s.Changes <- StatusChange{
		Variant: clientObserverAdded,
	}
}

func (s *Status) removeObservingAddress(remoteAddr string, observerVariant SSEChannelVariant) {
	var observers []SSEChannelVariant
	var ok bool

	s.lock.Lock()

	observers, ok = s.observingAddresses[remoteAddr]
	if !ok {
		return
	}

	filteredObservers := []SSEChannelVariant{}
	for _, observer := range observers {
		if observer != observerVariant {
			filteredObservers = append(filteredObservers, observer)
		}
	}

	if len(filteredObservers) == 0 {
		delete(s.observingAddresses, remoteAddr)
	} else {
		s.observingAddresses[remoteAddr] = filteredObservers
	}

	s.lock.Unlock()

	s.Changes <- StatusChange{
		Variant: clientObserverRemoved,
	}
}

// ObservingAddresses returns a mapping of a remote address to the channel variants
func (s *Status) ObservingAddresses() map[string][]SSEChannelVariant {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.observingAddresses
}

func (s *Status) jsonMarshal() ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	sJSON := statusJSON{
		ObservingAddresses: s.observingAddresses,
	}
	return json.Marshal(sJSON)
}
