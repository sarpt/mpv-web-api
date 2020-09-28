package api

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	moviesPath      = "/movies"
	directoriesPath = "/directories"
	playbackPath    = "/playback"
	ssePlaybackPath = "/sse/playback"
	sseMoviesPath   = "/sse/movies"

	methodsSeparator = ", "

	multiPartFormMaxMemory = 32 << 20
)

type pathHandlers map[string]http.HandlerFunc
type formArgumentHandler func(http.ResponseWriter, *http.Request, *Server) error

type handlerErrors struct {
	ArgumentErrors map[string]string `json:"argumentErrors"`
	GeneralError   string            `json:"generalError"`
}

func (s *Server) mainHandler() *http.ServeMux {
	ssePlaybackHandlers := map[string]http.HandlerFunc{
		http.MethodGet: s.getSsePlaybackHandler,
	}

	sseMoviesHandlers := map[string]http.HandlerFunc{
		http.MethodGet: s.getSseMoviesHandler,
	}

	playbackHandlers := map[string]http.HandlerFunc{
		http.MethodPost: s.postPlaybackHandler,
		http.MethodGet:  s.getPlaybackHandler,
	}

	moviesHandlers := map[string]http.HandlerFunc{
		http.MethodGet: s.getMoviesHandler,
	}

	directoriesHandlers := map[string]http.HandlerFunc{
		http.MethodGet:    s.getDirectoriesHandler,
		http.MethodPut:    s.putDirectoriesHandler,
		http.MethodDelete: s.deleteDirectoriesHandler,
	}

	allHandlers := map[string]pathHandlers{
		ssePlaybackPath: ssePlaybackHandlers,
		sseMoviesPath:   sseMoviesHandlers,
		playbackPath:    playbackHandlers,
		moviesPath:      moviesHandlers,
		directoriesPath: directoriesHandlers,
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
		if method == http.MethodOptions {
			optionsHandler(allowedMethods(handlers), res, req)

			return
		}

		if method == http.MethodHead {
			_, ok := handlers[http.MethodGet]
			if !ok {
				res.WriteHeader(404)

				return
			}

			res.WriteHeader(200) // TODO: parameters validation
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
	allowedMethods = append(allowedMethods, http.MethodOptions)

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

// validateFormRequest checks form body for arguments and their correctnes.
// Result of validation is an array of arguments that have handlers associated and handlerErrors (if any occured).
func validateFormRequest(req *http.Request, handlers map[string]formArgumentHandler) ([]formArgumentHandler, handlerErrors) {
	correctHandlers := []formArgumentHandler{}
	handlerErrors := handlerErrors{
		ArgumentErrors: map[string]string{},
	}

	var err error
	contentType, ok := req.Header["Content-Type"]
	if !ok || len(contentType) < 1 || !strings.Contains(contentType[0], "multipart/form-data") {
		err = req.ParseForm()
	} else {
		err = req.ParseMultipartForm(multiPartFormMaxMemory)
	}

	if err != nil {
		handlerErrors.GeneralError = fmt.Sprintf("could not parse form data: %s", err)

		return correctHandlers, handlerErrors
	}

	for arg := range req.PostForm {
		handler, ok := handlers[arg]
		if !ok {
			handlerErrors.ArgumentErrors[arg] = fmt.Sprintf("the %s argument is invalid", arg)
			continue
		} else {
			correctHandlers = append(correctHandlers, handler)
		}
	}

	return correctHandlers, handlerErrors
}
