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
	Directories *state.Directories
	ErrWriter   io.Writer
	MediaFiles  *state.MediaFiles
	OutWriter   io.Writer
	Playback    *state.Playback
	Playlists   *state.Playlists
	Status      *state.Status
}

// NewServer prepares and returns SSE server to handle SSE connections and observers.
func NewServer(cfg Config) *Server {
	return &Server{
		channels: map[state.SSEChannelVariant]channel{
			directoriesSSEChannelVariant: newDirectoriesChannel(cfg.Directories),
			mediaFilesSSEChannelVariant:  newMediaFilesChannel(cfg.MediaFiles),
			playbackSSEChannelVariant:    newPlaybackChannel(cfg.Playback),
			playlistsSSEChannelVariant:   newPlaylistsChannel(cfg.Playback, cfg.Playlists),
			statusSSEChannelVariant:      newStatusChannel(cfg.Status),
		},
		directories:      cfg.Directories,
		errLog:           log.New(cfg.ErrWriter, logPrefix, log.LstdFlags),
		mediaFiles:       cfg.MediaFiles,
		observersChanges: make(chan ObserversChange),
		outLog:           log.New(cfg.OutWriter, logPrefix, log.LstdFlags),
		playback:         cfg.Playback,
		playlists:        cfg.Playlists,
		status:           cfg.Status,
	}
}

// SubscribeToStateChanges starts listening on state changes channels for further distribution to its observers.
func (s *Server) SubscribeToStateChanges() {
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

// Handler returns map of HTTPs methods and their handlers.
func (s *Server) Handler() http.Handler {
	sseCfg := handlerConfig{
		Channels: s.channels,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(registerPath, s.createSseRegisterHandler(sseCfg))

	return mux
}

func (s Server) WatchSSEObserversChanges() {
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
