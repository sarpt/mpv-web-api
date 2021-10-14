package state

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

// Playlist holds state about currently playing playlist.
type Playlist struct {
	description                string
	directoryContentsAsEntries bool
	entries                    []PlaylistEntry
	name                       string
	lock                       *sync.RWMutex
	uuid                       string
}

type playlistJSON struct {
	Description                string          `json:"Description"`
	DirectoryContentsAsEntries bool            `json:"DirectoryContentsAsEntries"`
	Entries                    []PlaylistEntry `json:"Entries"`
	Name                       string          `json:"Name"`
	UUID                       string          `json:"UUID"`
}

type PlaylistConfig struct {
	Description                string
	DirectoryContentsAsEntries bool
	Entries                    []PlaylistEntry
	Name                       string
}

// NewPlaylist constructs Playlist state.
func NewPlaylist(cfg PlaylistConfig) *Playlist {
	return &Playlist{
		description:                cfg.Description,
		directoryContentsAsEntries: cfg.DirectoryContentsAsEntries,
		entries:                    []PlaylistEntry{},
		name:                       cfg.Name,
		lock:                       &sync.RWMutex{},
		uuid:                       uuid.NewString(),
	}
}

func (p *Playlist) DirectoryContentsAsEntries() bool {
	return p.directoryContentsAsEntries
}

// MarshalJSON satisifes json.Marshaller.
func (p *Playlist) MarshalJSON() ([]byte, error) {
	pJSON := playlistJSON{
		DirectoryContentsAsEntries: p.directoryContentsAsEntries,
		Description:                p.description,
		Entries:                    p.entries,
		Name:                       p.name,
		UUID:                       p.uuid,
	}
	return json.Marshal(pJSON)
}

func (p *Playlist) UUID() string {
	return p.uuid
}
