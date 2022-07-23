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
	AllowCORS        bool
	ErrWriter        io.Writer
	OutWriter        io.Writer
	StatesRepository state.Repository
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
type Server struct {
	Callbacks
	allowCORS        bool
	errLog           *log.Logger
	outLog           *log.Logger
	statesRepository state.Repository
}

// NewServer returns rest.Server instance.
func NewServer(cfg Config) *Server {
	return &Server{
		allowCORS:        cfg.AllowCORS,
		errLog:           log.New(cfg.ErrWriter, logPrefix, log.LstdFlags),
		outLog:           log.New(cfg.OutWriter, logPrefix, log.LstdFlags),
		statesRepository: cfg.StatesRepository,
	}
}

func (s *Server) Init(apiServer api.PluginApi) error {
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
