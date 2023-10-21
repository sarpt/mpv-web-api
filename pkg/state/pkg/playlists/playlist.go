package playlists

import (
	"encoding/json"
	"slices"
	"sync"

	"github.com/google/uuid"
)

type PlaylistOrigin string

const (
	ExternalOrigin PlaylistOrigin = "externalOrigin"
	CachedOrigin   PlaylistOrigin = "cachedOrigin"
	TempOrigin     PlaylistOrigin = "tempOrigin"
)

// Playlist holds state about currently playing playlist.
type Playlist struct {
	entryIdx                   int
	description                string
	directoryContentsAsEntries bool
	entries                    []Entry
	name                       string
	lock                       *sync.RWMutex
	path                       string
	origin                     PlaylistOrigin
	uuid                       string
}

type playlistJSON struct {
	CurrentEntryIdx            int     `json:"CurrentEntryIdx"`
	Description                string  `json:"Description"`
	DirectoryContentsAsEntries bool    `json:"DirectoryContentsAsEntries"`
	Entries                    []Entry `json:"Entries"`
	Name                       string  `json:"Name"`
	Path                       string  `json:"Path"`
	UUID                       string  `json:"UUID"`
}

type Config struct {
	CurrentEntryIdx            int
	Description                string
	DirectoryContentsAsEntries bool
	Entries                    []Entry
	Name                       string
	Origin                     PlaylistOrigin
	Path                       string
}

// NewPlaylist constructs Playlist state.
func NewPlaylist(cfg Config) *Playlist {
	return &Playlist{
		entryIdx:                   cfg.CurrentEntryIdx,
		description:                cfg.Description,
		directoryContentsAsEntries: cfg.DirectoryContentsAsEntries,
		entries:                    cfg.Entries,
		name:                       cfg.Name,
		lock:                       &sync.RWMutex{},
		path:                       cfg.Path,
		origin:                     cfg.Origin,
		uuid:                       uuid.NewString(),
	}
}

// All returns a copy of all PlaylistEntries being served by the instance of the server.
func (p *Playlist) All() []Entry {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return slices.Clone(p.entries)
}

func (p *Playlist) DirectoryContentsAsEntries() bool {
	return p.directoryContentsAsEntries
}

func (p *Playlist) CurrentEntryIdx() int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.entryIdx
}

// EntriesDiffer checks whether provided entries match entries stored in playlist.
// Currently only paths are taken into account.
func (p *Playlist) EntriesDiffer(entries []Entry) bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	if len(p.entries) != len(entries) {
		return true
	}

	for idx, entry := range p.entries {
		externalEntry := entries[idx]

		if entry.Path != externalEntry.Path {
			return true
		}
	}

	return false
}

// MarshalJSON satisifes json.Marshaller.
func (p *Playlist) MarshalJSON() ([]byte, error) {
	p.lock.Lock()
	pJSON := playlistJSON{
		CurrentEntryIdx:            p.entryIdx,
		DirectoryContentsAsEntries: p.directoryContentsAsEntries,
		Description:                p.description,
		Entries:                    p.entries,
		Name:                       p.name,
		UUID:                       p.uuid,
		Path:                       p.path,
	}
	p.lock.Unlock()

	return json.Marshal(pJSON)
}

func (p *Playlist) Description() string {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.description
}

func (p *Playlist) Name() string {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.name
}

func (p *Playlist) Path() string {
	return p.path
}

func (p *Playlist) setCurrentEntryIdx(idx int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.entryIdx = idx
}

func (p *Playlist) setEntries(entries []Entry) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.entries = entries
}

func (p *Playlist) UUID() string {
	return p.uuid
}

func (p *Playlist) Origin() PlaylistOrigin {
	return p.origin
}
