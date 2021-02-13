package rest

import (
	"net/http"

	"github.com/sarpt/mpv-web-api/internal/common"
)

const (
	moviesPath      = "/rest/movies"
	directoriesPath = "/rest/directories"
	playbackPath    = "/rest/playback"
)

// Handler returns http.Handler responsible for REST handling subtree.
func (s *Server) Handler() http.Handler {
	playbackHandlers := map[string]http.HandlerFunc{
		http.MethodPost: common.CreateFormHandler(s.postPlaybackFormArgumentsHandlers()),
		http.MethodGet:  s.getPlaybackHandler,
	}

	moviesHandlers := map[string]http.HandlerFunc{
		http.MethodGet: s.getMoviesHandler,
	}

	directoriesHandlers := map[string]http.HandlerFunc{
		http.MethodGet:    s.getDirectoriesHandler,
		http.MethodPut:    common.CreateFormHandler(s.putDirectoriesFormArgumentsHandlers()),
		http.MethodDelete: s.deleteDirectoriesHandler,
	}

	allHandlers := map[string]common.MethodHandlers{
		playbackPath:    playbackHandlers,
		moviesPath:      moviesHandlers,
		directoriesPath: directoriesHandlers,
	}

	mux := http.NewServeMux()
	for path, methodHandlers := range allHandlers {
		cfg := common.PathHandlerConfig{
			AllowCORS:      s.allowCORS,
			MethodHandlers: methodHandlers,
		}
		mux.HandleFunc(path, common.PathHandler(cfg))
	}

	return mux
}
