package api

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/sarpt/mpv-web-api/pkg/mpv"
)

var (
	// ErrResponseDataNotString occurs when observe response data is not a string.
	ErrResponseDataNotString = errors.New("response data is not a string")
	// ErrResponseDataNotInt occurs when observe response data is not an integer.
	ErrResponseDataNotInt = errors.New("response data is not an integer")

	// ErrPlaybackTimeNotFloat occurs when playback time is not a correct decimal number.
	ErrPlaybackTimeNotFloat = errors.New("playback time could not be converted to a float number")
	// ErrPlaybackPathNotServed occurs when playback path is set to file that is not being served by api.
	ErrPlaybackPathNotServed = errors.New("playback path is not served")
)

func (s *Server) handleFullscreenEvent(res mpv.ObservePropertyResponse) error {
	enabled, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	s.playback.Fullscreen = enabled == mpv.YesValue
	s.playback.Changes <- *s.playback
	return nil
}

func (s *Server) handleLoopFileEvent(res mpv.ObservePropertyResponse) error {
	enabled, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	if enabled != mpv.NoValue {
		s.playback.Loop.Variant = fileLoop
	} else {
		s.playback.Loop.Variant = ""
	}
	s.playback.Changes <- *s.playback
	return nil
}

func (s *Server) handlePauseEvent(res mpv.ObservePropertyResponse) error {
	paused, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	s.playback.Paused = paused == mpv.YesValue
	s.playback.Changes <- *s.playback
	return nil
}

func (s *Server) handleAudioIDChangeEvent(res mpv.ObservePropertyResponse) error {
	aid, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotInt
	}

	s.playback.SelectedAudioID = aid
	s.playback.Changes <- *s.playback
	return nil
}

func (s *Server) handleSubtitleIDChangeEvent(res mpv.ObservePropertyResponse) error {
	sid, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotInt
	}

	s.playback.SelectedAudioID = sid
	s.playback.Changes <- *s.playback
	return nil
}

func (s *Server) handleChapterChangeEvent(res mpv.ObservePropertyResponse) error {
	chapterIdx, ok := res.Data.(int)
	if !ok {
		return ErrResponseDataNotInt
	}

	s.playback.CurrentChapterIdx = chapterIdx
	s.playback.Changes <- *s.playback
	return nil
}

func (s *Server) handlePathEvent(res mpv.ObservePropertyResponse) error {
	if res.Data == nil {
		s.playback.Movie = Movie{}
		return nil
	}

	path, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	movie, err := s.movies.ByPath(path)
	if err != nil {
		return fmt.Errorf("%w:%s", ErrPlaybackPathNotServed, path)
	}

	s.playback.Movie = movie
	s.playback.Changes <- *s.playback
	return nil
}

func (s *Server) handlePlaybackTimeEvent(res mpv.ObservePropertyResponse) error {
	currentTime, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	if currentTime == "" {
		return nil
	}

	currentTimeNum, err := strconv.ParseFloat(currentTime, 64)
	if err != nil {
		return ErrPlaybackTimeNotFloat
	}

	s.playback.CurrentTime = currentTimeNum
	s.playback.Changes <- *s.playback
	return nil
}
