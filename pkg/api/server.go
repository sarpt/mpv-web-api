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
	"github.com/sarpt/mpv-web-api/pkg/mpv"
	"github.com/sarpt/mpv-web-api/pkg/state"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/directories"
)

const (
	logPrefix           = "api.Server#"
	defaultPlaylistUUID = ""
)

type observePropertyHandler = func(res mpv.ObservePropertyResponse) error

// Server is used to serve API and hold state accessible to the API.
type Server struct {
	address               string
	stopServing           chan string
	defaultPlaylistUUID   string
	errLog                *log.Logger
	fsWatcher             *fsnotify.Watcher
	mpvManager            *mpv.Manager
	outLog                *log.Logger
	statesRepository      state.Repository
	playlistFilesPrefixes []string
	pluginServers         map[string]PluginServer
}

type PluginApi interface {
	AddRootDirectories(directories []directories.Entry)
	ChangeChaptersOrder(chapters []int64, force bool) error
	TakeDirectory(path string) (directories.Entry, error)
	LoadPlaylist(uuid string, append bool) error
	LoadFile(filePath string, append bool) error
	ChangeFullscreen(fullscreen bool) error
	ChangeAudio(audioId string) error
	ChangeChapter(idx int64) error
	ChangeSubtitle(subtitleId string) error
	LoopFile(looped bool) error
	ChangePause(paused bool) error
	PlaylistPlayIndex(idx int) error
	StopPlayback() error
	WaitUntilMediaFile(mediaFilePath string) error
}

type PluginServer interface {
	Init(apiServ PluginApi) error
	Handler() http.Handler
	PathBase() string
	Name() string
	Shutdown()
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
	StatesRepository        state.Repository
	PluginServers           map[string]PluginServer
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

	mpvManagerCfg := mpv.ManagerConfig{
		ErrWriter:               cfg.ErrWriter,
		MpvSocketPath:           cfg.MpvSocketPath,
		OutWriter:               cfg.OutWriter,
		SocketConnectionTimeout: cfg.SocketConnectionTimeout,
		StartMpvInstance:        cfg.StartMpvInstance,
	}

	server := &Server{
		address:               cfg.Address,
		defaultPlaylistUUID:   defaultPlaylistUUID,
		errLog:                log.New(cfg.ErrWriter, logPrefix, log.LstdFlags),
		fsWatcher:             watcher,
		mpvManager:            mpv.NewManager(mpvManagerCfg),
		outLog:                log.New(cfg.OutWriter, logPrefix, log.LstdFlags),
		statesRepository:      cfg.StatesRepository,
		playlistFilesPrefixes: cfg.PlaylistFilesPrefixes,
		pluginServers:         cfg.PluginServers,
	}

	defaultPlaylistUUID, err := server.createDefaultPlaylist()
	if err != nil {
		return server, err
	}

	server.defaultPlaylistUUID = defaultPlaylistUUID
	server.statesRepository.Playback().SelectPlaylist(defaultPlaylistUUID)

	return server, nil
}

func (s *Server) init() error {
	for name, server := range s.pluginServers {
		s.outLog.Printf("initializing plugin server '%s' ...", name)
		err := server.Init(s)
		if err != nil {
			return fmt.Errorf("could not initialise plugin server '%s': %w", name, err)
		}
	}

	err := s.initWatchers()
	if err != nil {
		return fmt.Errorf("could not start watching for properties: %w", err)
	}

	s.watchForFsChanges()

	return nil
}

// StopServing instructs server to close API servers & mpv manager with a provided reason.
func (s *Server) StopServing(reason string) error {
	if s.stopServing == nil {
		return fmt.Errorf("server stop unsuccessful - server is not running")
	}

	s.stopServing <- reason
	return nil
}

// Serve starts handling plugin API servers passed to the server.
// It also starts mpv manager and (if neccessary).
// Blocks until either mpv manager or http server stops serving (with error or nil).
func (s *Server) Serve() error {
	err := s.init()
	if err != nil {
		return err
	}

	mpvManagerErr := make(chan error)
	httpServErr := make(chan error)

	serv := http.Server{
		Addr:    s.address,
		Handler: s.mainHandler(),
	}

	s.stopServing = make(chan string)
	defer func() { s.stopServing = nil }() // when Serve stops, whatever the reason, nothing will listen on the chan until Serve is called again

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
	case reason := <-s.stopServing:
		s.outLog.Printf("shutting down the server, reason: %s", reason)
	case err := <-mpvManagerErr:
		s.outLog.Printf("shutting down the server due to mpv manager error: %s", err)
	case err := <-httpServErr:
		s.outLog.Printf("shutting down the server due to http server error: %s", err)
	}

	err = s.saveCurrentPlaylist()
	if err != nil {
		s.errLog.Printf("saving of current playlist unsuccessful: %s\n", err)
	}

	err = s.mpvManager.Shutdown("API server shutting down")
	if err != nil {
		s.errLog.Printf("mpvManager closed with an error: %s\n", err)
	} else {
		s.outLog.Println("mpvManager closed successfully")
	}

	for name, serv := range s.pluginServers {
		s.outLog.Printf("shutting down '%s' plugin server\n", name)
		serv.Shutdown()
	}

	err = serv.Shutdown(context.Background())
	if err != nil {
		s.errLog.Printf("http server closed with an error: %s\n", err)
	} else {
		s.outLog.Println("http server closed successfully")
	}

	return nil
}

func (s *Server) initWatchers() error {
	s.statesRepository.Playback().Subscribe(s.handlePlaylistRelatedPlaybackChanges, func(err error) {})

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
