package sse

import (
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/sarpt/mpv-web-api/internal/state"
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
		directories: cfg.Directories,
		directoriesObservers: observers{
			items: map[string]chan interface{}{},
			lock:  &sync.RWMutex{},
		},
		errLog:     log.New(cfg.ErrWriter, logPrefix, log.LstdFlags),
		mediaFiles: cfg.MediaFiles,
		mediaFilesObservers: observers{
			items: map[string]chan interface{}{},
			lock:  &sync.RWMutex{},
		},
		observersChange: cfg.ObserversChanges,
		outLog:          log.New(cfg.OutWriter, logPrefix, log.LstdFlags),
		playback:        cfg.Playback,
		playbackObservers: observers{
			items: map[string]chan interface{}{},
			lock:  &sync.RWMutex{},
		},
		playlists: cfg.Playlists,
		playlistsObservers: observers{
			items: map[string]chan interface{}{},
			lock:  &sync.RWMutex{},
		},
		status: cfg.Status,
		statusObservers: observers{
			items: map[string]chan interface{}{},
			lock:  &sync.RWMutex{},
		},
	}
}

// InitDispatchers starts listening on state changes channels for further distribution to its observers.
// TODO: Changes of specific methods should be aware that they've been already called
// and someone is listening to avoid unwanted (unsupported by channels) broadcast
// (maybe channels are unsuitable in this situation at all - what else is there to consider?).
func (s *Server) InitDispatchers() {
	go distributeChangesToChannelObservers(s.directories.Changes(), s.directoriesObservers)
	go distributeChangesToChannelObservers(s.playback.Changes(), s.playbackObservers)
	go distributeChangesToChannelObservers(s.playlists.Changes(), s.playlistsObservers)
	go distributeChangesToChannelObservers(s.mediaFiles.Changes(), s.mediaFilesObservers)
	go distributeChangesToChannelObservers(s.status.Changes(), s.statusObservers)
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
