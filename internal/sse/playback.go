package sse

import (
	"errors"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"
	state_sse "github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
)

const (
	playbackSSEChannelVariant state_sse.ChannelVariant = "playback"

	playbackAllSseEvent    = "all"
	playbackReplaySseEvent = "replay"
)

type playbackChannel struct {
	playback  *playback.Playback
	lock      *sync.RWMutex
	observers map[string]chan playback.PlaybackChange
}

func newPlaybackChannel(playbackStorage *playback.Playback) *playbackChannel {
	return &playbackChannel{
		playback:  playbackStorage,
		observers: map[string]chan playback.PlaybackChange{},
		lock:      &sync.RWMutex{},
	}
}

func (pc *playbackChannel) AddObserver(address string) {
	changes := make(chan playback.PlaybackChange)

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

func (pc *playbackChannel) changeHandler(res ResponseWriter, change playback.PlaybackChange) error {
	if pc.playback.Stopped { // TODO: the changes are shot by state.Playback even after the mediaFilePath is cleared, as such it may be wasteful to push further changes through SSE. to think of a way to reduce number of those blank data calls after closing stopping playback
		return res.SendEmptyChange(pc.Variant(), string(change.Variant))
	}

	return res.SendChange(pc.playback, pc.Variant(), string(change.Variant))
}

func (pc *playbackChannel) BroadcastToChannelObservers(change playback.PlaybackChange) {
	pc.lock.RLock()
	defer pc.lock.RUnlock()

	for _, observer := range pc.observers {
		observer <- change
	}
}

func (pc playbackChannel) Variant() state_sse.ChannelVariant {
	return playbackSSEChannelVariant
}
