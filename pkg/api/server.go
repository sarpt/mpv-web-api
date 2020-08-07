package api

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/mpv"
	"github.com/sarpt/mpv-web-api/pkg/probe"
)

const (
	mpvName           = "mpv"
	idleArg           = "--idle"
	inputIpcServerArg = "--input-ipc-server"
)

type observeHandler = func(res mpv.ObserveResponse) error

// Movie specifies information about a movie file that can be played
type Movie struct {
	Path            string
	Duration        float64
	VideoStreams    []probe.VideoStream
	AudioStreams    []probe.AudioStream
	SubtitleStreams []probe.SubtitleStream
}

// Playback contains information about currently played movie file
type Playback struct {
	Movie       Movie
	Fullscreen  bool
	CurrentTime float64
}

// Server is used to serve API and hold state accessible to the API
type Server struct {
	mpvSocketPath         string
	movies                []Movie
	mpvManager            mpv.Manager
	playback              *Playback
	address               string
	allowCors             bool
	playbackChanges       chan Playback
	playbackObservers     map[string]chan Playback
	playbackObserversLock *sync.RWMutex
}

// Config controls behaviour of the api serve
type Config struct {
	Address           string
	MoviesDirectories []string
	MpvSocketPath     string
	AllowCors         bool
}

// NewServer prepares and returns a server that can be used to handle API
func NewServer(cfg Config) (*Server, error) {
	// TODO: move this to mpv.Manager
	cmd := exec.Command(mpvName, idleArg, fmt.Sprintf("%s=%s", inputIpcServerArg, cfg.MpvSocketPath))
	err := cmd.Start()
	if err != nil {
		return &Server{}, fmt.Errorf("could not start mpv binary: %w", err)
	}

	mpvManager, err := mpv.NewManager(cfg.MpvSocketPath)
	if err != nil {
		return &Server{}, err
	}

	movies := moviesInDirectories(cfg.MoviesDirectories)
	playback := &Playback{}

	return &Server{
		cfg.MpvSocketPath,
		movies,
		mpvManager,
		playback,
		cfg.Address,
		cfg.AllowCors,
		make(chan Playback),
		map[string]chan Playback{},
		&sync.RWMutex{},
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

	fmt.Fprintf(os.Stdout, "running server at %s\n", s.address)
	return serv.ListenAndServe()
}

// Close closes server, along with closing necessary helpers
func (s Server) Close() {
	s.mpvManager.Close()
}

func (s *Server) initWatchers() error {
	observeResponses := make(chan mpv.ObserveResponse)
	observeHandlers := map[string]observeHandler{
		mpv.FullscreenProperty:   s.handleFullscreenEvent,
		mpv.PathProperty:         s.handlePathEvent,
		mpv.PlaybackTimeProperty: s.handlePlaybackTimeEvent,
	}

	go func() {
		for {
			playback, ok := <-s.playbackChanges
			if !ok {
				return
			}

			s.playbackObserversLock.RLock()
			for _, observer := range s.playbackObservers {
				observer <- playback
			}
			s.playbackObserversLock.RUnlock()
		}
	}()

	go func() {
		for {
			observeResponse, open := <-observeResponses
			if !open {
				return
			}

			observeHandler, ok := observeHandlers[observeResponse.Property]
			if !ok {
				continue
			}

			err := observeHandler(observeResponse)
			if err != nil {
				fmt.Fprintf(os.Stdout, "could not handle property '%s' observer handling: %s\n", observeResponse.Property, err)
			}
			s.playbackChanges <- *s.playback
		}
	}()

	for _, propertyName := range mpv.ObservableProperties {
		_, err := s.mpvManager.ObserveProperty(propertyName, observeResponses)
		if err != nil {
			return fmt.Errorf("could not initialize watchers due to error when observing property: %w", err)
		}
	}

	return nil
}

func (s Server) movieByPath(path string) (Movie, error) {
	for _, movie := range s.movies {
		if movie.Path == path {
			return movie, nil
		}
	}

	return Movie{}, errNoMovieAvailable
}

func formatSseEvent(eventName string, data []byte) []byte {
	var out []byte

	out = append(out, []byte(fmt.Sprintf("event:%s\n", eventName))...)

	dataEntries := bytes.Split(data, []byte("\n"))
	for _, dataEntry := range dataEntries {
		out = append(out, []byte(fmt.Sprintf("data:%s\n", dataEntry))...)
	}

	out = append(out, []byte("\n\n")...)
	return out
}
