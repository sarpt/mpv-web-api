package state

import (
	"encoding/json"
	"sync"
)

// StatusChangeVariant specifies what type of change to server status occurs.
type StatusChangeVariant string

// SSEChannelVariant specifies type of observer (mediaFiles, playback, etc.)
type SSEChannelVariant string

const (
	// StatusReplay notifies about replay of status state.
	StatusReplay StatusChangeVariant = "replay"

	// ClientObserverAdded notifies about addition of new client observer.
	ClientObserverAdded StatusChangeVariant = "client-observer-added"

	// ClientObserverRemoved notifies about removal of connected client observer.
	ClientObserverRemoved StatusChangeVariant = "client-observer-removed"

	// MPVProcessChanged notifies about change of mpv process (due to restart, forced close, etc.).
	MPVProcessChanged StatusChangeVariant = "mpv-process-changed"
)

// statusJSON is a status information in JSON form.
type statusJSON struct {
	ObservingAddresses map[string][]SSEChannelVariant `json:"ObservingAddresses"`
}

// StatusChange holds information about changes to the server misc status.
type StatusChange struct {
	Variant StatusChangeVariant
}

// Status holds information about server misc status.
type Status struct {
	observingAddresses map[string][]SSEChannelVariant
	lock               *sync.RWMutex
	changes            chan interface{}
}

// NewStatus constructs Status state.
func NewStatus() *Status {
	return &Status{
		observingAddresses: map[string][]SSEChannelVariant{},
		lock:               &sync.RWMutex{},
		changes:            make(chan interface{}),
	}
}

// Changes returns read-only channel notifying of status changes
func (s *Status) Changes() <-chan interface{} {
	return s.changes
}

// ObservingAddresses returns a mapping of a remote address to the channel variants.
func (s *Status) ObservingAddresses() map[string][]SSEChannelVariant {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.observingAddresses
}

// AddObservingAddress adds remote address listening on specific channel variant to the status state.
func (s *Status) AddObservingAddress(remoteAddr string, observerVariant SSEChannelVariant) {
	var observers []SSEChannelVariant
	var ok bool

	s.lock.Lock()
	observers, ok = s.observingAddresses[remoteAddr]
	if !ok {
		observers = []SSEChannelVariant{}
	}

	s.observingAddresses[remoteAddr] = append(observers, observerVariant)
	s.lock.Unlock()

	s.changes <- StatusChange{
		Variant: ClientObserverAdded,
	}
}

// RemoveObservingAddress removes remote address listening on specific channel variant from the state.
func (s *Status) RemoveObservingAddress(remoteAddr string, observerVariant SSEChannelVariant) {
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

	s.changes <- StatusChange{
		Variant: ClientObserverRemoved,
	}
}

// MarshalJSON satisfies json.Marshaller.
func (s *Status) MarshalJSON() ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	sJSON := statusJSON{
		ObservingAddresses: s.observingAddresses,
	}
	return json.Marshal(&sJSON)
}
