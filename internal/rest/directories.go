package rest

import (
	"net/http"

	"github.com/sarpt/mpv-web-api/internal/common"
)

type AddDirectoriesHandler = func([]common.Directory) error

func (s *Server) SetAddDirectoriesHandler(handler AddDirectoriesHandler) {
	s.addDirectoriesHandler = handler
}

func (s *Server) getDirectoriesHandler(res http.ResponseWriter, req *http.Request) {
	// TODO: to be implemented
}

func (s *Server) deleteDirectoriesHandler(res http.ResponseWriter, req *http.Request) {
	// TODO: to be implemented
}

func (s *Server) directoriesPathHandler(res http.ResponseWriter, req *http.Request) error {
	dirPath := req.PostFormValue(pathArg)
	s.outLog.Printf("adding directory %s due to request from %s\n", dirPath, req.RemoteAddr)

	return s.addDirectoriesHandler([]common.Directory{
		{
			Path: dirPath,
		},
	})
}

func (s *Server) putDirectoriesFormArgumentsHandlers() map[string]common.FormArgument {
	return map[string]common.FormArgument{
		pathArg: {
			Handle: s.directoriesPathHandler,
		},
	}
}
