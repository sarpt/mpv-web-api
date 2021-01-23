package api

const (
	defaultName string = "default"
)

// Playlist holds state about currently playing playlist.
type Playlist struct {
	name       string
	currentIdx int
	items      []string
	changes    chan interface{}
}

type playlistJSON struct {
	Name       string   `json:"Name"`
	CurrentIdx int      `json:"CurrentIdx"`
	Items      []string `json:"Items"`
}
