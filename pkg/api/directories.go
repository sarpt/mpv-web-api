package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type putDirectoriesResponse struct {
	handlerErrors
}

var (
	postDirectoriesFormArgumentsHandlers = map[string]formArgumentHandler{
		pathArg: directoriesPathHandler,
	}
)

func (s *Server) putDirectoriesHandler(res http.ResponseWriter, req *http.Request) {
	responsePayload := putDirectoriesResponse{}

	args, errors := validateFormRequest(req, postDirectoriesFormArgumentsHandlers) // TODO: directories PUT and movies POST differ only on this line - consider creating abstraction for FORM handlers
	if errors.GeneralError != "" {
		s.errLog.Printf(errors.GeneralError)
		res.WriteHeader(400)
		res.Write([]byte(fmt.Sprintf(responsePayload.GeneralError)))

		return
	}

	responsePayload.ArgumentErrors = errors.ArgumentErrors

	for _, handler := range args {
		err := handler(res, req, s)
		if err != nil {
			responsePayload.GeneralError = err.Error()
			s.errLog.Printf(responsePayload.GeneralError)
			res.WriteHeader(400)
			res.Write([]byte(fmt.Sprintf(responsePayload.GeneralError)))

			return
		}
	}

	out, err := json.Marshal(responsePayload)
	if err != nil {
		responsePayload.GeneralError = fmt.Sprintf("could not encode json payload: %s", err)
		s.errLog.Printf(responsePayload.GeneralError)
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintf(responsePayload.GeneralError)))

		return
	}

	res.WriteHeader(200)
	res.Write([]byte(out))
}

type getDirectoriesResponse struct {
	Directories []string `json:"Directories"`
}

func (s *Server) getDirectoriesHandler(res http.ResponseWriter, req *http.Request) {
	// TODO: to be implemented
}

func (s *Server) deleteDirectoriesHandler(res http.ResponseWriter, req *http.Request) {
	// TODO: to be implemented
}

func directoriesPathHandler(res http.ResponseWriter, req *http.Request, s *Server) error {
	dirPath := req.PostFormValue(pathArg)
	s.outLog.Printf("adding directory %s due to request from %s\n", dirPath, req.RemoteAddr)

	return s.AddDirectories([]string{dirPath})
}
