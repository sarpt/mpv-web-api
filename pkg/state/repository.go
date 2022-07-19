package state

import (
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/directories"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/media_files"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playlists"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/status"
)

type Repository interface {
	Directories() *directories.Storage
	MediaFiles() *media_files.Storage
	Playback() *playback.Storage
	Playlists() *playlists.Storage
	Status() *status.Storage
}

type inMemoryRepository struct {
	directories *directories.Storage
	mediaFiles  *media_files.Storage
	playback    *playback.Storage
	playlists   *playlists.Storage
	status      *status.Storage
}

func (r *inMemoryRepository) Directories() *directories.Storage {
	return r.directories
}

func (r *inMemoryRepository) MediaFiles() *media_files.Storage {
	return r.mediaFiles
}

func (r *inMemoryRepository) Playback() *playback.Storage {
	return r.playback
}

func (r *inMemoryRepository) Playlists() *playlists.Storage {
	return r.playlists
}

func (r *inMemoryRepository) Status() *status.Storage {
	return r.status
}

func NewRepository() Repository {
	return &inMemoryRepository{
		directories: directories.NewStorage(),
		mediaFiles:  media_files.NewStorage(),
		playback:    playback.NewStorage(),
		playlists:   playlists.NewStorage(),
		status:      status.NewStorage(),
	}
}
