package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

const (
	playbackObserverVariant StatusObserverVariant = "playback"

	pathArg       = "path"
	fullscreenArg = "fullscreen"
	subtitleIDArg = "subtitleID"
	audioIDArg    = "audioID"
	pauseArg      = "pause"
	loopFileArg   = "loopFile"

	playbackAllSseEvent = "all"

	fileLoop loopVariant = "file"
	abLoop   loopVariant = "ab"
)

type loopVariant string

// PlaybackLoop contains information about playback loop
type PlaybackLoop struct {
	Variant loopVariant
	ATime   int
	BTime   int
}

// Playback contains information about currently played movie file
type Playback struct {
	CurrentTime        float64
	CurrentChapterIdx  int
	Fullscreen         bool
	Movie              Movie
	SelectedAudioID    int
	SelectedSubtitleID int
	Paused             bool
	Loop               PlaybackLoop
}

type getPlaybackResponse struct {
	Playback Playback `json:"playback"`
}

type postPlaybackResponse struct {
	handlerErrors
}

var (
	postPlaybackFormArgumentsHandlers = map[string]formArgumentHandler{
		pathArg:       pathHandler,
		fullscreenArg: fullscreenHandler,
		audioIDArg:    audioIDHandler,
		subtitleIDArg: subtitleIDHandler,
		pauseArg:      pauseHandler,
		loopFileArg:   loopFileHandler,
	}
)

func (s *Server) postPlaybackHandler(res http.ResponseWriter, req *http.Request) {
	responsePayload := postPlaybackResponse{}

	args, errors := validateFormRequest(req, postPlaybackFormArgumentsHandlers)
	if errors.GeneralError != "" {
		s.errLog.Printf(responsePayload.GeneralError)
		res.WriteHeader(400)
		res.Write([]byte(fmt.Sprintf(responsePayload.GeneralError)))

		return
	}

	responsePayload.ArgumentErrors = errors.ArgumentErrors

	for _, handler := range args {
		err := handler(res, req, s)
		if err != nil {
			responsePayload.GeneralError = err.Error()
			s.errLog.Printf(responsePayload.GeneralError)
			res.WriteHeader(500)
			res.Write([]byte(fmt.Sprintf(responsePayload.GeneralError)))

			return
		}
	}

	out, err := json.Marshal(responsePayload)
	if err != nil {
		responsePayload.GeneralError = fmt.Sprintf("could not encode json payload: %s", err)
		s.errLog.Printf(responsePayload.GeneralError)
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintf(responsePayload.GeneralError)))

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
	flusher, err := sseFlusher(res)
	if err != nil {
		res.WriteHeader(400)
		return
	}

	// Buffer of 1 in case connection is closed after playbackObservers fan-out dispatcher already acquired read lock (blocking the write lock).
	// The dispatcher will expect for the select below to receive the message but the Context().Done() already waits to acquire a write lock.
	// So the buffer of 1 ensures that one message will be buffered, dispatcher will not be blocked, and write lock will be obtained.
	// When the write lock is obtained to remove from the set, even if a new playback will be received, read lock will wait until Context().Done() finishes.
	playbackChanges := make(chan Playback, 1)
	s.playbackObserversLock.Lock()
	s.playbackObservers[req.RemoteAddr] = playbackChanges
	s.playbackObserversLock.Unlock()

	s.addObservingAddressToStatus(req.RemoteAddr, playbackObserverVariant)
	s.outLog.Printf("added /sse/playback observer with addr %s\n", req.RemoteAddr)

	if replaySseState(req) {
		err := sendPlayback(*s.playback, res, flusher)
		if err != nil {
			s.errLog.Println(err.Error())
		}
	}

	for {
		select {
		case playback, ok := <-playbackChanges:
			if !ok {
				return
			}

			err := sendPlayback(playback, res, flusher)
			if err != nil {
				s.errLog.Println(err.Error())
			}
		case <-req.Context().Done():
			s.playbackObserversLock.Lock()
			delete(s.playbackObservers, req.RemoteAddr)
			s.playbackObserversLock.Unlock()

			s.removeObservingAddressFromStatus(req.RemoteAddr, playbackObserverVariant)
			s.outLog.Printf("removing /sse/playback observer with addr %s\n", req.RemoteAddr)

			return
		}
	}
}

// watchPlaybackChanges reads all playbackChanges done by path/event handlers.
// It's a fan-out dispatcher, which notifies all playback observers (subscribers from SSE etc.) when a playbackChange occurs.
func (s Server) watchPlaybackChanges() {
	for {
		playback, ok := <-s.playbackChanges
		if !ok {
			return
		}

		s.playbackObserversLock.RLock()
		for _, observer := range s.playbackObservers {
			observer <- playback
		}
		s.playbackObserversLock.RUnlock()
	}
}

func sendPlayback(playback Playback, res http.ResponseWriter, flusher http.Flusher) error {
	out, err := json.Marshal(playback)
	if err != nil {
		return errResponseJSONCreationFailed
	}

	_, err = res.Write(formatSseEvent(playbackAllSseEvent, out))
	if err != nil {
		return errClientWritingFailed
	}

	flusher.Flush()
	return nil
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

	s.outLog.Printf("changing audio id to %s due to request from %s\n", audioID, req.RemoteAddr)
	return s.mpvManager.ChangeAudio(audioID)
}

func subtitleIDHandler(res http.ResponseWriter, req *http.Request, s *Server) error {
	subtitleID := req.PostFormValue(subtitleIDArg)

	s.outLog.Printf("changing subtitle id to %s due to request from %s\n", subtitleID, req.RemoteAddr)
	return s.mpvManager.ChangeSubtitle(subtitleID)
}

func loopFileHandler(res http.ResponseWriter, req *http.Request, s *Server) error {
	loopFile, err := strconv.ParseBool(req.PostFormValue(loopFileArg))
	if err != nil {
		return err
	}

	s.outLog.Printf("changing file looping to %t due to request from %s\n", loopFile, req.RemoteAddr)
	return s.mpvManager.LoopFile(loopFile)
}

func pauseHandler(res http.ResponseWriter, req *http.Request, s *Server) error {
	pause, err := strconv.ParseBool(req.PostFormValue(pauseArg))
	if err != nil {
		return err
	}

	s.outLog.Printf("changing pause to %t due to request from %s\n", pause, req.RemoteAddr)
	return s.mpvManager.ChangePause(pause)
}
