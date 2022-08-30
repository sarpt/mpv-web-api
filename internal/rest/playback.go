package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/sarpt/mpv-web-api/internal/common"
)

const (
	appendArg       = "append"
	audioIDArg      = "audioID"
	chapterArg      = "chapter"
	chaptersArgs    = "chapters"
	fullscreenArg   = "fullscreen"
	loopFileArg     = "loopFile"
	pauseArg        = "pause"
	playlistIdxArg  = "playlistIdx"
	playlistUUIDArg = "playlistUUID"
	stopArg         = "stop"
	subtitleIDArg   = "subtitleID"
)

type (
	loadFileCb            func(string, bool) error
	changeFullscreenCb    func(bool) error
	changeAudioCb         func(string) error
	changeChapterCb       func(int64) error
	changeSubtitleCb      func(string) error
	loopFileCb            func(bool) error
	changePauseCb         func(bool) error
	changeChaptersOrderCb func([]int64) error
	playlistPlayIndexCb   func(int) error
	stopPlaybackCb        func() error
)

func (s *Server) getPlaybackHandler(res http.ResponseWriter, req *http.Request) {
	json, err := json.Marshal(s.statesRepository.Playback())
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintf("could not marshall to JSON: %s\n", err))) // good enough for poc

		return
	}

	res.WriteHeader(200)
	res.Write(json)
}

func (s *Server) pathHandler(res http.ResponseWriter, req *http.Request) error {
	filePath := req.PostFormValue(pathArg)

	append, err := getAppendArgument(req)
	if err != nil {
		return err
	}

	s.outLog.Printf("loading file '%s' with '%t' argument due to request from %s\n", filePath, append, req.RemoteAddr)

	return s.loadFileCb(filePath, append)
}

func (s *Server) fullscreenHandler(res http.ResponseWriter, req *http.Request) error {
	fullscreen, err := strconv.ParseBool(req.PostFormValue(fullscreenArg))
	if err != nil {
		return err
	}

	s.outLog.Printf("changing fullscreen to %t due to request from %s\n", fullscreen, req.RemoteAddr)
	return s.changeFullscreenCb(fullscreen)
}

func (s *Server) audioIDHandler(res http.ResponseWriter, req *http.Request) error {
	audioID := req.PostFormValue(audioIDArg)

	s.outLog.Printf("changing audio id to %s due to request from %s\n", audioID, req.RemoteAddr)
	return s.changeAudioCb(audioID)
}

func (s *Server) chapterHandler(res http.ResponseWriter, req *http.Request) error {
	chapterIdx, err := strconv.ParseInt(req.PostFormValue(chapterArg), 10, 64)
	if err != nil {
		return err
	}

	s.outLog.Printf("changing chapter id to %d due to request from %s\n", chapterIdx, req.RemoteAddr)
	return s.changeChapterCb(chapterIdx)
}

func (s *Server) chaptersHandler(res http.ResponseWriter, req *http.Request) error {
	providedChaptersArg := req.PostFormValue(chaptersArgs)
	chapters := strings.Split(providedChaptersArg, ",")
	chapterIds := []int64{}
	for _, chapter := range chapters {
		chapterId, err := strconv.Atoi(chapter)
		if err != nil {
			return err
		}

		chapterIds = append(chapterIds, int64(chapterId))
	}

	s.outLog.Printf("changing chapters order to %s due to request from %s\n", providedChaptersArg, req.RemoteAddr)
	return s.changeChaptersOrderCb(chapterIds)
}

func (s *Server) subtitleIDHandler(res http.ResponseWriter, req *http.Request) error {
	subtitleID := req.PostFormValue(subtitleIDArg)

	s.outLog.Printf("changing subtitle id to %s due to request from %s\n", subtitleID, req.RemoteAddr)
	return s.changeSubtitleCb(subtitleID)
}

func (s *Server) loopFileHandler(res http.ResponseWriter, req *http.Request) error {
	loopFile, err := strconv.ParseBool(req.PostFormValue(loopFileArg))
	if err != nil {
		return err
	}

	s.outLog.Printf("changing file looping to %t due to request from %s\n", loopFile, req.RemoteAddr)
	return s.loopFileCb(loopFile)
}

func (s *Server) pauseHandler(res http.ResponseWriter, req *http.Request) error {
	pause, err := strconv.ParseBool(req.PostFormValue(pauseArg))
	if err != nil {
		return err
	}

	s.outLog.Printf("changing pause to %t due to request from %s\n", pause, req.RemoteAddr)
	return s.changePauseCb(pause)
}

func (s *Server) playlistIdxHandler(res http.ResponseWriter, req *http.Request) error {
	idx, err := strconv.Atoi(req.PostFormValue(playlistIdxArg))
	if err != nil {
		return err
	}

	s.outLog.Printf("changing playlist idx to %d due to request from %s\n", idx, req.RemoteAddr)
	return s.playlistPlayIndexCb(idx)
}

func (s *Server) playlistUUIDHandler(res http.ResponseWriter, req *http.Request) error {
	uuid := req.PostFormValue(playlistUUIDArg)

	append, err := getAppendArgument(req)
	if err != nil {
		return err
	}

	s.outLog.Printf("loading playlist with uuid '%s' and append '%t' due to request from %s\n", uuid, append, req.RemoteAddr)
	return s.loadPlaylistCb(uuid, append)
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
	return s.stopPlaybackCb()
}

func getAppendArgument(req *http.Request) (bool, error) {
	appendArgInForm := req.PostFormValue(appendArg)
	if appendArgInForm == "" {
		return false, nil
	}

	append, err := strconv.ParseBool(appendArgInForm)
	return append, err
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
		chaptersArgs: {
			Handle: s.chaptersHandler,
			Validate: func(req *http.Request) error {
				chapters := strings.Split(req.PostFormValue(chaptersArgs), ",")
				for _, chapter := range chapters {
					_, err := strconv.Atoi(chapter)
					if err == nil {
						continue
					}

					return err
				}

				return nil
			},
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
			Handle:   s.pathHandler,
			Priority: 1,
		},
		pauseArg: {
			Handle: s.pauseHandler,
			Validate: func(req *http.Request) error {
				_, err := strconv.ParseBool(req.PostFormValue(pauseArg))
				return err
			},
		},
		playlistIdxArg: {
			Handle: s.playlistIdxHandler,
			Validate: func(req *http.Request) error {
				_, err := strconv.Atoi(req.PostFormValue(playlistIdxArg))
				return err
			},
		},
		playlistUUIDArg: {
			Handle:   s.playlistUUIDHandler,
			Priority: 1,
		},
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
