package sse

import (
	"errors"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/state"
)

const (
	playlistsSSEChannelVariant state.SSEChannelVariant = "playlists"

	playlistsReplay state.PlaylistsChangeVariant = "replay"
)

type playlistsChannel struct {
	playback  *state.Playback
	playlists *state.Playlists
	lock      *sync.RWMutex
	observers map[string]chan state.PlaylistsChange
}

func newPlaylistsChannel(playback *state.Playback, playlists *state.Playlists) *playlistsChannel {
	return &playlistsChannel{
		playback:  playback,
		playlists: playlists,
		observers: map[string]chan state.PlaylistsChange{},
		lock:      &sync.RWMutex{},
	}
}

func (pc *playlistsChannel) AddObserver(address string) {
	changes := make(chan state.PlaylistsChange)

	pc.lock.Lock()
	defer pc.lock.Unlock()

	pc.observers[address] = changes
}

func (pc *playlistsChannel) RemoveObserver(address string) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	changes, ok := pc.observers[address]
	if !ok {
		return
	}

	close(changes)
	delete(pc.observers, address)
}

func (pc *playlistsChannel) Replay(res ResponseWriter) error {
	return res.SendChange(pc.playlists, pc.Variant(), string(playlistsReplay))
}

func (pc *playlistsChannel) ServeObserver(address string, res ResponseWriter, done chan<- bool, errs chan<- error) {
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

func (pc *playlistsChannel) changeHandler(res ResponseWriter, change state.PlaylistsChange) error {
	if pc.playback.Stopped { // TODO: the changes are shot by state.Playback even after the mediaFilePath is cleared, as such it may be wasteful to push further changes through SSE. to think of a way to reduce number of those blank data calls after closing stopping playback
		return res.SendEmptyChange(pc.Variant(), string(change.Variant))
	}

	return res.SendChange(change.Playlist, pc.Variant(), string(change.Variant))
}

func (pc *playlistsChannel) BroadcastToChannelObservers(change state.PlaylistsChange) {
	pc.lock.RLock()
	defer pc.lock.RUnlock()

	for _, observer := range pc.observers {
		observer <- change
	}
}

func (pc playlistsChannel) Variant() state.SSEChannelVariant {
	return playlistsSSEChannelVariant
}
