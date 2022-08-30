package api

import (
	"errors"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"
)

var (
	errChapterNotNumber = errors.New("chapter in change is not a number")
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

func newChaptersManagerPlaybackTrigger(chaptersOrder []int64) *chaptersManagerPlaybackTrigger {
	return &chaptersManagerPlaybackTrigger{
		chaptersOrder: chaptersOrder,
	}
}

func (t *chaptersManagerPlaybackTrigger) handler(change playback.Change, api PluginApi) error {
	if change.Variant() != playback.CurrentChapterIdxChange {
		return nil
	}

	if len(t.chaptersOrder) < 1 {
		return nil
	}

	if t.currentChapterIdx+1 >= len(t.chaptersOrder) {
		t.currentChapterIdx = 0
		return nil
	}

	t.currentChapterIdx += 1
	nextChapter := t.chaptersOrder[t.currentChapterIdx]

	newChapter, ok := change.Value.(int64)
	if !ok {
		return errChapterNotNumber
	}

	if newChapter == nextChapter {
		return nil
	}

	return api.ChangeChapter(nextChapter)
}
