package sse

import (
	"io"
	"log"
	"net/http"

	"github.com/sarpt/mpv-web-api/pkg/state"
)

const (
	logPrefix = "sse.Server#"

	registerPath = "/sse/channels"
)

// Server holds information about handled SSE connections and their observers.
type Server struct {
	directories          *state.Directories
	directoriesObservers observers
	errLog               *log.Logger
	mediaFiles           *state.MediaFiles // TODO: this state passing from the user is very iffy - consider using either callbacks or builder pattern.
	mediaFilesObservers  observers
	observersChange      chan<- ObserversChange
	outLog               *log.Logger
	playback             *state.Playback
	playbackObservers    observers
	playlists            *state.Playlists
	playlistsObservers   observers
	status               *state.Status
	statusObservers      observers
}

// Config controls behaviour of the SSE server.
type Config struct {
	Directories      *state.Directories
	ErrWriter        io.Writer
	MediaFiles       *state.MediaFiles
	ObserversChanges chan<- ObserversChange
	OutWriter        io.Writer
	Playback         *state.Playback
	Playlists        *state.Playlists
	Status           *state.Status
}

// NewServer prepares and returns SSE server to handle SSE connections and observers.
func NewServer(cfg Config) *Server {
	return &Server{
		directories:          cfg.Directories,
		directoriesObservers: newDirectoryObserver(),
		errLog:               log.New(cfg.ErrWriter, logPrefix, log.LstdFlags),
		mediaFiles:           cfg.MediaFiles,
		mediaFilesObservers:  newMediaFilesObservers(),
		observersChange:      cfg.ObserversChanges,
		outLog:               log.New(cfg.OutWriter, logPrefix, log.LstdFlags),
		playback:             cfg.Playback,
		playbackObservers:    newPlaybackObservers(),
		playlists:            cfg.Playlists,
		playlistsObservers:   newPlaylistsObservers(),
		status:               cfg.Status,
		statusObservers:      newStatusObservers(),
	}
}

// InitDispatchers starts listening on state changes channels for further distribution to its observers.
func (s *Server) InitDispatchers() {
	directoriesObservers := s.directoriesObservers.(*directoriesObserver) // TODO: Ooof... Eww... Remove when rewriting with generics
	s.directories.Subscribe(directoriesObservers.BroadcastToChannelObservers, func(err error) {})

	playbackObservers := s.playbackObservers.(*playbackObservers)
	s.playback.Subscribe(playbackObservers.BroadcastToChannelObservers, func(err error) {})

	playlistsObservers := s.playlistsObservers.(*playlistsObservers)
	s.playlists.Subscribe(playlistsObservers.BroadcastToChannelObservers, func(err error) {})

	mediaFilesObservers := s.mediaFilesObservers.(*mediaFilesObservers)
	s.mediaFiles.Subscribe(mediaFilesObservers.BroadcastToChannelObservers, func(err error) {})

	statusObservers := s.statusObservers.(*statusObservers)
	s.status.Subscribe(statusObservers.BroadcastToChannelObservers, func(err error) {})
}

// Handler returns map of HTTPs methods and their handlers.
func (s *Server) Handler() http.Handler {
	sseCfg := handlerConfig{
		Channels: map[state.SSEChannelVariant]channel{
			directoriesSSEChannelVariant: s.directoriesSSEChannel(),
			playbackSSEChannelVariant:    s.playbackSSEChannel(),
			playlistsSSEChannelVariant:   s.playlistsSSEChannel(),
			mediaFilesSSEChannelVariant:  s.mediaFilesSSEChannel(),
			statusSSEChannelVariant:      s.statusSSEChannel(),
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc(registerPath, s.createSseRegisterHandler(sseCfg))

	return mux
}
