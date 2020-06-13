package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/sarpt/mpv-web-api/pkg/mpv"
)

const (
	address = "localhost:3001"
)

// Server is used to serve API and hold state accessible to the API
type Server struct {
	mpvSocketPath string
	videosPaths   []string
	cd            *mpv.CommandDispatcher
}

// NewServer prepares and returns a server that can be used to handle API
func NewServer(videosPaths []string, mpvSocketPath string) (Server, error) {
	cd, err := mpv.NewCommandDispatcher(mpvSocketPath)
	if err != nil {
		return Server{}, err
	}

	return Server{
		mpvSocketPath,
		videosPaths,
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

	allHandlers := map[string]pathHandlers{
		playbackPath: playbackHandlers,
	}

	mux := http.NewServeMux()
	for path, pathHandlers := range allHandlers {
		mux.HandleFunc(path, pathHandler(pathHandlers))
	}

	return mux
}
