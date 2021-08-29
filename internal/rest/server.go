package rest

import (
	"io"
	"log"

	"github.com/sarpt/mpv-web-api/internal/state"
	"github.com/sarpt/mpv-web-api/pkg/mpv"
)

const (
	logPrefix = "rest.Server#"
)

// Config controls behaviour of the REST server.
type Config struct {
	AllowCORS  bool
	ErrWriter  io.Writer
	MediaFiles *state.MediaFiles
	MPVManger  *mpv.Manager
	Playback   *state.Playback
	Playlist   *state.Playlist
	OutWriter  io.Writer
	Status     *state.Status
}

// Server is responsible for creating REST handlers, argument parsing and validation.
// TODO: In the future, REST package might fullfill a function of a wrapper for OpenAPI generated code
// (if that ever will be implemented, not sure if it's not an overkill atm).
// TODO#2: As in SSE case, the pointers to the state should be replaced with a more separated approach
// - rest package should not have unlimited access to the whole state.
type Server struct {
	addDirectoriesHandler func([]string) error
	allowCORS             bool
	errLog                *log.Logger
	mediaFiles            *state.MediaFiles
	mpvManager            *mpv.Manager
	playback              *state.Playback
	playlist              *state.Playlist
	outLog                *log.Logger
	status                *state.Status
}

// NewServer returns rest.Server instance.
func NewServer(cfg Config) *Server {
	return &Server{
		allowCORS:  cfg.AllowCORS,
		errLog:     log.New(cfg.ErrWriter, logPrefix, log.LstdFlags),
		mediaFiles: cfg.MediaFiles,
		mpvManager: cfg.MPVManger,
		playback:   cfg.Playback,
		playlist:   cfg.Playlist,
		outLog:     log.New(cfg.OutWriter, logPrefix, log.LstdFlags),
		status:     cfg.Status,
	}
}
