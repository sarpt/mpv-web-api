package sse

import (
	"errors"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/state"
)

const (
	playbackSSEChannelVariant state.SSEChannelVariant = "playback"

	playbackAllSseEvent    = "all"
	playbackReplaySseEvent = "replay"
)

type playbackChannel struct {
	playback  *state.Playback
	lock      *sync.RWMutex
	observers map[string]chan state.PlaybackChange
}

func newPlaybackChannel(playback *state.Playback) *playbackChannel {
	return &playbackChannel{
		playback:  playback,
		observers: map[string]chan state.PlaybackChange{},
		lock:      &sync.RWMutex{},
	}
}

func (pc *playbackChannel) AddObserver(address string) {
	changes := make(chan state.PlaybackChange)

	pc.lock.Lock()
	defer pc.lock.Unlock()

	pc.observers[address] = changes
}

func (pc *playbackChannel) RemoveObserver(address string) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	changes, ok := pc.observers[address]
	if !ok {
		return
	}

	close(changes)
	delete(pc.observers, address)
}

func (pc *playbackChannel) Replay(res ResponseWriter) error {
	return res.SendChange(pc.playback, pc.Variant(), playbackReplaySseEvent)
}

func (pc *playbackChannel) ServeObserver(address string, res ResponseWriter, done chan<- bool, errs chan<- error) {
	defer close(done)
	defer close(errs)

	changes, ok := pc.observers[address]
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

		err := pc.changeHandler(res, change)
		if err != nil {
			errs <- err
		}
	}
}

func (pc *playbackChannel) changeHandler(res ResponseWriter, change state.PlaybackChange) error {
	if pc.playback.Stopped { // TODO: the changes are shot by state.Playback even after the mediaFilePath is cleared, as such it may be wasteful to push further changes through SSE. to think of a way to reduce number of those blank data calls after closing stopping playback
		return res.SendEmptyChange(pc.Variant(), string(change.Variant))
	}

	return res.SendChange(pc.playback, pc.Variant(), string(change.Variant))
}

func (pc *playbackChannel) BroadcastToChannelObservers(change state.PlaybackChange) {
	pc.lock.RLock()
	defer pc.lock.RUnlock()

	for _, observer := range pc.observers {
		observer <- change
	}
}

func (pc playbackChannel) Variant() state.SSEChannelVariant {
	return playbackSSEChannelVariant
}
