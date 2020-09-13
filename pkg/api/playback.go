package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

const (
	pathArg       = "path"
	fullscreenArg = "fullscreen"
	subtitleIDArg = "subtitleID"
	audioIDArg    = "audioID"
	pauseArg      = "pause"
	loopFileArg   = "loopFile"

	playbackSseEvent = "playback"
)

type getPlaybackResponse struct {
	Playback Playback `json:"playback"`
}

type postPlaybackResponse struct {
	ArgumentErrors map[string]string `json:"argumentErrors"`
	GeneralError   string            `json:"generalError"`
}

var (
	postPlaybackFormArgumentsHandlers = map[string]func(res http.ResponseWriter, req *http.Request, s *Server) error{
		pathArg:       pathHandler,
		fullscreenArg: fullscreenHandler,
		audioIDArg:    audioIDHandler,
		subtitleIDArg: subtitleIDHandler,
		pauseArg:      pauseHandler,
		loopFileArg:   loopFileHandler,
	}
)

func (s *Server) postPlaybackHandler(res http.ResponseWriter, req *http.Request) {
	responsePayload := postPlaybackResponse{
		ArgumentErrors: map[string]string{},
	}

	for arg, handler := range postPlaybackFormArgumentsHandlers {
		postVal := req.PostFormValue(arg)
		if postVal == "" {
			continue
		}

		err := handler(res, req, s)
		if err != nil {
			responsePayload.ArgumentErrors[arg] = err.Error()
			break
		}
	}

	out, err := json.Marshal(responsePayload)
	if err != nil {
		responsePayload.GeneralError = fmt.Sprintf("could not encode json payload: %s", err)
		s.errLog.Printf(responsePayload.GeneralError)
		res.WriteHeader(500)

		return
	}

	res.WriteHeader(200)
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

	// Buffer of 1 in case connection is closed after playbackObservers fan-out dispatcher already acquired read lock (blocking the write lock).
	// The dispatcher will expect for the select below to receive the message but the Context().Done() already waits to acquire a write lock.
	// So the buffer of 1 ensures that one message will be buffered, dispatcher will not be blocked, and write lock will be obtained.
	// When the write lock is obtained to remove from the set, even if a new playback will be received, read lock will wait until Context().Done() finishes.
	playbackChanges := make(chan Playback, 1)
	s.playbackObserversLock.Lock()
	s.playbackObservers[req.RemoteAddr] = playbackChanges
	s.playbackObserversLock.Unlock()

	for {
		select {
		case playback, ok := <-playbackChanges:
			if !ok {
				return
			}

			out, err := json.Marshal(playback)
			if err != nil {
				s.errLog.Println("could not write to the client")
			}

			_, err = res.Write(formatSseEvent(playbackSseEvent, out))
			if err != nil {
				s.errLog.Println("could not write to the client")
			}

			flusher.Flush()
		case <-req.Context().Done():
			s.playbackObserversLock.Lock()
			delete(s.playbackObservers, req.RemoteAddr)
			s.playbackObserversLock.Unlock()
			return
		}
	}
}

func pathHandler(res http.ResponseWriter, req *http.Request, s *Server) error {
	filePath := req.PostFormValue(pathArg)
	s.outLog.Printf("playing file '%s' due to request from %s\n", filePath, req.RemoteAddr)

	return s.mpvManager.LoadFile(filePath)
}

func fullscreenHandler(res http.ResponseWriter, req *http.Request, s *Server) error {
	fullscreen, err := strconv.ParseBool(req.PostFormValue(fullscreenArg))
	if err != nil {
		return err
	}

	s.outLog.Printf("changing fullscreen to %t due to request from %s\n", fullscreen, req.RemoteAddr)
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

func loopFileHandler(res http.ResponseWriter, req *http.Request, s *Server) error {
	loopFile, err := strconv.ParseBool(req.PostFormValue(loopFileArg))
	if err != nil {
		return err
	}

	return s.mpvManager.LoopFile(loopFile)
}

func pauseHandler(res http.ResponseWriter, req *http.Request, s *Server) error {
	pause, err := strconv.ParseBool(req.PostFormValue(pauseArg))
	if err != nil {
		return err
	}

	return s.mpvManager.ChangePause(pause)
}
