package state

import (
	"encoding/json"

	"github.com/google/uuid"
)

const (
	defaultName string = "default"
)

// Playlist holds state about currently playing playlist.
type Playlist struct {
	items []string
	name  string
	uuid  string
}

type playlistJSON struct {
	Items []string `json:"Items"`
	Name  string   `json:"Name"`
	UUID  string   `json:"UUID"`
}

type PlaylistConfig struct {
	Name string
}

// NewPlaylist constructs Playlist state.
func NewPlaylist(cfg PlaylistConfig) *Playlist {
	var name string = defaultName
	if cfg.Name != "" {
		name = cfg.Name
	}

	return &Playlist{
		items: []string{},
		name:  name,
		uuid:  uuid.NewString(),
	}
}

// MarshalJSON satisifes json.Marshaller.
func (p *Playlist) MarshalJSON() ([]byte, error) {
	pJSON := playlistJSON{
		Items: p.items,
		Name:  p.name,
		UUID:  p.uuid,
	}
	return json.Marshal(pJSON)
}

func (p *Playlist) UUID() string {
	return p.uuid
}
