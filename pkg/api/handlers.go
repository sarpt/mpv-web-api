package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/sarpt/mpv-web-api/pkg/mpv"
)

// TODO: CommandDispatcher should be scoped by server, not per request
// TODO: Options handling should be generic and declarative (without gorilla router if possible)
// TODO: Handlers should dispatch commands in form of already wrapped and typed commands, instead of raw commands with string arrays
func playbackHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method == "OPTIONS" {
		playbackOptionsHandler(res, req)

		return
	}

	filePath := req.PostFormValue("path")
	if filePath == "" {
		res.WriteHeader(400)
		res.Write([]byte(fmt.Sprintf("empty path in form\n"))) // good enough for poc

		return
	}

	cd, err := mpv.NewCommandDispatcher("/tmp/mpvsocket")
	if err != nil {
		res.WriteHeader(400)
		res.Write([]byte(fmt.Sprintf("cannot create command dispatcher: %s\n", err))) // good enough for poc

		return
	}
	defer cd.Close()

	fmt.Fprintf(os.Stdout, "playing the file '%s' on request from %s\n", filePath, req.RemoteAddr)
	result, err := cd.Dispatch([]string{"loadfile", filePath})
	if err != nil {
		res.WriteHeader(400)
		res.Write([]byte(fmt.Sprintf("could not successfully load the file: %s\n", err))) // good enough for poc

		return
	}

	out := fmt.Sprintf("Playing of file %s ended with status %s\n", filePath, result.Err)
	res.WriteHeader(200)
	res.Write([]byte(out))
}

func playbackOptionsHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	res.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Method")
}
