package sse

import (
	"errors"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playlists"
	state_sse "github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
)

const (
	playlistsSSEChannelVariant state_sse.ChannelVariant = "playlists"

	playlistsReplay playlists.PlaylistsChangeVariant = "replay"
)

type playlistsChannel struct {
	playback  *playback.Storage
	playlists *playlists.Playlists
	lock      *sync.RWMutex
	observers map[string]chan playlists.PlaylistsChange
}

func newPlaylistsChannel(playbackStorage *playback.Storage, playlistsStorage *playlists.Playlists) *playlistsChannel {
	return &playlistsChannel{
		playback:  playbackStorage,
		playlists: playlistsStorage,
		observers: map[string]chan playlists.PlaylistsChange{},
		lock:      &sync.RWMutex{},
	}
}

func (pc *playlistsChannel) AddObserver(address string) {
	changes := make(chan playlists.PlaylistsChange)

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

func (pc *playlistsChannel) changeHandler(res ResponseWriter, change playlists.PlaylistsChange) error {
	if pc.playback.Stopped { // TODO: the changes are shot by state.Playback even after the mediaFilePath is cleared, as such it may be wasteful to push further changes through SSE. to think of a way to reduce number of those blank data calls after closing stopping playback
		return res.SendEmptyChange(pc.Variant(), string(change.Variant))
	}

	return res.SendChange(change.Playlist, pc.Variant(), string(change.Variant))
}

func (pc *playlistsChannel) BroadcastToChannelObservers(change playlists.PlaylistsChange) {
	pc.lock.RLock()
	defer pc.lock.RUnlock()

	for _, observer := range pc.observers {
		observer <- change
	}
}

func (pc playlistsChannel) Variant() state_sse.ChannelVariant {
	return playlistsSSEChannelVariant
}
