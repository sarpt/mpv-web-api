package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/sarpt/mpv-web-api/internal/state"
)

const (
	playbackSSEChannelVariant state.SSEChannelVariant = "playback"

	appendArg     = "append"
	audioIDArg    = "audioID"
	fullscreenArg = "fullscreen"
	loopFileArg   = "loopFile"
	pathArg       = "path"
	pauseArg      = "pause"
	stopArg       = "stop"
	subtitleIDArg = "subtitleID"

	playbackAllSseEvent    = "all"
	playbackReplaySseEvent = "replay"
)

var (
	postPlaybackFormArgumentsHandlers = map[string]formArgument{
		appendArg: {
			validate: func(req *http.Request) bool {
				_, err := strconv.ParseBool(req.PostFormValue(appendArg))
				return err == nil
			},
		},
		pathArg: {
			handle: pathHandler,
		},
		fullscreenArg: {
			handle: fullscreenHandler,
			validate: func(req *http.Request) bool {
				_, err := strconv.ParseBool(req.PostFormValue(fullscreenArg))
				return err == nil
			},
		},
		audioIDArg: {
			handle: audioIDHandler,
		},
		subtitleIDArg: {
			handle: subtitleIDHandler,
		},
		pauseArg: {
			handle: pauseHandler,
			validate: func(req *http.Request) bool {
				_, err := strconv.ParseBool(req.PostFormValue(pauseArg))
				return err == nil
			},
		},
		loopFileArg: {
			handle: loopFileHandler,
			validate: func(req *http.Request) bool {
				_, err := strconv.ParseBool(req.PostFormValue(loopFileArg))
				return err == nil
			},
		},
		stopArg: {
			handle: stopHandler,
			validate: func(req *http.Request) bool {
				_, err := strconv.ParseBool(req.PostFormValue(stopArg))
				return err == nil
			},
		},
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
		change, ok := changes.(state.PlaybackChange)
		if !ok {
			return errIncorrectChangesType
		}

		return sendPlayback(s.playback, change.Variant, res)
	}
}

func pathHandler(res http.ResponseWriter, req *http.Request, s *Server) error {
	var append bool = false
	var err error

	filePath := req.PostFormValue(pathArg)

	appendArgInForm := req.PostFormValue(appendArg)
	if appendArgInForm != "" {
		append, err = strconv.ParseBool(appendArgInForm)
		if err != nil {
			return err
		}
	}

	s.outLog.Printf("loading file '%s' with '%t' argument due to request from %s\n", filePath, append, req.RemoteAddr)

	return s.mpvManager.LoadFile(filePath, append)
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

func sendPlayback(playback *state.Playback, changeVariant state.PlaybackChangeVariant, res SSEResponseWriter) error {
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
