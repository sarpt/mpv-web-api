package api

import (
	"errors"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"
)

var (
	errChapterNotNumber          = errors.New("chapter in change is not a number")
	errChaptersListIncorrectSize = errors.New("chapters list should not be less than 1 element")
)

type playbackTrigger interface {
	handler(change playback.Change, api PluginApi) error
}

func (s *Server) addPlaybackTrigger(mediaFilePath string, trigger playbackTrigger) {
	s.statesRepository.Playback().Subscribe(func(change playback.Change) {
		if s.statesRepository.Playback().MediaFilePath() != mediaFilePath {
			return
		}

		err := trigger.handler(change, s)
		if err != nil {
			s.errLog.Printf("playback trigger for media file \"%s\" returned error: %s", mediaFilePath, err)
		}
	}, func(err error) {})
}

type chaptersManagerPlaybackTrigger struct {
	chaptersOrder     []int64
	currentChapterIdx int
}

func newChaptersManagerPlaybackTrigger(chaptersOrder []int64) (*chaptersManagerPlaybackTrigger, error) {
	if len(chaptersOrder) < 1 {
		return nil, errChaptersListIncorrectSize
	}

	return &chaptersManagerPlaybackTrigger{
		chaptersOrder: chaptersOrder,
	}, nil
}

func (t *chaptersManagerPlaybackTrigger) handler(change playback.Change, api PluginApi) error {
	if change.Variant() != playback.CurrentChapterIdxChange {
		return nil
	}

	if t.currentChapterIdx+1 >= len(t.chaptersOrder) {
		t.currentChapterIdx = 0
		return nil
	}

	newChapter, ok := change.Value.(int64)
	if !ok {
		return errChapterNotNumber
	}

	t.currentChapterIdx += 1
	nextChapter := t.chaptersOrder[t.currentChapterIdx]

	if newChapter == nextChapter {
		return nil
	}

	return api.ChangeChapter(nextChapter)
}
