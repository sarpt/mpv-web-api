package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playlists"
)

type getPlaylistsRespone struct {
	Playlists map[string]*playlists.Playlist `json:"playlists"`
}

func (s *Server) getPlaylistsHandler(res http.ResponseWriter, req *http.Request) {
	stateRevision := s.statesRepository.Playlists().Revision()
	if checkRevisionIsSame(stateRevision, req) {
		res.WriteHeader(304)
		res.Write(nil)
		return
	}

	playlistsResponse := getPlaylistsRespone{
		Playlists: s.statesRepository.Playlists().All(),
	}

	response, err := json.Marshal(&playlistsResponse)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintln("could not prepare output")))

		return
	}

	setRevisionInResponse(stateRevision, res)
	res.WriteHeader(200)
	res.Write(response)
}
