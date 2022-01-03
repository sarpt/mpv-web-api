package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/sarpt/mpv-web-api/internal/common"
	"github.com/sarpt/mpv-web-api/pkg/state"
)

const (
	watchedArg = "watched"
)

type (
	addDirectoriesCb    = func([]state.Directory)
	loadPlaylistCb      = func(string, bool) error
	removeDirectoriesCb = func(string) (state.Directory, error)
)

type getDirectoriesRespone struct {
	Directories map[string]state.Directory `json:"directories"`
}

func (s *Server) getDirectoriesHandler(res http.ResponseWriter, req *http.Request) {
	directoriesResponse := getDirectoriesRespone{
		Directories: s.directories.All(),
	}

	response, err := json.Marshal(&directoriesResponse)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte("could not prepare output\n"))

		return
	}

	res.WriteHeader(200)
	res.Write(response)
}

func (s *Server) deleteDirectoriesHandler(res http.ResponseWriter, req *http.Request) {
	encodedPaths, ok := req.URL.Query()[pathArg]
	if !ok {
		res.WriteHeader(400)
		res.Write([]byte("no paths provided to delete directories handler request\n"))

		return
	}

	var paths []string
	for _, encodedPath := range encodedPaths {
		path, err := url.PathUnescape(encodedPath)
		if err != nil {
			res.WriteHeader(400)
			res.Write([]byte(fmt.Sprintf("could not decode path '%s'", encodedPath)))

			return
		}

		if !s.directories.Exists(path) {
			res.WriteHeader(404)
			res.Write([]byte(fmt.Sprintf("directory not found at path '%s'", encodedPath)))

			return
		}

		paths = append(paths, path)
	}

	for _, path := range paths {
		_, err := s.removeDirectoriesCb(path)
		if err != nil {
			s.errLog.Printf("couldn't remove directory at path '%s' due to error: %s\n", path, err)
			res.WriteHeader(404)
			res.Write([]byte(fmt.Sprintf("couldn't remove directory at path '%s'", path)))
		}
	}
}

func (s *Server) directoriesPathHandler(res http.ResponseWriter, req *http.Request) error {
	watchedDir, err := strconv.ParseBool(req.PostFormValue(watchedArg))
	if err != nil {
		return err
	}

	path := req.PostFormValue(pathArg)

	if watchedDir {
		s.outLog.Printf("watching directory '%s' due to request from %s\n", path, req.RemoteAddr)
	} else {
		s.outLog.Printf("reading directory '%s' due to request from %s\n", path, req.RemoteAddr)
	}

	s.addDirectoriesCb([]state.Directory{
		{
			Path:    path,
			Watched: watchedDir,
		},
	})

	return nil
}

func (s *Server) postDirectoriesFormArgumentsHandlers() map[string]common.FormArgument {
	return map[string]common.FormArgument{
		pathArg: {
			Handle: s.directoriesPathHandler,
		},
		watchedArg: {
			Validate: func(req *http.Request) error {
				_, err := strconv.ParseBool(req.PostFormValue(watchedArg))
				return err
			},
		},
	}
}
