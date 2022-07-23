package sse

import (
	"errors"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
	state_sse "github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
)

type Change interface {
	Variant() sse.ChangeVariant
	MarshalJSON() ([]byte, error)
}

type StateChannel[ST any, CT Change] struct {
	state     ST
	lock      *sync.RWMutex
	observers map[string]chan CT
	variant   sse.ChannelVariant
}

func NewStateChannel[ST any, CT Change](state ST, variant state_sse.ChannelVariant) StateChannel[ST, CT] {
	return StateChannel[ST, CT]{
		state:     state,
		lock:      &sync.RWMutex{},
		observers: map[string]chan CT{},
		variant:   variant,
	}
}

func (st *StateChannel[ST, CT]) AddObserver(address string) {
	changes := make(chan CT)

	st.lock.Lock()
	defer st.lock.Unlock()

	st.observers[address] = changes
}

func (st *StateChannel[ST, CT]) RemoveObserver(address string) {
	st.lock.Lock()
	defer st.lock.Unlock()

	changes, ok := st.observers[address]
	if !ok {
		return
	}

	close(changes)
	delete(st.observers, address)
}

func (st *StateChannel[ST, CT]) ServeObserver(address string, res ResponseWriter, done chan<- bool, errs chan<- error) {
	defer close(done)
	defer close(errs)

	changes, ok := st.observers[address]
	if !ok {
		errs <- errors.New("no observer found for provided address")
		done <- true

		return
	}

	for {
		change, more := <-changes
		if !more {
			done <- true

			return
		}

		err := st.changeHandler(res, change)
		if err != nil {
			errs <- err
		}
	}
}

func (st *StateChannel[ST, CT]) changeHandler(res ResponseWriter, change CT) error {
	return res.SendChange(change, st.Variant(), string(change.Variant()))
}

func (st *StateChannel[ST, CT]) BroadcastToChannelObservers(change CT) {
	st.lock.RLock()
	defer st.lock.RUnlock()

	for _, observer := range st.observers {
		observer <- change
	}
}

func (st *StateChannel[ST, CT]) Variant() sse.ChannelVariant {
	return st.variant
}
