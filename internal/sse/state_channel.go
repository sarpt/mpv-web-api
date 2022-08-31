package sse

import (
	"errors"

	state_sse "github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
)

type stateChangeBroadcaster[CT Change] interface {
	AddObserver(address string)
	RemoveObserver(address string)
	Replay(res ResponseWriter) error
	ChangeHandler(res ResponseWriter, change CT) error
	Observer(address string) (chan CT, bool)
	BroadcastToChannelObservers(change CT, unsub func())
}

type StateChannel[CT Change] struct {
	stateChangeBroadcaster[CT]
	variant state_sse.ChannelVariant
}

func (sc *StateChannel[CT]) ServeObserver(address string, res ResponseWriter, done chan<- bool, errs chan<- error) {
	defer close(done)
	defer close(errs)

	changes, ok := sc.Observer(address)
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

		err := sc.ChangeHandler(res, change)
		if err != nil {
			errs <- err
		}
	}
}

func (sc *StateChannel[CT]) Variant() state_sse.ChannelVariant {
	return sc.variant
}
