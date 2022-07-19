package status

import (
	"encoding/json"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/state/internal/common"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
)

type Subscriber = func(change Change)

// ChangeVariant specifies what type of change to server status occurs.
type ChangeVariant string

const (
	// ClientObserverAdded notifies about addition of new client observer.
	ClientObserverAdded ChangeVariant = "client-observer-added"

	// ClientObserverRemoved notifies about removal of connected client observer.
	ClientObserverRemoved ChangeVariant = "client-observer-removed"

	// MPVProcessChanged notifies about change of mpv process (due to restart, forced close, etc.).
	MPVProcessChanged ChangeVariant = "mpv-process-changed"
)

// storageJSON is a status information in JSON form.
type storageJSON struct {
	ObservingAddresses map[string][]sse.ChannelVariant `json:"ObservingAddresses"`
}

// Change holds information about changes to the server misc status.
type Change struct {
	Variant ChangeVariant
}

// Storage holds information about server misc status.
type Storage struct {
	broadcaster        *common.ChangesBroadcaster
	observingAddresses map[string][]sse.ChannelVariant
	lock               *sync.RWMutex
}

// NewStorage constructs Status state.
func NewStorage() *Storage {
	broadcaster := common.NewChangesBroadcaster()
	broadcaster.Broadcast()

	return &Storage{
		broadcaster:        broadcaster,
		observingAddresses: map[string][]sse.ChannelVariant{},
		lock:               &sync.RWMutex{},
	}
}

// ObservingAddresses returns a mapping of a remote address to the channel variants.
func (s *Storage) ObservingAddresses() map[string][]sse.ChannelVariant {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.observingAddresses
}

// AddObservingAddress adds remote address listening on specific channel variant to the status state.
func (s *Storage) AddObservingAddress(remoteAddr string, observerVariant sse.ChannelVariant) {
	var observers []sse.ChannelVariant
	var ok bool

	s.lock.Lock()
	observers, ok = s.observingAddresses[remoteAddr]
	if !ok {
		observers = []sse.ChannelVariant{}
	}

	s.observingAddresses[remoteAddr] = append(observers, observerVariant)
	s.lock.Unlock()

	s.broadcaster.Send(Change{
		Variant: ClientObserverAdded,
	})
}

// MarshalJSON satisfies json.Marshaller.
func (s *Storage) MarshalJSON() ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	sJSON := storageJSON{
		ObservingAddresses: s.observingAddresses,
	}
	return json.Marshal(&sJSON)
}

// RemoveObservingAddress removes remote address listening on specific channel variant from the state.
func (s *Storage) RemoveObservingAddress(remoteAddr string, observerVariant sse.ChannelVariant) {
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

	s.broadcaster.Send(Change{
		Variant: ClientObserverRemoved,
	})
}

func (p *Storage) Subscribe(sub Subscriber, onError func(err error)) {
	p.broadcaster.Subscribe(func(change interface{}) {
		statusChange, ok := change.(Change)
		if !ok {
			onError(common.ErrIncorrectChangesType)

			return
		}

		sub(statusChange)
	})
}
