package sse

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/sarpt/mpv-web-api/pkg/api"
	"github.com/sarpt/mpv-web-api/pkg/state"
	state_sse "github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
)

const (
	logPrefix = "sse.Server#"

	name     = "SSE Server"
	pathBase = "sse"
)

var (
	registerPath = fmt.Sprintf("/%s/channels", pathBase)
)

// Server holds information about handled SSE connections and their observers.
type Server struct {
	cancel           context.CancelFunc
	channels         map[state_sse.ChannelVariant]channel
	ctx              context.Context
	errLog           *log.Logger
	observersChanges chan ObserversChange
	outLog           *log.Logger
	statesRepository state.Repository
}

// Config controls behaviour of the SSE server.
type Config struct {
	ErrWriter        io.Writer
	OutWriter        io.Writer
	StatesRepository state.Repository
}

// NewServer prepares and returns SSE server to handle SSE connections and observers.
func NewServer(cfg Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		cancel:           cancel,
		channels:         map[state_sse.ChannelVariant]channel{},
		ctx:              ctx,
		errLog:           log.New(cfg.ErrWriter, logPrefix, log.LstdFlags),
		observersChanges: make(chan ObserversChange),
		outLog:           log.New(cfg.OutWriter, logPrefix, log.LstdFlags),
		statesRepository: cfg.StatesRepository,
	}
}

// Handler returns map of HTTPs methods and their handlers.
func (s *Server) Handler() http.Handler {
	sseCfg := handlerConfig{
		Channels: s.channels,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(registerPath, s.createSseRegisterHandler(sseCfg))

	return mux
}

func (s *Server) Init(apiServer *api.Server) error {
	s.channels[directoriesSSEChannelVariant] = newDirectoriesChannel(s.statesRepository.Directories())
	s.channels[mediaFilesSSEChannelVariant] = newMediaFilesChannel(s.statesRepository.MediaFiles())
	s.channels[playbackSSEChannelVariant] = newPlaybackChannel(s.statesRepository.Playback())
	s.channels[playlistsSSEChannelVariant] = newPlaylistsChannel(s.statesRepository.Playback(), s.statesRepository.Playlists())
	s.channels[statusSSEChannelVariant] = newStatusChannel(s.statesRepository.Status())

	go s.watchSSEObserversChanges()
	s.subscribeToStateChanges()

	return nil
}

func (s *Server) Name() string {
	return name
}

func (s *Server) PathBase() string {
	return pathBase
}

func (s *Server) Shutdown() {
	s.cancel()
}

// subscribeToStateChanges starts listening on state changes channels for further distribution to its observers.
func (s *Server) subscribeToStateChanges() {
	directoriesChannel := s.channels[directoriesSSEChannelVariant].(*directoriesChannel)
	s.statesRepository.Directories().Subscribe(directoriesChannel.BroadcastToChannelObservers, func(err error) {})

	playbackChannel := s.channels[playbackSSEChannelVariant].(*playbackChannel)
	s.statesRepository.Playback().Subscribe(playbackChannel.BroadcastToChannelObservers, func(err error) {})

	playlistsChannel := s.channels[playlistsSSEChannelVariant].(*playlistsChannel)
	s.statesRepository.Playlists().Subscribe(playlistsChannel.BroadcastToChannelObservers, func(err error) {})

	mediaFilesChannel := s.channels[mediaFilesSSEChannelVariant].(*mediaFilesChannel)
	s.statesRepository.MediaFiles().Subscribe(mediaFilesChannel.BroadcastToChannelObservers, func(err error) {})

	statusChannel := s.channels[statusSSEChannelVariant].(*statusChannel)
	s.statesRepository.Status().Subscribe(statusChannel.BroadcastToChannelObservers, func(err error) {})
}

func (s Server) watchSSEObserversChanges() {
	for {
		change, open := <-s.observersChanges
		if !open {
			return
		}

		switch change.ChangeVariant {
		case ObserverAdded:
			s.statesRepository.Status().AddObservingAddress(change.RemoteAddr, change.ChannelVariant)
		case ObserverRemoved:
			s.statesRepository.Status().RemoveObservingAddress(change.RemoteAddr, change.ChannelVariant)
		}
	}
}
