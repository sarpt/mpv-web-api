package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sarpt/mpv-web-api/internal/state"
)

type getMediaFilesRespone struct {
	MediaFiles map[string]state.MediaFile `json:"mediaFiles"`
}

func (s *Server) getMediaFilesHandler(res http.ResponseWriter, req *http.Request) {
	mediaFilesResponse := getMediaFilesRespone{
		MediaFiles: s.mediaFiles.All(),
	}

	response, err := json.Marshal(&mediaFilesResponse)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintln("could not prepare output")))

		return
	}

	res.WriteHeader(200)
	res.Write(response)
}
