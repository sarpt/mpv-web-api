package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sarpt/mpv-web-api/internal/rest"
	"github.com/sarpt/mpv-web-api/internal/sse"
	"github.com/sarpt/mpv-web-api/internal/state"
	"github.com/sarpt/mpv-web-api/pkg/mpv"
)

const (
	logPrefix = "api.Server#"
)

type observePropertyHandler = func(res mpv.ObservePropertyResponse) error

// Server is used to serve API and hold state accessible to the API.
type Server struct {
	address            string
	directories        []string
	directoriesLock    *sync.RWMutex
	errLog             *log.Logger
	movies             *state.Movies
	mpvManager         *mpv.Manager
	mpvSocketPath      string
	outLog             *log.Logger
	playback           *state.Playback
	playlists          *state.Playlists
	restServer         *rest.Server
	sseObserverChanges chan sse.ObserversChange
	sseServer          *sse.Server
	status             *state.Status
}

// Config controls behaviour of the api server.
type Config struct {
	Address                 string
	AllowCORS               bool
	ErrWriter               io.Writer
	MpvSocketPath           string
	OutWriter               io.Writer
	SocketConnectionTimeout time.Duration
	StartMpvInstance        bool
}

// NewServer prepares and returns a server that can be used to handle API calls.
func NewServer(cfg Config) (*Server, error) {
	if cfg.OutWriter == nil {
		cfg.OutWriter = os.Stdout
	}
	if cfg.ErrWriter == nil {
		cfg.ErrWriter = os.Stderr
	}

	managerCfg := mpv.ManagerConfig{
		ErrWriter:               cfg.ErrWriter,
		MpvSocketPath:           cfg.MpvSocketPath,
		OutWriter:               cfg.OutWriter,
		SocketConnectionTimeout: cfg.SocketConnectionTimeout,
		StartMpvInstance:        cfg.StartMpvInstance,
	}
	mpvManager := mpv.NewManager(managerCfg)
	movies := state.NewMovies()
	playback := state.NewPlayback()
	playlists := state.NewPlaylists()
	status := state.NewStatus()

	sseObserversChanges := make(chan sse.ObserversChange)
	sseCfg := sse.Config{
		ErrWriter:        cfg.ErrWriter,
		Movies:           movies,
		OutWriter:        cfg.OutWriter,
		ObserversChanges: sseObserversChanges,
		Playback:         playback,
		Playlists:        playlists,
		Status:           status,
	}
	sseServer := sse.NewServer(sseCfg)

	restCfg := rest.Config{
		AllowCORS: cfg.AllowCORS,
		ErrWriter: cfg.ErrWriter,
		Movies:    movies,
		MPVManger: mpvManager,
		OutWriter: cfg.OutWriter,
		Playback:  playback,
		Status:    status,
	}
	restServer := rest.NewServer(restCfg)

	server := &Server{
		cfg.Address,
		[]string{},
		&sync.RWMutex{},
		log.New(cfg.ErrWriter, logPrefix, log.LstdFlags),
		movies,
		mpvManager,
		cfg.MpvSocketPath,
		log.New(cfg.OutWriter, logPrefix, log.LstdFlags),
		playback,
		playlists,
		restServer,
		sseObserversChanges,
		sseServer,
		status,
	}

	restServer.SetAddDirectoriesHandler(server.AddDirectories)

	return server, nil
}

// Serve starts handling API endpoints - both REST and SSE.
// It also starts mpv manager.
// Blocks until either mpv manager or http server stops serving (with error or nil).
func (s *Server) Serve() error {
	mpvManagerErr := make(chan error)
	httpServErr := make(chan error)

	serv := http.Server{
		Addr:    s.address,
		Handler: s.mainHandler(),
	}

	err := s.initWatchers()
	if err != nil {
		return errors.New("could not start watching for properties")
	}

	go func() {
		mpvManagerErr <- s.mpvManager.Serve()

		close(mpvManagerErr)
	}()

	go func() {
		s.outLog.Printf("running server at %s\n", s.address)
		err = serv.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			httpServErr <- err
		}

		close(httpServErr)
	}()

	select {
	case err := <-mpvManagerErr:
		serv.Shutdown(context.Background())
		return err
	case err := <-httpServErr:
		s.mpvManager.Close()
		return err
	}
}

func (s *Server) initWatchers() error {
	observePropertyResponses := make(chan mpv.ObservePropertyResponse)
	observePropertyHandlers := map[string]observePropertyHandler{
		mpv.AudioIDProperty:            s.handleAudioIDChangeEvent,
		mpv.ChapterProperty:            s.handleChapterChangeEvent,
		mpv.FullscreenProperty:         s.handleFullscreenEvent,
		mpv.LoopFileProperty:           s.handleLoopFileEvent,
		mpv.PathProperty:               s.handlePathEvent,
		mpv.PauseProperty:              s.handlePauseEvent,
		mpv.PlaybackTimeProperty:       s.handlePlaybackTimeEvent,
		mpv.PlaylistProperty:           s.handlePlaylistProperty,
		mpv.PlaylistPlayingPosProperty: s.handlePlaylistPlayingPosEvent,
		mpv.SubtitleIDProperty:         s.handleSubtitleIDChangeEvent,
	}

	go s.watchSSEObserversChanges()
	s.sseServer.InitDispatchers()
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

func (s Server) watchSSEObserversChanges() {
	for {
		change, open := <-s.sseObserverChanges
		if !open {
			return
		}

		switch change.ChangeVariant {
		case sse.ObserverAdded:
			s.status.AddObservingAddress(change.RemoteAddr, change.ChannelVariant)
		case sse.ObserverRemoved:
			s.status.RemoveObservingAddress(change.RemoteAddr, change.ChannelVariant)
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
