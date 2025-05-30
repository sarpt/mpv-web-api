package playlists

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/sarpt/mpv-web-api/internal/common"
	"github.com/sarpt/mpv-web-api/pkg/state/internal/revision"
)

type SubscriberCB = func(change Change)

type playlistChangeSubscriber struct {
	cb SubscriberCB
}

func (s *playlistChangeSubscriber) Receive(change Change) {
	s.cb(change)
}

type Storage struct {
	broadcaster *common.ChangesBroadcaster[Change]
	items       map[string]*Playlist
	lock        *sync.RWMutex
	revision    *revision.Storage
}

const (
	// PlaylistsAdded notifies of a new playlist being served.
	PlaylistsAdded common.ChangeVariant = "added"

	// PlaylistsCurrentEntryIdxChange notifies about change to the most current idx
	// (not neccessarily currently played by the mpv, but most recent idx in the scope of this playlist).
	PlaylistsCurrentEntryIdxChange common.ChangeVariant = "currentEntryIdxChange"

	// PlaylistsEntriesChange notifies about changes to the entries in a playlist.
	PlaylistsEntriesChange common.ChangeVariant = "entriesChange"
)

var (
	ErrPlaylistWithUUIDAlreadyExists = errors.New("playlist with UUID already exists")
	ErrPlaylistWithUUIDDoesNotExist  = errors.New("playlist with provided uuid does not exist")
)

// Change is used to inform about changes to the Playback.
type Change struct {
	ChangeVariant common.ChangeVariant
	Playlist      *Playlist
}

// MarshalJSON returns change items in JSON format. Satisfies json.Marshaller.
func (s Change) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Playlist)
}

func (s Change) Variant() common.ChangeVariant {
	return s.ChangeVariant
}

func NewStorage(broadcaster *common.ChangesBroadcaster[Change]) *Storage {
	return &Storage{
		broadcaster: broadcaster,
		items:       map[string]*Playlist{},
		lock:        &sync.RWMutex{},
		revision:    revision.NewStorage(),
	}
}

// All returns a copy of all Playlists being served by the instance of the server.
func (p *Storage) All() map[string]*Playlist {
	allPlaylists := map[string]*Playlist{}

	p.lock.RLock()
	defer p.lock.RUnlock()

	for uuid, playlist := range p.items {
		allPlaylists[uuid] = playlist
	}

	return allPlaylists
}

func (p *Storage) ByUUID(uuid string) (*Playlist, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	playlist, ok := p.items[uuid]
	if !ok {
		return &Playlist{}, ErrPlaylistWithUUIDDoesNotExist
	}

	return playlist, nil
}

// MarshalJSON satisifes json.Marshaller.
func (p *Storage) MarshalJSON() ([]byte, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return json.Marshal(p.items)
}

func (p *Storage) Revision() revision.Identifier {
	return p.revision.Revision()
}

// SetPlaylistCurrentEntryIdx sets currently played entry Idx for a playlist with provided UUID.
func (p *Storage) SetPlaylistCurrentEntryIdx(uuid string, idx int) error {
	playlist, err := p.ByUUID(uuid)
	if err != nil {
		return err
	}

	playlist.setCurrentEntryIdx(idx)

	p.revision.Tick()
	p.broadcaster.Send(Change{
		ChangeVariant: PlaylistsCurrentEntryIdxChange,
		Playlist:      playlist,
	})
	return nil
}

// SetPlaylistEntries sets items of the playlist with uuid.
func (p *Storage) SetPlaylistEntries(uuid string, entries []Entry) error {
	playlist, err := p.ByUUID(uuid)
	if err != nil {
		return err
	}

	playlist.setEntries(entries)

	p.revision.Tick()
	p.broadcaster.Send(Change{
		ChangeVariant: PlaylistsEntriesChange,
		Playlist:      playlist,
	})
	return nil
}

func (p *Storage) Subscribe(cb SubscriberCB, onError func(err error)) func() {
	subscriber := playlistChangeSubscriber{
		cb,
	}

	return p.broadcaster.Subscribe(&subscriber)
}

// AddPlaylist sets items of the playlist with uuid.
// Returns UUID of new playlist.
// Note: AddPlaylist is responsible for creating an UUID since users of the
// functions should not be trusted that playlist has a valid, if any, UUID.
func (p *Storage) AddPlaylist(playlist *Playlist) (string, error) {
	playlist.uuid = uuid.NewString()

	if _, ok := p.items[playlist.uuid]; ok {
		return playlist.uuid, fmt.Errorf("%w: %s", ErrPlaylistWithUUIDAlreadyExists, playlist.uuid)
	}

	p.lock.Lock()
	p.items[playlist.uuid] = playlist
	p.lock.Unlock()

	p.revision.Tick()
	p.broadcaster.Send(Change{
		ChangeVariant: PlaylistsAdded,
		Playlist:      playlist,
	})

	return playlist.uuid, nil
}
