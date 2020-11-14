package api

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/mpv"
)

const (
	logPrefix = "api.Server#"
)

type observePropertyHandler = func(res mpv.ObservePropertyResponse) error

// Server is used to serve API and hold state accessible to the API
type Server struct {
	address              string
	allowCors            bool
	directories          []string
	directoriesLock      *sync.RWMutex
	movies               Movies
	moviesSSEObservers   SSEObservers
	mpvManager           *mpv.Manager
	mpvSocketPath        string
	playback             *Playback
	playbackSSEObservers SSEObservers
	status               *Status
	statusSSEObservers   SSEObservers
	errLog               *log.Logger
	outLog               *log.Logger
}

// Config controls behaviour of the api serve
type Config struct {
	Address       string
	AllowCors     bool
	MpvSocketPath string
	outWriter     io.Writer
	errWriter     io.Writer
}

// NewServer prepares and returns a server that can be used to handle API
func NewServer(cfg Config) (*Server, error) {
	if cfg.outWriter == nil {
		cfg.outWriter = os.Stdout
	}
	if cfg.errWriter == nil {
		cfg.errWriter = os.Stderr
	}

	mpvManager := mpv.NewManager(cfg.MpvSocketPath, cfg.outWriter, cfg.errWriter)

	return &Server{
		cfg.Address,
		cfg.AllowCors,
		[]string{},
		&sync.RWMutex{},
		Movies{
			items:   map[string]Movie{},
			Changes: make(chan interface{}),
			lock:    &sync.RWMutex{},
		},
		SSEObservers{
			Items: map[string]chan interface{}{},
			Lock:  &sync.RWMutex{},
		},
		mpvManager,
		cfg.MpvSocketPath,
		&Playback{
			Changes: make(chan interface{}),
		},
		SSEObservers{
			Items: map[string]chan interface{}{},
			Lock:  &sync.RWMutex{},
		},
		&Status{
			observingAddresses: map[string][]SSEChannelVariant{},
			lock:               &sync.RWMutex{},
			Changes:            make(chan interface{}),
		},
		SSEObservers{
			Items: map[string]chan interface{}{},
			Lock:  &sync.RWMutex{},
		},
		log.New(cfg.outWriter, logPrefix, log.LstdFlags),
		log.New(cfg.errWriter, logPrefix, log.LstdFlags),
	}, nil
}

// Serve starts handling requests to the API endpoints. Blocks until canceled
func (s *Server) Serve() error {
	serv := http.Server{
		Addr:    s.address,
		Handler: s.mainHandler(),
	}

	err := s.initWatchers()
	if err != nil {
		return errors.New("could not start watching for properties")
	}

	s.outLog.Printf("running server at %s\n", s.address)
	return serv.ListenAndServe()
}

// Close closes server, along with closing necessary helpers
func (s Server) Close() {
	s.mpvManager.Close()
}

func (s *Server) initWatchers() error {
	observePropertyResponses := make(chan mpv.ObservePropertyResponse)
	observePropertyHandlers := map[string]observePropertyHandler{
		mpv.FullscreenProperty:   s.handleFullscreenEvent,
		mpv.LoopFileProperty:     s.handleLoopFileEvent,
		mpv.PauseProperty:        s.handlePauseEvent,
		mpv.PathProperty:         s.handlePathEvent,
		mpv.PlaybackTimeProperty: s.handlePlaybackTimeEvent,
		mpv.AudioIDProperty:      s.handleAudioIDChangeEvent,
		mpv.SubtitleIDProperty:   s.handleSubtitleIDChangeEvent,
	}

	go distributeChangesToSSEObservers(s.playback.Changes, s.playbackSSEObservers)
	go distributeChangesToSSEObservers(s.movies.Changes, s.moviesSSEObservers)
	go distributeChangesToSSEObservers(s.status.Changes, s.statusSSEObservers)
	go s.watchObservePropertyResponses(observePropertyHandlers, observePropertyResponses)

	return s.observeProperties(observePropertyResponses)
}

func (s Server) watchObservePropertyResponses(handlers map[string]observePropertyHandler, responses chan mpv.ObservePropertyResponse) {
	for {
		observePropertyResponse, open := <-responses
		if !open {
			return
		}

		observeHandler, ok := handlers[observePropertyResponse.Property]
		if !ok {
			continue
		}

		err := observeHandler(observePropertyResponse)
		if err != nil {
			s.errLog.Printf("could not handle property '%s' observer handling: %s\n", observePropertyResponse.Property, err)
		}
	}
}

func (s Server) observeProperties(observeResponses chan mpv.ObservePropertyResponse) error {
	for _, propertyName := range mpv.ObservableProperties {
		_, err := s.mpvManager.SubscribeToProperty(propertyName, observeResponses)
		if err != nil {
			return fmt.Errorf("could not initialize watchers due to error when observing property: %w", err)
		}
	}

	return nil
}

// distributeChangesToSSEObservers is a fan-out dispatcher, which notifies all playback observers (subscribers from SSE etc.) when a playbackChange occurs.
func distributeChangesToSSEObservers(changes chan interface{}, observers SSEObservers) {
	for {
		change, ok := <-changes
		if !ok {
			return
		}

		observers.Lock.RLock()
		for _, observer := range observers.Items {
			observer <- change
		}
		observers.Lock.RUnlock()
	}
}
