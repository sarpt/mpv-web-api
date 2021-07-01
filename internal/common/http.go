package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
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

type FormArgumentHandler func(http.ResponseWriter, *http.Request) error
type FormArgumentValidator func(*http.Request) error
type FormArgument struct {
	Handle   FormArgumentHandler
	Validate FormArgumentValidator
}
type FormResponse struct {
	HandlerErrors
	Payload
}

type HandlerErrors struct {
	ArgumentErrors map[string]string `json:"argumentErrors"`
	GeneralError   string            `json:"generalError"`
}

type Payload interface{}

// MethodHandlers specifiy map between http method and respective handler function.
type MethodHandlers map[string]http.HandlerFunc

// PathHandlerConfig specifies per-path behavior for path handling middleware.
type PathHandlerConfig struct {
	MethodHandlers
	AllowCORS bool
}

// PathHandler returns a function acting as a middleware before handling specified path.
func PathHandler(cfg PathHandlerConfig) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if cfg.AllowCORS {
			res.Header().Set(accessControlAllowOriginHeader, allowedOrigins)
		}

		method := req.Method
		if method == http.MethodOptions {
			optionsHandler(allowedMethods(cfg.MethodHandlers), res, req)

			return
		}

		if method == http.MethodHead {
			_, ok := cfg.MethodHandlers[http.MethodGet]
			if !ok {
				res.WriteHeader(404)

				return
			}

			res.WriteHeader(200)
			return
		}

		handler, ok := cfg.MethodHandlers[method]
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

func allowedMethods(handlers MethodHandlers) []string {
	var allowedMethods []string

	for method := range handlers {
		allowedMethods = append(allowedMethods, method)
	}

	return allowedMethods
}

// CreateFormHandler returns handler function responsible for correct validation and routing of arguments to their handlers.
func CreateFormHandler(allArgHandlers map[string]FormArgument) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		responsePayload := FormResponse{}

		selectedArgHandlers, errors := validateFormRequest(req, allArgHandlers)
		responsePayload.GeneralError = errors.GeneralError
		responsePayload.ArgumentErrors = errors.ArgumentErrors

		if responsePayload.GeneralError != "" || len(responsePayload.ArgumentErrors) != 0 {
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
			err := handler(res, req)
			if err != nil {
				responsePayload.GeneralError = err.Error()
				out, _ := prepareJSONOutput(responsePayload)
				res.WriteHeader(500)
				res.Write(out)

				return
			}
		}

		out, err := prepareJSONOutput(responsePayload)
		if err != nil {
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
func validateFormRequest(req *http.Request, arguments map[string]FormArgument) ([]FormArgumentHandler, HandlerErrors) {
	correctHandlers := []FormArgumentHandler{}
	handlerErrors := HandlerErrors{
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
			handlerErrors.ArgumentErrors[argName] = fmt.Sprintf("the %s argument handler is not defined", argName)
			continue
		}

		var validateErr error = nil
		if argument.Validate != nil {
			validateErr = argument.Validate(req)
		}

		if validateErr != nil {
			handlerErrors.ArgumentErrors[argName] = fmt.Sprintf("the %s argument is invalid: %s", argName, validateErr)
			continue
		}

		if argument.Handle == nil {
			continue
		}

		correctHandlers = append(correctHandlers, argument.Handle)
	}

	return correctHandlers, handlerErrors
}

func multipartFormRequest(req *http.Request) bool {
	contentType, ok := req.Header[contentTypeHeader]

	return ok && len(contentType) > 0 && strings.Contains(contentType[0], multiPartFormContentType)
}

func prepareJSONOutput(res FormResponse) ([]byte, error) {
	out, err := json.Marshal(res)
	if err != nil {
		return []byte(fmt.Sprintf("could not encode json payload: %s", err)), err
	}

	return out, nil
}
