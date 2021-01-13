package api

import (
	"fmt"
	"net/http"
	"os"
)

var (
	putDirectoriesFormArgumentsHandlers = map[string]formArgumentHandler{
		pathArg: directoriesPathHandler,
	}
)

type putDirectoriesResponse struct {
	handlerErrors
}

// AddDirectories executes probing of each directory concurrently.
func (s *Server) AddDirectories(directories []string) error {
	for _, directory := range directories {
		info, err := os.Stat(directory)
		if err != nil {
			return err // TODO: directories added before will still be added, so it needs to be refactored for directories to be checked before probing (or aggregate probing errors)
		}

		if !info.IsDir() {
			return fmt.Errorf("%w: %s", ErrPathNotDirectory, directory)
		}

		go s.probeDirectory(directory)
	}

	return nil
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
