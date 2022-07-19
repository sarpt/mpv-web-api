package status

import (
	"encoding/json"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/state/internal/common"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
)

type StatusSubscriber = func(change StatusChange)

// StatusChangeVariant specifies what type of change to server status occurs.
type StatusChangeVariant string

const (
	// ClientObserverAdded notifies about addition of new client observer.
	ClientObserverAdded StatusChangeVariant = "client-observer-added"

	// ClientObserverRemoved notifies about removal of connected client observer.
	ClientObserverRemoved StatusChangeVariant = "client-observer-removed"

	// MPVProcessChanged notifies about change of mpv process (due to restart, forced close, etc.).
	MPVProcessChanged StatusChangeVariant = "mpv-process-changed"
)

// statusJSON is a status information in JSON form.
type statusJSON struct {
	ObservingAddresses map[string][]sse.ChannelVariant `json:"ObservingAddresses"`
}

// StatusChange holds information about changes to the server misc status.
type StatusChange struct {
	Variant StatusChangeVariant
}

// Status holds information about server misc status.
type Status struct {
	broadcaster        *common.ChangesBroadcaster
	observingAddresses map[string][]sse.ChannelVariant
	lock               *sync.RWMutex
}

// NewStatus constructs Status state.
func NewStatus() *Status {
	broadcaster := common.NewChangesBroadcaster()
	broadcaster.Broadcast()

	return &Status{
		broadcaster:        broadcaster,
		observingAddresses: map[string][]sse.ChannelVariant{},
		lock:               &sync.RWMutex{},
	}
}

// ObservingAddresses returns a mapping of a remote address to the channel variants.
func (s *Status) ObservingAddresses() map[string][]sse.ChannelVariant {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.observingAddresses
}

// AddObservingAddress adds remote address listening on specific channel variant to the status state.
func (s *Status) AddObservingAddress(remoteAddr string, observerVariant sse.ChannelVariant) {
	var observers []sse.ChannelVariant
	var ok bool

	s.lock.Lock()
	observers, ok = s.observingAddresses[remoteAddr]
	if !ok {
		observers = []sse.ChannelVariant{}
	}

	s.observingAddresses[remoteAddr] = append(observers, observerVariant)
	s.lock.Unlock()

	s.broadcaster.Send(StatusChange{
		Variant: ClientObserverAdded,
	})
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

// RemoveObservingAddress removes remote address listening on specific channel variant from the state.
func (s *Status) RemoveObservingAddress(remoteAddr string, observerVariant sse.ChannelVariant) {
	var observers []sse.ChannelVariant
	var ok bool

	s.lock.Lock()

	observers, ok = s.observingAddresses[remoteAddr]
	if !ok {
		return
	}

	filteredObservers := []sse.ChannelVariant{}
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

	s.broadcaster.Send(StatusChange{
		Variant: ClientObserverRemoved,
	})
}

func (p *Status) Subscribe(sub StatusSubscriber, onError func(err error)) {
	p.broadcaster.Subscribe(func(change interface{}) {
		statusChange, ok := change.(StatusChange)
		if !ok {
			onError(common.ErrIncorrectChangesType)

			return
		}

		sub(statusChange)
	})
}
