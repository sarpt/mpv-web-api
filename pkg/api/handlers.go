package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	optionsMethod = "OPTIONS"
	postMethod    = "POST"
	getMethod     = "GET"

	videosPath   = "/videos"
	playbackPath = "/playback"

	pathArg = "path"

	methodsSeparator = ", "
)

type pathHandlers map[string]http.HandlerFunc

type videosResponse struct {
	Videos []string `json:"videos"`
}

// TODO: Handlers should dispatch commands in form of already wrapped and typed commands, instead of raw commands with string arrays
func (s Server) playbackHandler(res http.ResponseWriter, req *http.Request) {
	filePath := req.PostFormValue(pathArg)
	if filePath == "" {
		res.WriteHeader(400)
		res.Write([]byte(fmt.Sprintf("empty path in form\n"))) // good enough for poc

		return
	}

	fmt.Fprintf(os.Stdout, "playing the file '%s' on request from %s\n", filePath, req.RemoteAddr)
	result, err := s.cd.Dispatch([]string{"loadfile", filePath})
	if err != nil {
		res.WriteHeader(400)
		res.Write([]byte(fmt.Sprintf("could not successfully load the file: %s\n", err))) // good enough for poc

		return
	}

	out := fmt.Sprintf("Playing of file %s ended with status %s\n", filePath, result.Err)
	res.WriteHeader(200)
	res.Write([]byte(out))
}

func (s Server) videosHandler(res http.ResponseWriter, req *http.Request) {
	videosResponse := videosResponse{
		Videos: []string{},
	}

	for _, videosPath := range s.videosPaths {
		filepath.Walk(videosPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				videosResponse.Videos = append(videosResponse.Videos, path)
			}

			return nil
		})
	}

	out, err := json.Marshal(&videosResponse)
	if err != nil {
		res.WriteHeader(400)
		res.Write([]byte(fmt.Sprintf("could not prepare output: %s\n", err))) // good enough for poc

		return
	}

	res.WriteHeader(200)
	res.Write(out)
}

func optionsHandler(allowedMethods []string, res http.ResponseWriter, req *http.Request) {
	allowedMethods = append(allowedMethods, optionsMethod)

	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, methodsSeparator))
	res.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Method")
}

func pathHandler(handlers pathHandlers) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		method := req.Method
		if method == optionsMethod {
			optionsHandler(allowedMethods(handlers), res, req)

			return
		}

		handler, ok := handlers[method]

		if !ok {
			res.WriteHeader(404)

			return
		}

		handler(res, req)
	}
}

func allowedMethods(handlers pathHandlers) []string {
	var allowedMethods []string

	for method := range handlers {
		allowedMethods = append(allowedMethods, method)
	}

	return allowedMethods
}
