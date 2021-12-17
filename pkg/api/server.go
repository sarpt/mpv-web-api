package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sarpt/mpv-web-api/internal/rest"
	"github.com/sarpt/mpv-web-api/internal/sse"
	"github.com/sarpt/mpv-web-api/pkg/mpv"
	"github.com/sarpt/mpv-web-api/pkg/state"
)

const (
	logPrefix           = "api.Server#"
	defaultPlaylistUUID = ""
)

type observePropertyHandler = func(res mpv.ObservePropertyResponse) error

// Server is used to serve API and hold state accessible to the API.
type Server struct {
	address               string
	defaultPlaylistUUID   string
	directories           *state.Directories
	errLog                *log.Logger
	fsWatcher             *fsnotify.Watcher
	mediaFiles            *state.MediaFiles
	mpvManager            *mpv.Manager
	mpvSocketPath         string
	outLog                *log.Logger
	playback              *state.Playback
	playlists             *state.Playlists
	playlistFilesPrefixes []string
	restServer            *rest.Server
	sseObserverChanges    chan sse.ObserversChange
	sseServer             *sse.Server
	status                *state.Status
}

// Config controls behaviour of the api server.
type Config struct {
	Address                 string
	AllowCORS               bool
	ErrWriter               io.Writer
	MpvSocketPath           string
	PlaylistFilesPrefixes   []string
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

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("could not initialize filesystem watcher: %w", err)
	}

	managerCfg := mpv.ManagerConfig{
		ErrWriter:               cfg.ErrWriter,
		MpvSocketPath:           cfg.MpvSocketPath,
		OutWriter:               cfg.OutWriter,
		SocketConnectionTimeout: cfg.SocketConnectionTimeout,
		StartMpvInstance:        cfg.StartMpvInstance,
	}
	mpvManager := mpv.NewManager(managerCfg)

	directories := state.NewDirectories()
	mediaFiles := state.NewMediaFiles()
	playback := state.NewPlayback()
	playlists := state.NewPlaylists()
	status := state.NewStatus()

	sseObserversChanges := make(chan sse.ObserversChange)
	sseCfg := sse.Config{
		Directories: directories,
		ErrWriter:   cfg.ErrWriter,
		MediaFiles:  mediaFiles,
		OutWriter:   cfg.OutWriter,
		Playback:    playback,
		Playlists:   playlists,
		Status:      status,
	}
	sseServer := sse.NewServer(sseCfg)

	restCfg := rest.Config{
		AllowCORS:   cfg.AllowCORS,
		Directories: directories,
		ErrWriter:   cfg.ErrWriter,
		MediaFiles:  mediaFiles,
		MPVManger:   mpvManager,
		OutWriter:   cfg.OutWriter,
		Playback:    playback,
		Playlists:   playlists,
		Status:      status,
	}
	restServer := rest.NewServer(restCfg)

	server := &Server{
		address:               cfg.Address,
		defaultPlaylistUUID:   defaultPlaylistUUID,
		directories:           directories,
		errLog:                log.New(cfg.ErrWriter, logPrefix, log.LstdFlags),
		fsWatcher:             watcher,
		mediaFiles:            mediaFiles,
		mpvManager:            mpvManager,
		mpvSocketPath:         cfg.MpvSocketPath,
		outLog:                log.New(cfg.OutWriter, logPrefix, log.LstdFlags),
		playback:              playback,
		playlists:             playlists,
		playlistFilesPrefixes: cfg.PlaylistFilesPrefixes,
		restServer:            restServer,
		sseObserverChanges:    sseObserversChanges,
		sseServer:             sseServer,
		status:                status,
	}

	restServer.SetAddDirectoriesCallback(server.AddRootDirectories)
	restServer.SetDeleteDirectoriesCallback(server.TakeDirectory)
	restServer.SetLoadPlaylistCallback(server.LoadPlaylist)
	err = server.initWatchers()
	if err != nil {
		return server, errors.New("could not start watching for properties")
	}

	defaultPlaylistUUID, err := server.createDefaultPlaylist()
	if err != nil {
		return server, err
	}

	server.defaultPlaylistUUID = defaultPlaylistUUID
	server.playback.SelectPlaylist(defaultPlaylistUUID)

	return server, nil
}

// Serve starts handling API endpoints - both REST and SSE.
// It also starts mpv manager.
// Blocks until either mpv manager or http server stops serving (with error or nil).
func (s *Server) Serve() error {
	s.watchForFsChanges()

	mpvManagerErr := make(chan error)
	httpServErr := make(chan error)

	serv := http.Server{
		Addr:    s.address,
		Handler: s.mainHandler(),
	}

	go func() {
		mpvManagerErr <- s.mpvManager.Serve()

		close(mpvManagerErr)
	}()

	go func() {
		s.outLog.Printf("running server at %s\n", s.address)
		err := serv.ListenAndServe()
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
	go s.sseServer.WatchSSEObserversChanges()
	s.sseServer.SubscribeToStateChanges()
	s.playback.Subscribe(s.handlePlaylistRelatedPlaybackChanges, func(err error) {})

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
	go s.watchObservePropertyResponses(observePropertyHandlers, observePropertyResponses)

	return s.subscribeToMpvProperties(observePropertyResponses)
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
			s.errLog.Printf("error during '%s' property observer handling: %s\n", observePropertyResponse.Property, err)
		}
	}
}

func (s Server) subscribeToMpvProperties(observeResponses chan mpv.ObservePropertyResponse) error {
	for _, propertyName := range mpv.ObservableProperties {
		_, err := s.mpvManager.SubscribeToProperty(propertyName, observeResponses)
		if err != nil {
			return fmt.Errorf("could not initialize watchers due to error when observing property: %w", err)
		}
	}

	return nil
}
