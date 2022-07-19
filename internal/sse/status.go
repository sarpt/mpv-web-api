package sse

import (
	"errors"
	"sync"

	state_sse "github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/status"
)

const (
	statusSSEChannelVariant state_sse.ChannelVariant = "status"

	// statusReplay notifies about replay of status state.
	statusReplay status.ChangeVariant = "replay"
)

type statusChannel struct {
	status    *status.Storage
	lock      *sync.RWMutex
	observers map[string]chan status.Change
}

func newStatusChannel(statusStorage *status.Storage) *statusChannel {
	return &statusChannel{
		status:    statusStorage,
		observers: map[string]chan status.Change{},
		lock:      &sync.RWMutex{},
	}
}

func (sc *statusChannel) AddObserver(address string) {
	changes := make(chan status.Change)

	sc.lock.Lock()
	defer sc.lock.Unlock()

	sc.observers[address] = changes
}

func (sc *statusChannel) RemoveObserver(address string) {
	sc.lock.Lock()
	defer sc.lock.Unlock()

	changes, ok := sc.observers[address]
	if !ok {
		return
	}

	close(changes)
	delete(sc.observers, address)
}

func (sc *statusChannel) Replay(res ResponseWriter) error {
	return res.SendChange(sc.status, sc.Variant(), string(statusReplay))
}

func (sc *statusChannel) ServeObserver(address string, res ResponseWriter, done chan<- bool, errs chan<- error) {
	defer close(done)
	defer close(errs)

	changes, ok := sc.observers[address]
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

		err := sc.changeHandler(res, change)
		if err != nil {
			errs <- err
		}
	}
}

func (sc *statusChannel) changeHandler(res ResponseWriter, change status.Change) error {
	return res.SendChange(sc.status, sc.Variant(), string(change.Variant))
}

func (sc *statusChannel) BroadcastToChannelObservers(change status.Change) {
	sc.lock.RLock()
	defer sc.lock.RUnlock()

	for _, observer := range sc.observers {
		observer <- change
	}
}

func (sc statusChannel) Variant() state_sse.ChannelVariant {
	return statusSSEChannelVariant
}
