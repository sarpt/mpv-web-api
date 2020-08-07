package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
)

const (
	pathArg       = "path"
	fullscreenArg = "fullscreen"
	subtitleIDArg = "subtitleID"
	audioIDArg    = "audioID"

	playbackSseEvent = "playback"
)

type getPlaybackResponse struct {
	Playback Playback `json:"playback"`
}

type postPlaybackResponse struct {
	getPlaybackResponse
	Error string `json:"error"`
}

var (
	postFormArgumentsHandlers = map[string]func(res http.ResponseWriter, req *http.Request, s *Server) error{
		pathArg: pathHandler,
	}
)

func (s *Server) postPlaybackHandler(res http.ResponseWriter, req *http.Request) {
	responsePayload := postPlaybackResponse{}

	for arg, handler := range postFormArgumentsHandlers {
		postVal := req.PostFormValue(arg)
		if postVal == "" {
			continue
		}

		err := handler(res, req, s)
		if err != nil {
			responsePayload.Error = err.Error()
			break
		}
	}

	responsePayload.Playback = *s.playback
	out, err := json.Marshal(responsePayload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not encode json payload: %s", err)
		res.WriteHeader(500)

		return
	}

	if responsePayload.Error != "" {
		res.WriteHeader(400)
	} else {
		res.WriteHeader(200)
	}
	res.Write([]byte(out))
}

func (s *Server) getPlaybackHandler(res http.ResponseWriter, req *http.Request) {
	playbackResponse := getPlaybackResponse{
		Playback: *s.playback,
	}

	response, err := json.Marshal(&playbackResponse)
	if err != nil {
		res.WriteHeader(400)
		res.Write([]byte(fmt.Sprintf("could not prepare output: %s\n", err))) // good enough for poc

		return
	}
	res.WriteHeader(200)
	res.Write(response)
}

func (s *Server) getSsePlaybackHandler(res http.ResponseWriter, req *http.Request) {
	flusher, ok := res.(http.Flusher)
	if !ok {
		res.WriteHeader(400)
		return
	}
	res.Header().Set("Connection", "keep-alive")
	res.Header().Set("Content-Type", "text/event-stream")
	res.Header().Set("Access-Control-Allow-Origin", "*")

	playbackChanges := make(chan Playback)
	s.playbackObservers = append(s.playbackObservers, playbackChanges)

	for {
		select {
		case playback, ok := <-playbackChanges:
			if !ok {
				return
			}

			out, err := json.Marshal(playback)
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not write to the client")
			}

			_, err = res.Write(formatSseEvent(playbackSseEvent, out))
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not write to the client")
			}

			flusher.Flush()
		case <-req.Context().Done():
			return
		}
	}
}

func pathHandler(res http.ResponseWriter, req *http.Request, s *Server) error {
	filePath := req.PostFormValue(pathArg)
	fmt.Fprintf(os.Stdout, "playing file '%s' due to request from %s\n", filePath, req.RemoteAddr)

	return s.mpvManager.LoadFile(filePath)
}

func fullscreenHandler(res http.ResponseWriter, req *http.Request, s *Server) error {
	fullscreen, err := strconv.ParseBool(req.PostFormValue(fullscreenArg))
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "changing fullscreen to %t due to request from %s\n", fullscreen, req.RemoteAddr)
	return s.mpvManager.ChangeFullscreen(fullscreen)
}

func audioIDHandler(res http.ResponseWriter, req *http.Request, s *Server) error {
	audioID := req.PostFormValue(audioIDArg)

	return s.mpvManager.ChangeAudio(audioID)
}

func subtitleIDHandler(res http.ResponseWriter, req *http.Request, s *Server) error {
	subtitleID := req.PostFormValue(subtitleIDArg)

	return s.mpvManager.ChangeSubtitle(subtitleID)
}
