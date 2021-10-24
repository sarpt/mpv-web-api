package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
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
	PlaylistsAdded PlaylistsChangeVariant = "added"

	// PlaylistsReplay notifies about replay of a whole playlist state.
	PlaylistsReplay PlaylistsChangeVariant = "replay"

	// PlaylistsItemsChange notifies about changes to the items/entries in a playlist.
	PlaylistsItemsChange PlaylistsChangeVariant = "itemsChange"
)

var (
	ErrPlaylistWithUUIDAlreadyExists = errors.New("playlist with UUID already exists")
	ErrPlaylistWithUUIDDoesNotExist  = errors.New("playlist with provided uuid does not exist")
)

// PlaylistsChange is used to inform about changes to the Playback.
type PlaylistsChange struct {
	Variant PlaylistsChangeVariant
}

func NewPlaylists() *Playlists {
	return &Playlists{
		changes: make(chan interface{}),
		items:   map[string]*Playlist{},
		lock:    &sync.RWMutex{},
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

func (p *Playlists) ByUUID(uuid string) (*Playlist, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	playlist, ok := p.items[uuid]
	if !ok {
		return &Playlist{}, ErrPlaylistWithUUIDDoesNotExist
	}

	return playlist, nil
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

// SetPlaylistCurrentEntryIdx sets currently played entry Idx for a playlist with provided UUID.
func (p *Playlists) SetPlaylistCurrentEntryIdx(uuid string, idx int) error {
	playlist, err := p.ByUUID(uuid)
	if err != nil {
		return err
	}

	playlist.setCurrentEntryIdx(idx)

	p.changes <- PlaylistsChange{
		Variant: PlaylistsItemsChange,
	}
	return nil
}

// SetPlaylistEntries sets items of the playlist with uuid.
func (p *Playlists) SetPlaylistEntries(uuid string, entries []PlaylistEntry) error {
	playlist, err := p.ByUUID(uuid)
	if err != nil {
		return err
	}

	playlist.setEntries(entries)

	p.changes <- PlaylistsChange{
		Variant: PlaylistsItemsChange,
	}
	return nil
}

// AddPlaylist sets items of the playlist with uuid.
// Returns UUID of new playlist.
// Note: AddPlaylist is responsible for creating an UUID since users of the
// functions should not be trusted that playlist has a valid, if any, UUID.
func (p *Playlists) AddPlaylist(playlist *Playlist) (string, error) {
	if playlist.uuid == "" {
		uuid := uuid.NewString()
		playlist.uuid = uuid
	}

	if _, ok := p.items[playlist.uuid]; ok {
		return playlist.uuid, fmt.Errorf("%w: %s", ErrPlaylistWithUUIDAlreadyExists, playlist.uuid)
	}

	p.lock.Lock()
	p.items[playlist.uuid] = playlist
	p.lock.Unlock()

	p.changes <- PlaylistsChange{
		Variant: PlaylistsAdded,
	}

	return playlist.uuid, nil
}
