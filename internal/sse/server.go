package sse

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/sarpt/mpv-web-api/pkg/api"
	"github.com/sarpt/mpv-web-api/pkg/state"
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
	channels         map[state.SSEChannelVariant]channel
	directories      *state.Directories
	errLog           *log.Logger
	mediaFiles       *state.MediaFiles // TODO: this state passing from the user is very iffy - consider using either callbacks or builder pattern.
	observersChanges chan ObserversChange
	outLog           *log.Logger
	playback         *state.Playback
	playlists        *state.Playlists
	status           *state.Status
}

// Config controls behaviour of the SSE server.
type Config struct {
	ErrWriter io.Writer
	OutWriter io.Writer
}

// NewServer prepares and returns SSE server to handle SSE connections and observers.
func NewServer(cfg Config) *Server {
	return &Server{
		channels:         map[state.SSEChannelVariant]channel{},
		errLog:           log.New(cfg.ErrWriter, logPrefix, log.LstdFlags),
		observersChanges: make(chan ObserversChange),
		outLog:           log.New(cfg.OutWriter, logPrefix, log.LstdFlags),
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
	s.directories = apiServer.Directories()
	s.mediaFiles = apiServer.MediaFiles()
	s.playback = apiServer.Playback()
	s.playlists = apiServer.Playlists()
	s.status = apiServer.Status()

	s.channels[directoriesSSEChannelVariant] = newDirectoriesChannel(s.directories)
	s.channels[mediaFilesSSEChannelVariant] = newMediaFilesChannel(s.mediaFiles)
	s.channels[playbackSSEChannelVariant] = newPlaybackChannel(s.playback)
	s.channels[playlistsSSEChannelVariant] = newPlaylistsChannel(s.playback, s.playlists)
	s.channels[statusSSEChannelVariant] = newStatusChannel(s.status)

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

// subscribeToStateChanges starts listening on state changes channels for further distribution to its observers.
func (s *Server) subscribeToStateChanges() {
	directoriesChannel := s.channels[directoriesSSEChannelVariant].(*directoriesChannel) // TODO: Ooof... Eww... Remove when rewriting with generics
	s.directories.Subscribe(directoriesChannel.BroadcastToChannelObservers, func(err error) {})

	playbackChannel := s.channels[playbackSSEChannelVariant].(*playbackChannel)
	s.playback.Subscribe(playbackChannel.BroadcastToChannelObservers, func(err error) {})

	playlistsChannel := s.channels[playlistsSSEChannelVariant].(*playlistsChannel)
	s.playlists.Subscribe(playlistsChannel.BroadcastToChannelObservers, func(err error) {})

	mediaFilesChannel := s.channels[mediaFilesSSEChannelVariant].(*mediaFilesChannel)
	s.mediaFiles.Subscribe(mediaFilesChannel.BroadcastToChannelObservers, func(err error) {})

	statusChannel := s.channels[statusSSEChannelVariant].(*statusChannel)
	s.status.Subscribe(statusChannel.BroadcastToChannelObservers, func(err error) {})
}

func (s Server) watchSSEObserversChanges() {
	for {
		change, open := <-s.observersChanges
		if !open {
			return
		}

		switch change.ChangeVariant {
		case ObserverAdded:
			s.status.AddObservingAddress(change.RemoteAddr, change.ChannelVariant)
		case ObserverRemoved:
			s.status.RemoveObservingAddress(change.RemoteAddr, change.ChannelVariant)
		}
	}
}
