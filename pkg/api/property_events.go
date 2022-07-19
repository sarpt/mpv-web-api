package api

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/sarpt/mpv-web-api/pkg/mpv"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playlists"
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

	s.statesRepository.Playback().SetFullscreen(enabled == mpv.YesValue)
	return nil
}

func (s *Server) handleLoopFileEvent(res mpv.ObservePropertyResponse) error {
	enabled, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	s.statesRepository.Playback().SetLoopFile(enabled != mpv.NoValue)
	return nil
}

func (s *Server) handlePauseEvent(res mpv.ObservePropertyResponse) error {
	paused, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	s.statesRepository.Playback().SetPause(paused == mpv.YesValue)
	return nil
}

func (s *Server) handleAudioIDChangeEvent(res mpv.ObservePropertyResponse) error {
	aid, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	s.statesRepository.Playback().SetAudioID(aid)
	return nil
}

func (s *Server) handlePlaylistProperty(res mpv.ObservePropertyResponse) error {
	currentPlaylist, err := s.statesRepository.Playlists().ByUUID(s.statesRepository.Playback().PlaylistUUID())
	if err != nil {
		return fmt.Errorf("selected playlist UUID does not point to an existing playlist: %s", err)
	}

	playlistItems, ok := res.Data.(mpv.PlaylistFormatNodeArray)
	if !ok {
		return ErrResponseDataNotExpectedFormatNode
	}

	entries := []playlists.Entry{}
	for _, playlistItem := range playlistItems {
		entries = append(entries, playlists.Entry{
			Path: playlistItem.Filename,
		})
	}

	if !currentPlaylist.EntriesDiffer(entries) {
		return nil
	}

	if !s.DefaultPlaylistSelected() {
		// To prevent unwanted changes to a named playlist when entries don't match, a default playlist
		// should be selected and modified. Mismatched entries for a named playlist suggest
		// changes introduced from outside the server.
		s.outLog.Printf("entries do not match for a named, not-default playlist (uuid: %s) - switching to a default playlist", s.statesRepository.Playback().PlaylistUUID())
		s.statesRepository.Playback().SelectPlaylist(s.defaultPlaylistUUID)
	}

	return s.statesRepository.Playlists().SetPlaylistEntries(s.statesRepository.Playback().PlaylistUUID(), entries)
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

	s.statesRepository.Playback().SelectPlaylistCurrentIdx(idx)
	return nil
}

func (s *Server) handleSubtitleIDChangeEvent(res mpv.ObservePropertyResponse) error {
	sid, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	s.statesRepository.Playback().SetSubtitleID(sid)
	return nil
}

func (s *Server) handleChapterChangeEvent(res mpv.ObservePropertyResponse) error {
	chapterIdx, ok := res.Data.(int64)
	if !ok {
		return ErrResponseDataNotInt
	}

	s.statesRepository.Playback().SetCurrentChapter(chapterIdx)
	return nil
}

func (s *Server) handlePathEvent(res mpv.ObservePropertyResponse) error {
	if res.Data == nil {
		s.statesRepository.Playback().Stop()

		return nil
	}

	path, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	mediaFile, err := s.statesRepository.MediaFiles().ByPath(path)
	if err != nil {
		return fmt.Errorf("%w:%s", ErrPlaybackPathNotServed, path)
	}

	s.statesRepository.Playback().SetMediaFile(mediaFile)
	return nil
}

func (s *Server) handlePlaybackTimeEvent(res mpv.ObservePropertyResponse) error {
	currentTime, ok := res.Data.(string)
	if !ok {
		return ErrResponseDataNotString
	}

	if currentTime == "" {
		s.statesRepository.Playback().SetPlaybackTime(0)

		return nil
	}

	currentTimeNum, err := strconv.ParseFloat(currentTime, 64)
	if err != nil {
		return ErrPlaybackTimeNotFloat
	}

	s.statesRepository.Playback().SetPlaybackTime(currentTimeNum)
	return nil
}
