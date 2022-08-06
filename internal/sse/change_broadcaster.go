package sse

import (
	"sync"

	"github.com/sarpt/mpv-web-api/internal/common"
)

type Change interface {
	Variant() common.ChangeVariant
	MarshalJSON() ([]byte, error)
}

type ChangesBroadcaster[CT Change] struct {
	lock      *sync.RWMutex
	observers map[string]chan CT
}

func NewChangesBroadcaster[CT Change]() ChangesBroadcaster[CT] {
	return ChangesBroadcaster[CT]{
		lock:      &sync.RWMutex{},
		observers: map[string]chan CT{},
	}
}

func (st *ChangesBroadcaster[CT]) AddObserver(address string) {
	changes := make(chan CT)

	st.lock.Lock()
	defer st.lock.Unlock()

	st.observers[address] = changes
}

func (st *ChangesBroadcaster[CT]) RemoveObserver(address string) {
	st.lock.Lock()
	defer st.lock.Unlock()

	changes, ok := st.observers[address]
	if !ok {
		return
	}

	close(changes)
	delete(st.observers, address)
}

func (st *ChangesBroadcaster[CT]) BroadcastToChannelObservers(change CT) {
	st.lock.RLock()
	defer st.lock.RUnlock()

	for _, observer := range st.observers {
		observer <- change
	}
}

func (st *ChangesBroadcaster[CT]) Observer(address string) (chan CT, bool) {
	observer, ok := st.observers[address]
	return observer, ok
}
