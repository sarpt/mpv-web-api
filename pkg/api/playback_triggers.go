package api

import (
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"
)

type playbackTriggerCondition = func(change playback.Change) bool
type playbackTriggerHandler = func(api PluginApi)

type playbackTrigger interface {
	handler(change playback.Change, api PluginApi)
}

func (s *Server) addPlaybackTrigger(mediaFile string, trigger playbackTrigger) {
	s.statesRepository.Playback().Subscribe(func(change playback.Change) {
		trigger.handler(change, s)
	}, func(err error) {})
}

type chaptersManagerPlaybackTrigger struct {
	chaptersOrder     []int64
	currentChapterIdx int
}

func newSkipChapterPlaybackTrigger(chaptersOrder []int64) *chaptersManagerPlaybackTrigger {
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
