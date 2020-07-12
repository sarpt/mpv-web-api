package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/sarpt/mpv-web-api/pkg/mpv"
	"github.com/sarpt/mpv-web-api/pkg/probe"
)

const (
	address = "localhost:3001"
)

type observeHandler = func(res mpv.ObserveResponse) error

// Movie specifies information about a movie file that can be played
type Movie struct {
	Path            string
	VideoStreams    []probe.VideoStream
	AudioStreams    []probe.AudioStream
	SubtitleStreams []probe.SubtitleStream
}

// Playback contains information about currently played movie file
type Playback struct {
	Movie      Movie
	Fullscreen bool
}

// Server is used to serve API and hold state accessible to the API
type Server struct {
	mpvSocketPath string
	movies        []Movie
	cd            *mpv.CommandDispatcher
	playback      *Playback
}

// NewServer prepares and returns a server that can be used to handle API
func NewServer(moviesDirectories []string, mpvSocketPath string) (*Server, error) {
	cd, err := mpv.NewCommandDispatcher(mpvSocketPath)
	if err != nil {
		return &Server{}, err
	}

	movies := moviesInDirectories(moviesDirectories)
	playback := &Playback{}

	return &Server{
		mpvSocketPath,
		movies,
		cd,
		playback,
	}, nil
}

// Serve starts handling requests to the API endpoints. Blocks until canceled
func (s *Server) Serve() error {
	fmt.Fprintf(os.Stdout, "running server at %s\n", address)
	serv := http.Server{
		Addr:    address,
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
		mpv.FullscreenProperty: s.handleFullscreenEvent,
		mpv.PathProperty:       s.handlePathEvent,
	}

	observeResponses := make(chan mpv.ObserveResponse)
	_, err := s.cd.ObserveProperty(mpv.FullscreenProperty, observeResponses)
	if err != nil {
		return err
	}

	_, err = s.cd.ObserveProperty(mpv.PathProperty, observeResponses)
	if err != nil {
		return err
	}

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
		}
	}()

	return nil
}

func (s *Server) handleFullscreenEvent(res mpv.ObserveResponse) error {
	enabled, ok := res.Data.(string)
	if !ok {
		return errors.New("could not decode data for fullscreen change event")
	}

	s.playback.Fullscreen = enabled == mpv.FullscreenEnabled
	return nil
}

func (s *Server) handlePathEvent(res mpv.ObserveResponse) error {
	if res.Data == nil {
		s.playback.Movie = Movie{}
		return nil
	}

	path, ok := res.Data.(string)
	if !ok {
		return errors.New("could not decode data for path change event")
	}

	movie, err := s.movieByPath(path)
	if err != nil {
		return fmt.Errorf("could not retrieve movie by path %s", path)
	}

	s.playback.Movie = movie
	return nil
}

func (s *Server) mainHandler() *http.ServeMux {
	playbackHandlers := map[string]http.HandlerFunc{
		postMethod: s.postPlaybackHandler,
		getMethod:  s.getPlaybackHandler,
	}

	moviesHandlers := map[string]http.HandlerFunc{
		getMethod: s.getMoviesHandler,
	}

	allHandlers := map[string]pathHandlers{
		playbackPath: playbackHandlers,
		moviesPath:   moviesHandlers,
	}

	mux := http.NewServeMux()
	for path, pathHandlers := range allHandlers {
		mux.HandleFunc(path, pathHandler(pathHandlers))
	}

	return mux
}
