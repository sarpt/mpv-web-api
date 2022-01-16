package rest

import (
	"io"
	"log"

	"github.com/sarpt/mpv-web-api/pkg/api"
	"github.com/sarpt/mpv-web-api/pkg/state"
)

const (
	logPrefix = "rest.Server#"

	name     = "REST Server"
	pathBase = "rest"
)

// Config controls behaviour of the REST server.
type Config struct {
	AllowCORS bool
	ErrWriter io.Writer
	OutWriter io.Writer
}

type Callbacks struct {
	addDirectoriesCb
	removeDirectoriesCb
	loadPlaylistCb
	loadFileCb
	changeFullscreenCb
	changeAudioCb
	changeChapterCb
	changeSubtitleCb
	loopFileCb
	changePauseCb
	playlistPlayIndexCb
	stopPlaybackCb
}

// Server is responsible for creating REST handlers, argument parsing and validation.
// TODO: In the future, REST package might fullfill a function of a wrapper for OpenAPI generated code
// (if that ever will be implemented, not sure if it's not an overkill atm).
// TODO#2: As in SSE case, the pointers to the state should be replaced with a more separated approach
// - rest package should not have unlimited access to the whole state.
type Server struct {
	Callbacks
	allowCORS   bool
	directories *state.Directories
	errLog      *log.Logger
	mediaFiles  *state.MediaFiles
	playback    *state.Playback
	playlists   *state.Playlists
	outLog      *log.Logger
	status      *state.Status
}

// NewServer returns rest.Server instance.
func NewServer(cfg Config) *Server {
	return &Server{
		allowCORS: cfg.AllowCORS,
		errLog:    log.New(cfg.ErrWriter, logPrefix, log.LstdFlags),
		outLog:    log.New(cfg.OutWriter, logPrefix, log.LstdFlags),
	}
}

func (s *Server) Init(apiServer *api.Server) error {
	s.directories = apiServer.Directories()
	s.mediaFiles = apiServer.MediaFiles()
	s.playback = apiServer.Playback()
	s.playlists = apiServer.Playlists()
	s.status = apiServer.Status()

	s.addDirectoriesCb = apiServer.AddRootDirectories
	s.removeDirectoriesCb = apiServer.TakeDirectory
	s.loadPlaylistCb = apiServer.LoadPlaylist

	s.loadFileCb = apiServer.LoadFile
	s.changeFullscreenCb = apiServer.ChangeFullscreen
	s.changeAudioCb = apiServer.ChangeAudio
	s.changeChapterCb = apiServer.ChangeChapter
	s.changeSubtitleCb = apiServer.ChangeSubtitle
	s.loopFileCb = apiServer.LoopFile
	s.changePauseCb = apiServer.ChangePause
	s.playlistPlayIndexCb = apiServer.PlaylistPlayIndex
	s.stopPlaybackCb = apiServer.StopPlayback

	return nil
}

func (s *Server) Name() string {
	return name
}

func (s *Server) PathBase() string {
	return pathBase
}

func (s *Server) Shutdown() {}
