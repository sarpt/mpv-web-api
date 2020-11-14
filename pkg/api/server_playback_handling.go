package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

const (
	playbackSSEChannelVariant SSEChannelVariant = "playback"

	pathArg       = "path"
	fullscreenArg = "fullscreen"
	subtitleIDArg = "subtitleID"
	audioIDArg    = "audioID"
	pauseArg      = "pause"
	loopFileArg   = "loopFile"

	playbackAllSseEvent = "all"
)

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

type postPlaybackResponse struct {
	handlerErrors
}

func (s *Server) playbackSSEChannel() SSEChannel {
	return SSEChannel{
		Variant:       playbackSSEChannelVariant,
		Observers:     s.playbackSSEObservers,
		ChangeHandler: s.createPlaybackChangesHandler(),
		ReplayHandler: s.createPlaybackReplayHandler(),
	}
}
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
	json, err := json.Marshal(s.playback)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintf("could not marshall to JSON: %s\n", err))) // good enough for poc

		return
	}

	res.WriteHeader(200)
	res.Write(json)
}

func (s *Server) createPlaybackReplayHandler() sseReplayHandler {
	return func(res SSEResponseWriter) error {
		return sendPlayback(*s.playback, res)
	}
}

func (s *Server) createPlaybackChangesHandler() sseChangeHandler {
	return func(res SSEResponseWriter, changes interface{}) error {
		newPlayback, ok := changes.(Playback)
		if !ok {
			return errIncorrectChangesType
		}

		return sendPlayback(newPlayback, res)
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

func sendPlayback(playback Playback, res SSEResponseWriter) error {
	out, err := json.Marshal(&playback)
	if err != nil {
		return fmt.Errorf("%w: %s", errResponseJSONCreationFailed, err)
	}

	_, err = res.Write(formatSseEvent(playbackAllSseEvent, out))
	if err != nil {
		return fmt.Errorf("sending playback failed: %w: %s", errClientWritingFailed, err)
	}

	return nil
}
