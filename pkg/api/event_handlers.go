package api

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/sarpt/mpv-web-api/pkg/mpv"
)

func (s *Server) handleFullscreenEvent(res mpv.ObserveResponse) error {
	enabled, ok := res.Data.(string)
	if !ok {
		return errors.New("could not decode data for fullscreen change event")
	}

	s.playback.Fullscreen = enabled == mpv.FullscreenEnabled
	return nil
}

func (s *Server) handlePathEvent(res mpv.ObserveResponse) error {
	if res.Data == nil {
		s.playback.Movie = Movie{}
		return nil
	}

	path, ok := res.Data.(string)
	if !ok {
		return errors.New("could not decode data for path change event")
	}

	movie, err := s.movieByPath(path)
	if err != nil {
		return fmt.Errorf("could not retrieve movie by path %s", path)
	}

	s.playback.Movie = movie
	return nil
}

func (s *Server) handlePlaybackTimeEvent(res mpv.ObserveResponse) error {
	currentTime, ok := res.Data.(string)
	if !ok {
		return errors.New("could not decode data for playback time change event")
	}

	if currentTime == "" {
		return nil
	}

	currentTimeNum, err := strconv.ParseFloat(currentTime, 64)
	if err != nil {
		return errors.New("the playback time could not be converted to number")
	}

	s.playback.CurrentTime = currentTimeNum
	return nil
}
