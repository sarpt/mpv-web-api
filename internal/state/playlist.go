package state

import "encoding/json"

const (
	defaultName string = "default"
)

// PlaybackChangeVariant specifies type of change that happened to a playlist.
type PlaylistChangeVariant string

const (
	// PlaylistReplay notifies about replay of a whole playlist state.
	PlaylistReplay PlaylistChangeVariant = "replay"

	// CurrentIdxChange notifies about change of currently played entry in a playlist.
	CurrentIdxChange PlaylistChangeVariant = "currentIdxChange"

	// PlaylistItemsChange notifies about changes to the items/entries in a playlist.
	PlaylistItemsChange PlaylistChangeVariant = "playlistItemsChange"
)

// PlaylistChange is used to inform about changes to the Playback.
type PlaylistChange struct {
	Variant PlaylistChangeVariant
}

// Playlist holds state about currently playing playlist.
type Playlist struct {
	changes    chan interface{}
	currentIdx int
	items      []string
	name       string
}

type playlistJSON struct {
	CurrentIdx int      `json:"CurrentIdx"`
	Items      []string `json:"Items"`
	Name       string   `json:"Name"`
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
		changes: make(chan interface{}),
		items:   []string{},
		name:    name,
	}
}

func (p *Playlist) Changes() <-chan interface{} {
	return p.changes
}

// MarshalJSON satisifes json.Marshaller.
func (p *Playlist) MarshalJSON() ([]byte, error) {
	pJSON := playlistJSON{
		CurrentIdx: p.currentIdx,
		Items:      p.items,
		Name:       p.name,
	}
	return json.Marshal(pJSON)
}

// SetCurrentIdx sets currently played index in a playlist.
func (p *Playlist) SetCurrentIdx(idx int) {
	p.currentIdx = idx
	p.changes <- PlaylistChange{
		Variant: CurrentIdxChange,
	}
}

// SetItems sets items of the playlist.
func (p *Playlist) SetItems(items []string) {
	p.items = items
	p.changes <- PlaylistChange{
		Variant: PlaylistItemsChange,
	}
}
