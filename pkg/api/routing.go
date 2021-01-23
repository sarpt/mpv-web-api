package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	moviesPath      = "/movies"
	directoriesPath = "/directories"
	playbackPath    = "/playback"
	ssePath         = "/sse"

	methodsSeparator = ", "

	multiPartFormMaxMemory   = 32 << 20
	multiPartFormContentType = "multipart/form-data"

	accessControlAllowOriginHeader  = "Access-Control-Allow-Origin"
	accessControlAllowMethodsHeader = "Access-Control-Allow-Methods"
	accessControlAllowHeadersHeader = "Access-Control-Allow-Headers"
	contentTypeHeader               = "Content-Type"

	allowedOrigins = "*"
	allowedHeaders = "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Method"
)

type pathHandlers map[string]http.HandlerFunc
type formArgumentHandler func(http.ResponseWriter, *http.Request, *Server) error
type formArgumentValidator func(*http.Request) bool
type formArgument struct {
	handle   formArgumentHandler
	validate formArgumentValidator
}
type payload interface{}
type formResponse struct {
	handlerErrors
	payload
}

type handlerErrors struct {
	ArgumentErrors map[string]string `json:"argumentErrors"`
	GeneralError   string            `json:"generalError"`
}

func (s *Server) mainHandler() *http.ServeMux {
	sseCfg := SSEHandlerConfig{
		Channels: map[SSEChannelVariant]SSEChannel{
			playbackSSEChannelVariant: s.playbackSSEChannel(),
			moviesSSEChannelVariant:   s.moviesSSEChannel(),
			statusSSEChannelVariant:   s.statusSSEChannel(),
		},
	}
	sseHandlers := map[string]http.HandlerFunc{
		http.MethodGet: s.createGetSseHandler(sseCfg),
	}

	playbackHandlers := map[string]http.HandlerFunc{
		http.MethodPost: s.createFormHandler(postPlaybackFormArgumentsHandlers),
		http.MethodGet:  s.getPlaybackHandler,
	}

	moviesHandlers := map[string]http.HandlerFunc{
		http.MethodGet: s.getMoviesHandler,
	}

	directoriesHandlers := map[string]http.HandlerFunc{
		http.MethodGet:    s.getDirectoriesHandler,
		http.MethodPut:    s.createFormHandler(putDirectoriesFormArgumentsHandlers),
		http.MethodDelete: s.deleteDirectoriesHandler,
	}

	allHandlers := map[string]pathHandlers{
		ssePath:         sseHandlers,
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
			res.Header().Set(accessControlAllowOriginHeader, allowedOrigins)
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

	res.Header().Set(accessControlAllowMethodsHeader, strings.Join(allowedMethods, methodsSeparator))
	res.Header().Set(accessControlAllowHeadersHeader, allowedHeaders)
}

func allowedMethods(handlers pathHandlers) []string {
	var allowedMethods []string

	for method := range handlers {
		allowedMethods = append(allowedMethods, method)
	}

	return allowedMethods
}

func (s *Server) createFormHandler(allArgHandlers map[string]formArgument) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		responsePayload := formResponse{}

		selectedArgHandlers, errors := validateFormRequest(req, allArgHandlers)
		responsePayload.GeneralError = errors.GeneralError
		responsePayload.ArgumentErrors = errors.ArgumentErrors

		if responsePayload.GeneralError != "" || len(responsePayload.ArgumentErrors) != 0 {
			s.errLog.Printf("invalid request from %s\n", req.RemoteAddr)
			out, err := prepareJSONOutput(responsePayload)
			if err != nil {
				res.WriteHeader(400)
			} else {
				res.WriteHeader(500)
			}
			res.Write(out)

			return
		}

		for _, handler := range selectedArgHandlers {
			err := handler(res, req, s)
			if err != nil {
				responsePayload.GeneralError = err.Error()
				s.errLog.Printf(responsePayload.GeneralError)
				out, _ := prepareJSONOutput(responsePayload)
				res.WriteHeader(500)
				res.Write(out)

				return
			}
		}

		out, err := prepareJSONOutput(responsePayload)
		if err != nil {
			s.errLog.Printf("%s", out)
			res.WriteHeader(500)
			res.Write(out)

			return
		}

		res.WriteHeader(200)
		res.Write(out)
	}
}

// validateFormRequest checks form body for arguments and their correctnes.
// Result of validation is an array of arguments that have handlers associated and handlerErrors (if any occured).
func validateFormRequest(req *http.Request, arguments map[string]formArgument) ([]formArgumentHandler, handlerErrors) {
	correctHandlers := []formArgumentHandler{}
	handlerErrors := handlerErrors{
		ArgumentErrors: map[string]string{},
	}

	var err error
	if multipartFormRequest(req) {
		err = req.ParseMultipartForm(multiPartFormMaxMemory)
	} else {
		err = req.ParseForm()
	}

	if err != nil {
		handlerErrors.GeneralError = fmt.Sprintf("could not parse form data: %s", err)

		return correctHandlers, handlerErrors
	}

	for argName := range req.PostForm {
		argument, ok := arguments[argName]
		if !ok {
			handlerErrors.ArgumentErrors[argName] = fmt.Sprintf("the %s argument is not defined", argName)
			continue
		}

		if argument.validate != nil && !argument.validate(req) {
			handlerErrors.ArgumentErrors[argName] = fmt.Sprintf("the %s argument is invalid", argName)
			continue
		}

		if argument.handle == nil {
			continue
		}

		correctHandlers = append(correctHandlers, argument.handle)
	}

	return correctHandlers, handlerErrors
}

func multipartFormRequest(req *http.Request) bool {
	contentType, ok := req.Header[contentTypeHeader]

	return ok && len(contentType) > 0 && strings.Contains(contentType[0], multiPartFormContentType)
}

func prepareJSONOutput(res formResponse) ([]byte, error) {
	out, err := json.Marshal(res)
	if err != nil {
		return []byte(fmt.Sprintf("could not encode json payload: %s", err)), err
	}

	return out, nil
}
