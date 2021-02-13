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

	registerPath = "/sse/register"
)

// Server holds information about handled SSE connections and their observers.
type Server struct {
	errLog            *log.Logger
	movies            *state.Movies // TODO: this state passing from the user is very iffy - consider using either callbacks or builder pattern.
	moviesObservers   observers
	observersChange   chan<- ObserversChange
	outLog            *log.Logger
	playback          *state.Playback
	playbackObservers observers
	status            *state.Status
	statusObservers   observers
}

// Config controls behaviour of the SSE server.
type Config struct {
	ErrWriter        io.Writer
	Movies           *state.Movies
	ObserversChanges chan<- ObserversChange
	OutWriter        io.Writer
	Playback         *state.Playback
	Status           *state.Status
}

// NewServer prepares and returns SSE server to handle SSE connections and observers.
func NewServer(cfg Config) *Server {
	return &Server{
		errLog: log.New(cfg.ErrWriter, logPrefix, log.LstdFlags),
		movies: cfg.Movies,
		moviesObservers: observers{
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
	go distributeChangesToChannelObservers(s.playback.Changes(), s.playbackObservers)
	go distributeChangesToChannelObservers(s.movies.Changes(), s.moviesObservers)
	go distributeChangesToChannelObservers(s.status.Changes(), s.statusObservers)
}

// Handler returns map of HTTPs methods and their handlers.
// TODO: This should return ideally a http.Handler for a subtree, to be done when refactoring routing and separating REST handling.
func (s *Server) Handler() http.Handler {
	sseCfg := handlerConfig{
		Channels: map[state.SSEChannelVariant]channel{
			playbackSSEChannelVariant: s.playbackSSEChannel(),
			moviesSSEChannelVariant:   s.moviesSSEChannel(),
			statusSSEChannelVariant:   s.statusSSEChannel(),
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc(registerPath, s.createGetSseRegisterHandler(sseCfg))

	return mux
}
