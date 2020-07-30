package api

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"

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
	Duration        int
	VideoStreams    []probe.VideoStream
	AudioStreams    []probe.AudioStream
	SubtitleStreams []probe.SubtitleStream
}

// Playback contains information about currently played movie file
type Playback struct {
	Movie       Movie
	Fullscreen  bool
	CurrentTime int
}

// Server is used to serve API and hold state accessible to the API
type Server struct {
	mpvSocketPath     string
	movies            []Movie
	cd                *mpv.CommandDispatcher
	playback          *Playback
	address           string
	allowCors         bool
	playbackChanges   chan Playback
	playbackObservers []chan Playback
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
	cmd := exec.Command(mpvName, idleArg, fmt.Sprintf("%s=%s", inputIpcServerArg, cfg.MpvSocketPath))
	err := cmd.Start()
	if err != nil {
		return &Server{}, fmt.Errorf("could not start mpv binary: %w", err)
	}

	cd, err := mpv.NewCommandDispatcher(cfg.MpvSocketPath)
	if err != nil {
		return &Server{}, err
	}

	movies := moviesInDirectories(cfg.MoviesDirectories)
	playback := &Playback{}

	return &Server{
		cfg.MpvSocketPath,
		movies,
		cd,
		playback,
		cfg.Address,
		cfg.AllowCors,
		make(chan Playback),
		[]chan Playback{},
	}, nil
}

// Serve starts handling requests to the API endpoints. Blocks until canceled
func (s *Server) Serve() error {
	fmt.Fprintf(os.Stdout, "running server at %s\n", s.address)
	serv := http.Server{
		Addr:    s.address,
		Handler: s.mainHandler(),
	}

	err := s.initWatchers()
	if err != nil {
		return errors.New("could not start watching for properties")
	}

	return serv.ListenAndServe()
}

// Close closes underlying command dispatcher
func (s Server) Close() {
	s.cd.Close()
}

func (s *Server) initWatchers() error {
	observeHandlers := map[string]observeHandler{
		mpv.FullscreenProperty:   s.handleFullscreenEvent,
		mpv.PathProperty:         s.handlePathEvent,
		mpv.PlaybackTimeProperty: s.handlePlaybackTimeEvent,
	}

	observeResponses := make(chan mpv.ObserveResponse)
	for _, propertyName := range mpv.ObservableProperties {
		_, err := s.cd.ObserveProperty(propertyName, observeResponses)
		if err != nil {
			return fmt.Errorf("could not initialize watchers due to error when observing property: %w", err)
		}
	}

	go func() {
		for {
			playback, ok := <-s.playbackChanges
			if !ok {
				return
			}

			for _, observer := range s.playbackObservers {
				observer <- playback
			}
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

	return nil
}

func (s *Server) mainHandler() *http.ServeMux {
	ssePlaybackHandlers := map[string]http.HandlerFunc{
		getMethod: s.getSsePlaybackHandler,
	}
	playbackHandlers := map[string]http.HandlerFunc{
		postMethod: s.postPlaybackHandler,
		getMethod:  s.getPlaybackHandler,
	}

	moviesHandlers := map[string]http.HandlerFunc{
		getMethod: s.getMoviesHandler,
	}

	allHandlers := map[string]pathHandlers{
		ssePlaybackPath: ssePlaybackHandlers,
		playbackPath:    playbackHandlers,
		moviesPath:      moviesHandlers,
	}

	mux := http.NewServeMux()
	for path, pathHandlers := range allHandlers {
		mux.HandleFunc(path, s.pathHandler(pathHandlers))
	}

	return mux
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
