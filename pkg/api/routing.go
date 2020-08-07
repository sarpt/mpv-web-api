package api

import (
	"errors"
	"net/http"
	"strings"
)

const (
	optionsMethod = "OPTIONS"
	postMethod    = "POST"
	getMethod     = "GET"

	moviesPath      = "/movies"
	playbackPath    = "/playback"
	ssePlaybackPath = "/sse/playback"

	methodsSeparator = ", "
)

var (
	errNoMovieAvailable = errors.New("Movie with specified path does not exist")
)

type pathHandlers map[string]http.HandlerFunc

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

func optionsHandler(allowedMethods []string, res http.ResponseWriter, req *http.Request) {
	allowedMethods = append(allowedMethods, optionsMethod)

	res.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, methodsSeparator))
	res.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Method")
}

func allowedMethods(handlers pathHandlers) []string {
	var allowedMethods []string

	for method := range handlers {
		allowedMethods = append(allowedMethods, method)
	}

	return allowedMethods
}
