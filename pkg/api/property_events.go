package api

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/sarpt/mpv-web-api/internal/state"
	"github.com/sarpt/mpv-web-api/pkg/mpv"
)

var (
	// ErrResponseDataNotString occurs when observe response data is not a string.
	ErrResponseDataNotString = errors.New("response data is not a string")
	// ErrResponseDataNotInt occurs when observe response data is not an integer.
	ErrResponseDataNotInt = errors.New("response data is not an integer")
	// ErrResponseDataNotExpectedFormatNode occurs when observe response data is not expected MPV_FORMAT_NODE type.
	ErrResponseDataNotExpectedFormatNode = errors.New("response data is not of expected MPV_FORMAT_NODE type")

	// ErrPlaybackTimeNotFloat occurs when playback time is not a correct decimal number.
	ErrPlaybackTimeNotFloat = errors.New("playback time could not be converted to a float number")
	// ErrPlaybackPathNotServed occurs when playback path is set to file that is not being served by api.
	ErrPlaybackPathNotServed = errors.New("playback path is not served")

	// ErrPlaylistMapDataNotParsable occurs when data being sent during observation of MPV's "playlist" property is not a correct JSON.
	ErrPlaylistMapDataNotParsable = errors.New("could not parse playlist map data as JSON")
)

func (s *Server) handleFullscreenEvent(res mpv.ObservePropertyResponse) error {
	enabled, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	s.playback.SetFullscreen(enabled == mpv.YesValue)
	return nil
}

func (s *Server) handleLoopFileEvent(res mpv.ObservePropertyResponse) error {
	enabled, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	s.playback.SetLoopFile(enabled != mpv.NoValue)
	return nil
}

func (s *Server) handlePauseEvent(res mpv.ObservePropertyResponse) error {
	paused, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	s.playback.SetPause(paused == mpv.YesValue)
	return nil
}

func (s *Server) handleAudioIDChangeEvent(res mpv.ObservePropertyResponse) error {
	aid, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	s.playback.SetAudioID(aid)
	return nil
}

func (s *Server) handlePlaylistProperty(res mpv.ObservePropertyResponse) error {
	playlistItems, ok := res.Data.(mpv.PlaylistFormatNodeArray)
	if !ok {
		return ErrResponseDataNotExpectedFormatNode
	}

	// TODO: below should be used something resembling a set - the playlist will fire at every possible change to the MPV map type,
	// which due to having flag specifying which of the item is currently played will result in firing with the same items array,
	// even though the list did not change.
	items := []string{}
	for _, playlistItem := range playlistItems {
		items = append(items, playlistItem.Filename)
	}

	if !s.playback.PlaylistSelected() {
		newPlaylist := state.NewPlaylist(state.PlaylistConfig{})
		s.playlists.AddPlaylist(newPlaylist)
		s.playback.SelectPlaylist(newPlaylist.UUID())
	}

	return s.playlists.SetPlaylistItems(s.playback.PlaylistUUID(), items)
}

func (s *Server) handlePlaylistPlayingPosEvent(res mpv.ObservePropertyResponse) error {
	idxStr, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return ErrResponseDataNotInt
	}

	s.playback.SelectPlaylistCurrentIdx(idx)
	return nil
}

func (s *Server) handleSubtitleIDChangeEvent(res mpv.ObservePropertyResponse) error {
	sid, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	s.playback.SetSubtitleID(sid)
	return nil
}

func (s *Server) handleChapterChangeEvent(res mpv.ObservePropertyResponse) error {
	chapterIdx, ok := res.Data.(int64)
	if !ok {
		return ErrResponseDataNotInt
	}

	s.playback.SetCurrentChapter(chapterIdx)
	return nil
}

func (s *Server) handlePathEvent(res mpv.ObservePropertyResponse) error {
	if res.Data == nil {
		s.playback.Stop()

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

	s.playback.SetMovie(movie)
	return nil
}

func (s *Server) handlePlaybackTimeEvent(res mpv.ObservePropertyResponse) error {
	currentTime, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	if currentTime == "" {
		s.playback.SetPlaybackTime(0)

		return nil
	}

	currentTimeNum, err := strconv.ParseFloat(currentTime, 64)
	if err != nil {
		return ErrPlaybackTimeNotFloat
	}

	s.playback.SetPlaybackTime(currentTimeNum)
	return nil
}
