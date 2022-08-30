package api

import (
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"
)

type playbackTrigger interface {
	handler(change playback.Change, api PluginApi)
}

func (s *Server) addPlaybackTrigger(mediaFilePath string, trigger playbackTrigger) {
	s.statesRepository.Playback().Subscribe(func(change playback.Change) {
		if s.statesRepository.Playback().MediaFilePath() != mediaFilePath {
			return
		}

		trigger.handler(change, s)
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

func (t *chaptersManagerPlaybackTrigger) handler(change playback.Change, api PluginApi) {
	if change.Variant() != playback.CurrentChapterIdxChange {
		return
	}

	if len(t.chaptersOrder) < 1 {
		return
	}

	if t.currentChapterIdx+1 >= len(t.chaptersOrder) {
		t.currentChapterIdx = 0
		return
	}

	t.currentChapterIdx += 1
	nextChapter := t.chaptersOrder[t.currentChapterIdx]

	newChapter, ok := change.Value.(int64)
	if !ok {
		return // TODO: add cast error handling instead silently ignoring it
	}

	if newChapter == nextChapter {
		return
	}

	api.ChangeChapter(nextChapter)
}