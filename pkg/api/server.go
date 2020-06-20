package api

import (
	"fmt"
	"net/http"
	"os"

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

// Server is used to serve API and hold state accessible to the API
type Server struct {
	mpvSocketPath string
	movies        []Movie
	cd            *mpv.CommandDispatcher
}

// NewServer prepares and returns a server that can be used to handle API
func NewServer(moviesDirectories []string, mpvSocketPath string) (Server, error) {
	cd, err := mpv.NewCommandDispatcher(mpvSocketPath)
	if err != nil {
		return Server{}, err
	}

	movies := moviesInDirectories(moviesDirectories)

	return Server{
		mpvSocketPath,
		movies,
		cd,
	}, nil
}

// Serve starts handling requests to the API endpoints. Blocks until canceled
func (s Server) Serve() error {
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

func (s Server) mainHandler() *http.ServeMux {
	playbackHandlers := map[string]http.HandlerFunc{
		postMethod: s.playbackHandler,
	}

	videosHandlers := map[string]http.HandlerFunc{
		getMethod: s.videosHandler,
	}

	allHandlers := map[string]pathHandlers{
		playbackPath: playbackHandlers,
		moviesPath:   videosHandlers,
	}

	mux := http.NewServeMux()
	for path, pathHandlers := range allHandlers {
		mux.HandleFunc(path, pathHandler(pathHandlers))
	}

	return mux
}
