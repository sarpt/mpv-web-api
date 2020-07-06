package api

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/mpv"
	"github.com/sarpt/mpv-web-api/pkg/probe"
)

const (
	address = "localhost:3001"
)

// Movie specifies information about a movie file that can be played
type Movie struct {
	Path            string
	VideoStreams    []probe.VideoStream
	AudioStreams    []probe.AudioStream
	SubtitleStreams []probe.SubtitleStream
}

// Playback contains information about currently played movie file
type Playback struct {
	movie      Movie
	fullscreen bool
}

// Server is used to serve API and hold state accessible to the API
type Server struct {
	mpvSocketPath string
	movies        []Movie
	cd            *mpv.CommandDispatcher
	playback      *Playback
	playbackLock  *sync.RWMutex
}

// NewServer prepares and returns a server that can be used to handle API
func NewServer(moviesDirectories []string, mpvSocketPath string) (*Server, error) {
	cd, err := mpv.NewCommandDispatcher(mpvSocketPath)
	if err != nil {
		return &Server{}, err
	}

	movies := moviesInDirectories(moviesDirectories)
	playback := &Playback{}
	playbackLock := &sync.RWMutex{}

	return &Server{
		mpvSocketPath,
		movies,
		cd,
		playback,
		playbackLock,
	}, nil
}

// Serve starts handling requests to the API endpoints. Blocks until canceled
func (s *Server) Serve() error {
	fmt.Fprintf(os.Stdout, "running server at %s\n", address)
	serv := http.Server{
		Addr:    address,
		Handler: s.mainHandler(),
	}

	return serv.ListenAndServe()
}

// Close closes underlying command dispatcher
func (s Server) Close() {
	s.cd.Close()
}

func (s *Server) mainHandler() *http.ServeMux {
	playbackHandlers := map[string]http.HandlerFunc{
		postMethod: s.postPlaybackHandler,
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
