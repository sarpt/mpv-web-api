package api

import (
	"errors"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"
)

var (
	errMediaFileNotString        = errors.New("media file path in change is not a string")
	errChapterNotNumber          = errors.New("chapter in change is not a number")
	errChaptersListIncorrectSize = errors.New("chapters list should not be less than 1 element")
)

type playbackTrigger interface {
	handler(change playback.Change) error
}

func (s *Server) addPlaybackTrigger(trigger playbackTrigger) func() {
	return s.statesRepository.Playback().Subscribe(func(change playback.Change) {
		err := trigger.handler(change)
		if err != nil {
			s.errLog.Printf("playback trigger for media file returned error: %s", err)
		}
	}, func(err error) {})
}

type mediaFileChangeTrigger struct {
	targetMediaFilePath string
	done                chan<- bool
}

func newMediaFileChangeTrigger(targetMediaFilePath string, done chan<- bool) (*mediaFileChangeTrigger, error) {
	return &mediaFileChangeTrigger{
		targetMediaFilePath: targetMediaFilePath,
		done:                done,
	}, nil
}

func (t *mediaFileChangeTrigger) handler(change playback.Change) error {
	if change.Variant() != playback.MediaFileChange {
		return nil
	}

	newMediaFilePath, ok := change.Value.(string)
	if !ok {
		return errMediaFileNotString
	}

	if newMediaFilePath != t.targetMediaFilePath {
		return nil
	}

	t.done <- true

	return nil
}

type chaptersManagerPlaybackTrigger struct {
	api               PluginApi
	chaptersOrder     []int64
	currentChapterIdx int
}

func newChaptersManagerPlaybackTrigger(chaptersOrder []int64, api PluginApi) (*chaptersManagerPlaybackTrigger, error) {
	if len(chaptersOrder) < 1 {
		return nil, errChaptersListIncorrectSize
	}

	return &chaptersManagerPlaybackTrigger{
		api:               api,
		chaptersOrder:     chaptersOrder,
		currentChapterIdx: -1,
	}, nil
}

func (t *chaptersManagerPlaybackTrigger) handler(change playback.Change) error {
	if change.Variant() != playback.CurrentChapterIdxChange {
		return nil
	}

	if t.currentChapterIdx+1 >= len(t.chaptersOrder) {
		return nil
	}

	newChapter, ok := change.Value.(int64)
	if !ok {
		return errChapterNotNumber
	}

	var currentChapter int64 = -1
	if t.currentChapterIdx >= 0 {
		currentChapter = t.chaptersOrder[t.currentChapterIdx]
	}

	nextChapter := t.chaptersOrder[t.currentChapterIdx+1]
	if currentChapter == newChapter || nextChapter == newChapter {
		return nil
	}

	t.currentChapterIdx += 1
	return t.api.ChangeChapter(nextChapter)
}
