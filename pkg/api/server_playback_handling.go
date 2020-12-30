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
	stopArg       = "stop"

	playbackAllSseEvent    = "all"
	playbackReplaySseEvent = "replay"
)

var (
	postPlaybackFormArgumentsHandlers = map[string]formArgumentHandler{
		pathArg:       pathHandler,
		fullscreenArg: fullscreenHandler,
		audioIDArg:    audioIDHandler,
		subtitleIDArg: subtitleIDHandler,
		pauseArg:      pauseHandler,
		loopFileArg:   loopFileHandler,
		stopArg:       stopHandler,
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
	responsePayload.GeneralError = errors.GeneralError
	if responsePayload.GeneralError != "" {
		s.errLog.Printf(errors.GeneralError)
		out, err := prepareJSONOutput(responsePayload)
		if err != nil {
			res.WriteHeader(400)
		} else {
			res.WriteHeader(500)
		}
		res.Write(out)

		return
	}

	responsePayload.ArgumentErrors = errors.ArgumentErrors

	for _, handler := range args {
		err := handler(res, req, s)
		if err != nil {
			responsePayload.GeneralError = err.Error()
			s.errLog.Printf(responsePayload.GeneralError)
			out, _ := prepareJSONOutput(responsePayload)
			res.WriteHeader(500)
			res.Write(out)

			return
		}
	}

	out, err := prepareJSONOutput(responsePayload)
	if err == nil {
		s.errLog.Printf("%s", out)
		res.WriteHeader(500)
		res.Write(out)

		return
	}

	res.WriteHeader(200)
	res.Write(out)
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
		return sendPlayback(s.playback, playbackReplaySseEvent, res)
	}
}

func (s *Server) createPlaybackChangesHandler() sseChangeHandler {
	return func(res SSEResponseWriter, changes interface{}) error {
		change, ok := changes.(PlaybackChange)
		if !ok {
			return errIncorrectChangesType
		}

		return sendPlayback(s.playback, change.Variant, res)
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

func stopHandler(res http.ResponseWriter, req *http.Request, s *Server) error {
	stop, err := strconv.ParseBool(req.PostFormValue(stopArg))
	if err != nil {
		return err
	}

	if !stop {
		return nil
	}

	s.outLog.Printf("stopping playback due to request from %s\n", req.RemoteAddr)
	return s.mpvManager.Stop()
}

func sendPlayback(playback *Playback, changeVariant PlaybackChangeVariant, res SSEResponseWriter) error {
	out, err := json.Marshal(playback)
	if err != nil {
		return fmt.Errorf("%w: %s", errResponseJSONCreationFailed, err)
	}

	_, err = res.Write(formatSseEvent(playbackSSEChannelVariant, string(changeVariant), out))
	if err != nil {
		return fmt.Errorf("sending playback failed: %w: %s", errClientWritingFailed, err)
	}

	return nil
}

func prepareJSONOutput(res postPlaybackResponse) ([]byte, error) {
	out, err := json.Marshal(res)
	if err != nil {
		return []byte(fmt.Sprintf("could not encode json payload: %s", err)), err
	}

	return out, nil
}
