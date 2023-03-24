package state

import (
	"github.com/sarpt/mpv-web-api/internal/common"
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
	directoriesBroadcaster := createAndInitChangesBroadcaster[directories.Change]()
	mediaFilesBroadcaster := createAndInitChangesBroadcaster[media_files.Change]()
	playbackBroadcaster := createAndInitChangesBroadcaster[playback.Change]()
	playlistsBroadcaster := createAndInitChangesBroadcaster[playlists.Change]()
	statusBroadcaster := createAndInitChangesBroadcaster[status.Change]()

	return &inMemoryRepository{
		directories: directories.NewStorage(directoriesBroadcaster),
		mediaFiles:  media_files.NewStorage(mediaFilesBroadcaster),
		playback:    playback.NewStorage(playbackBroadcaster),
		playlists:   playlists.NewStorage(playlistsBroadcaster),
		status:      status.NewStorage(statusBroadcaster),
	}
}

func createAndInitChangesBroadcaster[Change common.Change]() *common.ChangesBroadcaster[Change] {
	broadcaster := common.NewChangesBroadcaster[Change]()
	broadcaster.Broadcast()

	return broadcaster
}
