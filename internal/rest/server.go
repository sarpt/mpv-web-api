package rest

import (
	"io"
	"log"

	"github.com/sarpt/mpv-web-api/pkg/api"
	"github.com/sarpt/mpv-web-api/pkg/mpv"
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

// Server is responsible for creating REST handlers, argument parsing and validation.
// TODO: In the future, REST package might fullfill a function of a wrapper for OpenAPI generated code
// (if that ever will be implemented, not sure if it's not an overkill atm).
// TODO#2: As in SSE case, the pointers to the state should be replaced with a more separated approach
// - rest package should not have unlimited access to the whole state.
type Server struct {
	addDirectoriesCallback    AddDirectoriesCallback
	allowCORS                 bool
	removeDirectoriesCallback RemoveDirectoriesCallback
	directories               *state.Directories
	errLog                    *log.Logger
	loadPlaylistCallback      LoadPlaylistCallback
	mediaFiles                *state.MediaFiles
	mpvManager                *mpv.Manager
	playback                  *state.Playback
	playlists                 *state.Playlists
	outLog                    *log.Logger
	status                    *state.Status
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

	s.mpvManager = apiServer.MpvManager()

	s.addDirectoriesCallback = apiServer.AddRootDirectories
	s.removeDirectoriesCallback = apiServer.TakeDirectory
	s.loadPlaylistCallback = apiServer.LoadPlaylist

	return nil
}

func (s *Server) Name() string {
	return name
}

func (s *Server) PathBase() string {
	return pathBase
}
