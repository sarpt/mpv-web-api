package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/media_files"
)

type getMediaFilesRespone struct {
	MediaFiles map[string]media_files.Entry `json:"mediaFiles"`
}

func (s *Server) getMediaFilesHandler(res http.ResponseWriter, req *http.Request) {
	stateRevision := s.statesRepository.MediaFiles().Revision()
	if checkRevisionIsSame(stateRevision, req) {
		res.WriteHeader(304)
		res.Write(nil)
		return
	}

	mediaFilesResponse := getMediaFilesRespone{
		MediaFiles: s.statesRepository.MediaFiles().All(),
	}

	response, err := json.Marshal(&mediaFilesResponse)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintln("could not prepare output")))

		return
	}

	setRevisionInResponse(stateRevision, res)
	res.WriteHeader(200)
	res.Write(response)
}
