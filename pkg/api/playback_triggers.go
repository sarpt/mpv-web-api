package api

import (
	"errors"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"
)

type MediaFileChangeTriggerNotification string
type ChapterManagerTriggerNotification string

var (
	errMediaFileNotString        = errors.New("media file path in change is not a string")
	errChapterNotNumber          = errors.New("chapter in change is not a number")
	errChaptersListIncorrectSize = errors.New("chapters list should not be less than 1 element")

	ChangedMediaFileDoesNotMatch MediaFileChangeTriggerNotification = "Changed media file does not match provided target"
	ChangedMediaFileMatches      MediaFileChangeTriggerNotification = "Changed media file matches provided target"

	ChaptersIterationDone     ChapterManagerTriggerNotification = "All provided chapters iterated"
	NextChapterAlreadyPlaying ChapterManagerTriggerNotification = "Next chapter in iteration order is already playing"
	TriggeringChapterChange   ChapterManagerTriggerNotification = "Triggering next chapter change"
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
	notifications       chan<- MediaFileChangeTriggerNotification
}

func newMediaFileChangeTrigger(targetMediaFilePath string, notifications chan<- MediaFileChangeTriggerNotification) (*mediaFileChangeTrigger, error) {
	return &mediaFileChangeTrigger{
		targetMediaFilePath: targetMediaFilePath,
		notifications:       notifications,
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
		t.notifications <- ChangedMediaFileDoesNotMatch
		return nil
	}

	t.notifications <- ChangedMediaFileMatches

	return nil
}

type chaptersManagerTrigger struct {
	api               PluginApi
	chaptersOrder     []int64
	currentChapterIdx int
	notifications     chan<- ChapterManagerTriggerNotification
}

func newChaptersManagerTrigger(chaptersOrder []int64, api PluginApi, notifications chan<- ChapterManagerTriggerNotification) (*chaptersManagerTrigger, error) {
	if len(chaptersOrder) < 1 {
		return nil, errChaptersListIncorrectSize
	}

	return &chaptersManagerTrigger{
		api:               api,
		chaptersOrder:     chaptersOrder,
		currentChapterIdx: -1,
		notifications:     notifications,
	}, nil
}

func (t *chaptersManagerTrigger) handler(change playback.Change) error {
	if change.Variant() != playback.CurrentChapterIdxChange {
		return nil
	}

	if t.currentChapterIdx+1 >= len(t.chaptersOrder) {
		t.notifications <- ChaptersIterationDone
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
		t.notifications <- NextChapterAlreadyPlaying
		return nil
	}

	t.currentChapterIdx += 1
	t.notifications <- TriggeringChapterChange
	return t.api.ChangeChapter(nextChapter)
}
