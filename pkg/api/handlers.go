package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	optionsMethod = "OPTIONS"
	postMethod    = "POST"
	getMethod     = "GET"

	moviesPath      = "/movies"
	playbackPath    = "/playback"
	ssePlaybackPath = "/sse/playback"

	pathArg       = "path"
	fullscreenArg = "fullscreen"
	subtitleIDArg = "subtitleID"
	audioIDArg    = "audioID"

	methodsSeparator = ", "

	playbackSseEvent = "playback"
)

var (
	errNoMovieAvailable = errors.New("Movie with specified path does not exist")
)

type pathHandlers map[string]http.HandlerFunc

type moviesRespone struct {
	Movies []Movie `json:"movies"`
}

type playbackResponse struct {
	Playback Playback `json:"playback"`
}

func (s *Server) postPlaybackHandler(res http.ResponseWriter, req *http.Request) {
	var out string

	// TODO: this code duplication is getting out of hand, need to refactor this
	subtitleID := req.PostFormValue(subtitleIDArg)
	if subtitleID != "" {
		err := s.mpvManager.ChangeSubtitle(subtitleID)
		if err != nil {
			res.WriteHeader(400)
			res.Write([]byte(fmt.Sprintf("could not successfully change subtitle: %s\n", err)))

			return
		}

		out = fmt.Sprintf("%s\nchanged subtitle to %s\n", out, subtitleID)
	}

	audioID := req.PostFormValue(audioIDArg)
	if audioID != "" {
		err := s.mpvManager.ChangeAudio(audioID)
		if err != nil {
			res.WriteHeader(400)
			res.Write([]byte(fmt.Sprintf("could not successfully change audio: %s\n", err)))

			return
		}

		out = fmt.Sprintf("%s\nchanged audio to %s\n", out, audioID)
	}

	if req.PostFormValue(fullscreenArg) != "" {
		fullscreen, err := strconv.ParseBool(req.PostFormValue(fullscreenArg))
		if err != nil {
			res.WriteHeader(400)
			res.Write([]byte(fmt.Sprintf("invalid fullscreen argument: %s\n", err))) // good enough for poc

			return
		}

		if fullscreen {
			fmt.Fprintf(os.Stdout, "changing fullscreen to %t due to request from %s\n", fullscreen, req.RemoteAddr)

			err := s.mpvManager.ChangeFullscreen(fullscreen)
			if err != nil {
				res.WriteHeader(400)
				res.Write([]byte(fmt.Sprintf("could not successfully change fullscreen: %s\n", err))) // good enough for poc

				return
			}

			out = fmt.Sprintf("%s\nchanged fullscreen to %t\n", out, fullscreen)
		}
	}

	filePath := req.PostFormValue(pathArg)
	if filePath != "" {
		fmt.Fprintf(os.Stdout, "playing file '%s' due to request from %s\n", filePath, req.RemoteAddr)
		err := s.mpvManager.LoadFile(filePath)
		if err != nil {
			res.WriteHeader(400)
			res.Write([]byte(fmt.Sprintf("could not successfully load the file: %s\n", err))) // good enough for poc

			return
		}

		out = fmt.Sprintf("%s\nplayback of file %s started\n", out, filePath)
	}

	res.WriteHeader(200)
	res.Write([]byte(out))
}

func (s *Server) getPlaybackHandler(res http.ResponseWriter, req *http.Request) {
	playbackResponse := playbackResponse{
		Playback: *s.playback,
	}

	response, err := json.Marshal(&playbackResponse)
	if err != nil {
		res.WriteHeader(400)
		res.Write([]byte(fmt.Sprintf("could not prepare output: %s\n", err))) // good enough for poc

		return
	}
	res.WriteHeader(200)
	res.Write(response)
}

func (s *Server) getSsePlaybackHandler(res http.ResponseWriter, req *http.Request) {
	flusher, ok := res.(http.Flusher)
	if !ok {
		res.WriteHeader(400)
		return
	}
	res.Header().Set("Connection", "keep-alive")
	res.Header().Set("Content-Type", "text/event-stream")
	res.Header().Set("Access-Control-Allow-Origin", "*")

	playbackChanges := make(chan Playback)
	s.playbackObservers = append(s.playbackObservers, playbackChanges)

	for {
		select {
		case playback, ok := <-playbackChanges:
			if !ok {
				return
			}

			out, err := json.Marshal(playback)
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not write to the client")
			}

			_, err = res.Write(formatSseEvent(playbackSseEvent, out))
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not write to the client")
			}

			flusher.Flush()
		case <-req.Context().Done():
			return
		}
	}
}

func (s *Server) getMoviesHandler(res http.ResponseWriter, req *http.Request) {
	moviesResponse := moviesRespone{
		Movies: s.movies,
	}

	response, err := json.Marshal(&moviesResponse)
	if err != nil {
		res.WriteHeader(400)
		res.Write([]byte(fmt.Sprintf("could not prepare output: %s\n", err))) // good enough for poc

		return
	}

	res.WriteHeader(200)
	res.Write(response)
}

func optionsHandler(allowedMethods []string, res http.ResponseWriter, req *http.Request) {
	allowedMethods = append(allowedMethods, optionsMethod)

	res.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, methodsSeparator))
	res.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Method")
}

func (s *Server) pathHandler(handlers pathHandlers) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if s.allowCors {
			res.Header().Set("Access-Control-Allow-Origin", "*")
		}

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

func (s Server) movieByPath(path string) (Movie, error) {
	for _, movie := range s.movies {
		if movie.Path == path {
			return movie, nil
		}
	}

	return Movie{}, errNoMovieAvailable
}

func allowedMethods(handlers pathHandlers) []string {
	var allowedMethods []string

	for method := range handlers {
		allowedMethods = append(allowedMethods, method)
	}

	return allowedMethods
}
