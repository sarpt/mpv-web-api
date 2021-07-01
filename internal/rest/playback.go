package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/sarpt/mpv-web-api/internal/common"
)

const (
	appendArg     = "append"
	audioIDArg    = "audioID"
	chapterArg    = "chapter"
	fullscreenArg = "fullscreen"
	loopFileArg   = "loopFile"
	pathArg       = "path"
	pauseArg      = "pause"
	playlistIdx   = "playlistIdx"
	playlistUUID  = "playlistUUID"
	stopArg       = "stop"
	subtitleIDArg = "subtitleID"
)

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

func (s *Server) pathHandler(res http.ResponseWriter, req *http.Request) error {
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

func (s *Server) fullscreenHandler(res http.ResponseWriter, req *http.Request) error {
	fullscreen, err := strconv.ParseBool(req.PostFormValue(fullscreenArg))
	if err != nil {
		return err
	}

	s.outLog.Printf("changing fullscreen to %t due to request from %s\n", fullscreen, req.RemoteAddr)
	return s.mpvManager.ChangeFullscreen(fullscreen)
}

func (s *Server) audioIDHandler(res http.ResponseWriter, req *http.Request) error {
	audioID := req.PostFormValue(audioIDArg)

	s.outLog.Printf("changing audio id to %s due to request from %s\n", audioID, req.RemoteAddr)
	return s.mpvManager.ChangeAudio(audioID)
}

func (s *Server) chapterHandler(res http.ResponseWriter, req *http.Request) error {
	chapterIdx, err := strconv.ParseInt(req.PostFormValue(chapterArg), 10, 64)
	if err != nil {
		return err
	}

	s.outLog.Printf("changing chapter id to %d due to request from %s\n", chapterIdx, req.RemoteAddr)
	return s.mpvManager.ChangeChapter(chapterIdx)
}

func (s *Server) subtitleIDHandler(res http.ResponseWriter, req *http.Request) error {
	subtitleID := req.PostFormValue(subtitleIDArg)

	s.outLog.Printf("changing subtitle id to %s due to request from %s\n", subtitleID, req.RemoteAddr)
	return s.mpvManager.ChangeSubtitle(subtitleID)
}

func (s *Server) loopFileHandler(res http.ResponseWriter, req *http.Request) error {
	loopFile, err := strconv.ParseBool(req.PostFormValue(loopFileArg))
	if err != nil {
		return err
	}

	s.outLog.Printf("changing file looping to %t due to request from %s\n", loopFile, req.RemoteAddr)
	return s.mpvManager.LoopFile(loopFile)
}

func (s *Server) pauseHandler(res http.ResponseWriter, req *http.Request) error {
	pause, err := strconv.ParseBool(req.PostFormValue(pauseArg))
	if err != nil {
		return err
	}

	s.outLog.Printf("changing pause to %t due to request from %s\n", pause, req.RemoteAddr)
	return s.mpvManager.ChangePause(pause)
}

func (s *Server) playlistIdxHandler(res http.ResponseWriter, req *http.Request) error {
	idx, err := strconv.Atoi(req.PostFormValue(playlistIdx))
	if err != nil {
		return err
	}

	s.outLog.Printf("changing playlist idx to %d due to request from %s\n", idx, req.RemoteAddr)
	return s.mpvManager.PlaylistPlayIndex(idx)
}

func (s *Server) stopHandler(res http.ResponseWriter, req *http.Request) error {
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

func (s *Server) postPlaybackFormArgumentsHandlers() map[string]common.FormArgument {
	return map[string]common.FormArgument{
		appendArg: {
			Validate: func(req *http.Request) error {
				_, err := strconv.ParseBool(req.PostFormValue(appendArg))
				return err
			},
		},
		audioIDArg: {
			Handle: s.audioIDHandler,
		},
		chapterArg: {
			Handle: s.chapterHandler,
		},
		fullscreenArg: {
			Handle: s.fullscreenHandler,
			Validate: func(req *http.Request) error {
				_, err := strconv.ParseBool(req.PostFormValue(fullscreenArg))
				return err
			},
		},
		loopFileArg: {
			Handle: s.loopFileHandler,
			Validate: func(req *http.Request) error {
				_, err := strconv.ParseBool(req.PostFormValue(loopFileArg))
				return err
			},
		},
		pathArg: {
			Handle: s.pathHandler,
		},
		pauseArg: {
			Handle: s.pauseHandler,
			Validate: func(req *http.Request) error {
				_, err := strconv.ParseBool(req.PostFormValue(pauseArg))
				return err
			},
		},
		playlistIdx: {
			Handle: s.playlistIdxHandler,
			Validate: func(req *http.Request) error {
				_, err := strconv.Atoi(req.PostFormValue(playlistIdx))
				return err
			},
		},
		// TODO: to be implemented. Prereqs: api server should handle reading/storing user-defined playlists.
		// The handler should also take into account possibility that with the same POST both playlistIdx and playlistUUID will be passed -
		// in such case, playlistIdx handler should not be called/handled on it's own, but by this playlistUUID handler. This requires
		// implementation of some form of a "ShouldHandle" callback (in the vein of "Validate") which will check whether specific handler should be called.
		// playlistUUID: {
		// 	Handle: s.playlistUUIDHandler,
		// },
		subtitleIDArg: {
			Handle: s.subtitleIDHandler,
		},
		stopArg: {
			Handle: s.stopHandler,
			Validate: func(req *http.Request) error {
				_, err := strconv.ParseBool(req.PostFormValue(stopArg))
				return err
			},
		},
	}
}
