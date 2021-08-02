package state

import (
	"encoding/json"
	"fmt"
	"sync"
)

type Playlists struct {
	changes chan interface{}
	items   map[string]*Playlist
	lock    *sync.RWMutex
}

type playlistsJSON struct {
	Items map[string]*Playlist `json:"Items"`
}

// PlaybackChangeVariant specifies type of change that happened to a playlist.
type PlaylistsChangeVariant string

const (
	// PlaylistsAdded notifies of a new playlist being served.
	PlaylistsAdded PlaylistsChangeVariant = "playlistAdded" // TODO: playlist prefix redundant

	// PlaylistsReplay notifies about replay of a whole playlist state.
	PlaylistsReplay PlaylistsChangeVariant = "replay"

	// PlaylistsItemsChange notifies about changes to the items/entries in a playlist.
	PlaylistsItemsChange PlaylistsChangeVariant = "playlistItemsChange" // TODO: playlist prefix redundant
)

// PlaylistsChange is used to inform about changes to the Playback.
type PlaylistsChange struct {
	Variant PlaylistsChangeVariant
}

func NewPlaylists() *Playlists {
	defaultPlaylist := NewPlaylist(PlaylistConfig{})

	return &Playlists{
		changes: make(chan interface{}),
		items: map[string]*Playlist{
			defaultPlaylist.uuid: defaultPlaylist,
		},
		lock: &sync.RWMutex{},
	}
}

// All returns a copy of all Playlists being served by the instance of the server.
func (p *Playlists) All() map[string]*Playlist {
	allPlaylists := map[string]*Playlist{}

	p.lock.RLock()
	defer p.lock.RUnlock()

	for uuid, playlist := range p.items {
		allPlaylists[uuid] = playlist
	}

	return allPlaylists
}

func (p *Playlists) Changes() <-chan interface{} {
	return p.changes
}

// MarshalJSON satisifes json.Marshaller.
func (p *Playlists) MarshalJSON() ([]byte, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	pJSON := playlistsJSON{
		Items: p.items,
	}
	return json.Marshal(pJSON)
}

// SetPlaylistItems sets items of the playlist with uuid.
func (p *Playlists) SetPlaylistItems(uuid string, items []string) error {
	p.lock.RLock()
	playlist, ok := p.items[uuid]
	p.lock.RUnlock()
	if !ok {
		return fmt.Errorf("could not set items for a playlist with uuid '%s': no such uuid exist", uuid)
	}

	p.lock.Lock()
	playlist.items = items
	p.lock.Unlock()

	p.changes <- PlaylistsChange{
		Variant: PlaylistsItemsChange,
	}
	return nil
}

// SetPlaylistItems sets items of the playlist with uuid.
func (p *Playlists) AddPlaylist(playlist *Playlist) error {
	p.lock.Lock()
	p.items[playlist.UUID()] = playlist
	p.lock.Unlock()

	p.changes <- PlaylistsChange{
		Variant: PlaylistsAdded,
	}
	return nil
}
