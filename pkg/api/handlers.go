package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/sarpt/mpv-web-api/pkg/mpv"
)

const (
	optionsMethod = "OPTIONS"
	postMethod    = "POST"
	getMethod     = "GET"

	moviesPath   = "/movies"
	playbackPath = "/playback"

	pathArg       = "path"
	fullscreenArg = "fullscreen"

	methodsSeparator = ", "
)

type pathHandlers map[string]http.HandlerFunc

type moviesRespone struct {
	Movies []Movie `json:"movies"`
}

func (s Server) playbackHandler(res http.ResponseWriter, req *http.Request) {
	var out string

	fullscreen, err := strconv.ParseBool(req.PostFormValue(fullscreenArg))
	if err != nil {
		res.WriteHeader(400)
		res.Write([]byte(fmt.Sprintf("invalid fullscreen argument: %s\n", err))) // good enough for poc

		return
	}

	if fullscreen {
		fmt.Fprintf(os.Stdout, "changing fullscreen to %t due to request from %s\n", fullscreen, req.RemoteAddr)
		result, err := s.cd.Dispatch(mpv.NewFullscreen(fullscreen))
		if err != nil || result.Err != "success" {
			res.WriteHeader(400)
			res.Write([]byte(fmt.Sprintf("could not successfully change fullscreen: %s\n", err))) // good enough for poc

			return
		}

		out = fmt.Sprintf("%s\nchanged fullscreen to %t\n", out, fullscreen)
	}

	filePath := req.PostFormValue(pathArg)
	if filePath != "" {
		fmt.Fprintf(os.Stdout, "playing file '%s' due to request from %s\n", filePath, req.RemoteAddr)
		result, err := s.cd.Dispatch(mpv.NewLoadFile(filePath))
		if err != nil || result.Err != "success" {
			res.WriteHeader(400)
			res.Write([]byte(fmt.Sprintf("could not successfully load the file: %s\n", err))) // good enough for poc

			return
		}

		out = fmt.Sprintf("%s\nplayback of file %s started\n", out, filePath)
	}

	res.WriteHeader(200)
	res.Write([]byte(out))
}

func (s Server) moviesHandler(res http.ResponseWriter, req *http.Request) {
	moviesResponse := moviesRespone{
		Movies: s.movies,
	}

	respone, err := json.Marshal(&moviesResponse)
	if err != nil {
		res.WriteHeader(400)
		res.Write([]byte(fmt.Sprintf("could not prepare output: %s\n", err))) // good enough for poc

		return
	}

	res.WriteHeader(200)
	res.Write(respone)
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
