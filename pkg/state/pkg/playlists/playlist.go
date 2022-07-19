package playlists

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

// Playlist holds state about currently playing playlist.
type Playlist struct {
	currentEntryIdx            int
	description                string
	directoryContentsAsEntries bool
	entries                    []PlaylistEntry
	name                       string
	lock                       *sync.RWMutex
	path                       string
	uuid                       string
}

type playlistJSON struct {
	CurrentEntryIdx            int             `json:"CurrentEntryIdx"`
	Description                string          `json:"Description"`
	DirectoryContentsAsEntries bool            `json:"DirectoryContentsAsEntries"`
	Entries                    []PlaylistEntry `json:"Entries"`
	Name                       string          `json:"Name"`
	Path                       string          `json:"Path"`
	UUID                       string          `json:"UUID"`
}

type PlaylistConfig struct {
	CurrentEntryIdx            int
	Description                string
	DirectoryContentsAsEntries bool
	Entries                    []PlaylistEntry
	Name                       string
	Path                       string
}

// NewPlaylist constructs Playlist state.
func NewPlaylist(cfg PlaylistConfig) *Playlist {
	return &Playlist{
		currentEntryIdx:            cfg.CurrentEntryIdx,
		description:                cfg.Description,
		directoryContentsAsEntries: cfg.DirectoryContentsAsEntries,
		entries:                    cfg.Entries,
		name:                       cfg.Name,
		lock:                       &sync.RWMutex{},
		path:                       cfg.Path,
		uuid:                       uuid.NewString(),
	}
}

// All returns a copy of all PlaylistEntries being served by the instance of the server.
func (p *Playlist) All() []PlaylistEntry {
	entries := []PlaylistEntry{}

	p.lock.RLock()
	defer p.lock.RUnlock()

	return append(entries, p.entries...)
}

func (p *Playlist) DirectoryContentsAsEntries() bool {
	return p.directoryContentsAsEntries
}

func (p *Playlist) CurrentEntryIdx() int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.currentEntryIdx
}

// EntriesDiffer checks whether provided entries match entries stored in playlist.
// Currently only paths are taken into account.
func (p *Playlist) EntriesDiffer(entries []PlaylistEntry) bool {
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
		DirectoryContentsAsEntries: p.directoryContentsAsEntries,
		Description:                p.description,
		Entries:                    p.entries,
		Name:                       p.name,
		UUID:                       p.uuid,
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

	p.currentEntryIdx = idx
}

func (p *Playlist) setEntries(entries []PlaylistEntry) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.entries = entries
}

func (p *Playlist) UUID() string {
	return p.uuid
}
