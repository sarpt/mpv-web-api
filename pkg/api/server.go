package api

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
)

var handlers = map[string]http.HandlerFunc{
	"/playback": playbackHandler,
}

// Serve starts handling requests to the API endpoints
// TODO: move starting of the command outside of the serve
func Serve() error {
	cmd := exec.Command("mpv", "--idle", "--input-ipc-server=/tmp/mpvsocket")
	err := cmd.Start()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "running server at localhost:3001\n")
	s := http.Server{
		Addr:    "localhost:3001",
		Handler: mainHandler(),
	}

	return s.ListenAndServe()
}

func mainHandler() *http.ServeMux {
	mux := http.NewServeMux()
	for path, handler := range handlers {
		mux.HandleFunc(path, handler)
	}

	return mux
}
