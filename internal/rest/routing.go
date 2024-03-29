package rest

import (
	"net/http"

	"github.com/sarpt/mpv-web-api/internal/common"
)

const (
	mediaFilesPath  = "/rest/media-files"
	directoriesPath = "/rest/directories"
	playbackPath    = "/rest/playback"
	playlistsPath   = "/rest/playlists"
)

// Handler returns http.Handler responsible for REST handling subtree.
func (s *Server) Handler() http.Handler {
	playbackHandlers := map[string]http.HandlerFunc{
		http.MethodPost: common.CreateFormHandler(s.postPlaybackFormArgumentsHandlers()),
		http.MethodGet:  s.getPlaybackHandler,
	}

	mediaFilesHandlers := map[string]http.HandlerFunc{
		http.MethodGet: s.getMediaFilesHandler,
	}

	playlistsHandlers := map[string]http.HandlerFunc{
		http.MethodGet: s.getPlaylistsHandler,
	}

	directoriesHandlers := map[string]http.HandlerFunc{
		http.MethodGet:    s.getDirectoriesHandler,
		http.MethodPost:   common.CreateFormHandler(s.postDirectoriesFormArgumentsHandlers()),
		http.MethodDelete: s.deleteDirectoriesHandler,
	}

	allHandlers := map[string]common.MethodHandlers{
		playbackPath:    playbackHandlers,
		mediaFilesPath:  mediaFilesHandlers,
		directoriesPath: directoriesHandlers,
		playlistsPath:   playlistsHandlers,
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
