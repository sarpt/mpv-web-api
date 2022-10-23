package playback_triggers

import (
	"errors"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"
)

type ChapterManagerNotification string

var (
	errChapterNotNumber          = errors.New("chapter in change is not a number")
	errChaptersListIncorrectSize = errors.New("chapters list should not be less than 1 element")

	ChaptersIterationDone           ChapterManagerNotification = "All provided chapters iterated"
	NextChapterAlreadyPlaying       ChapterManagerNotification = "Next chapter in iteration order is already playing"
	TriggeringChapterChange         ChapterManagerNotification = "Triggering next chapter change"
	MediaFileChangedDuringIteration ChapterManagerNotification = "Media file changed during chapters iteration"
)

type Api interface {
	ChangeChapter(idx int64) error
}

type ChaptersManager struct {
	api               Api
	chaptersOrder     []int64
	currentChapterIdx int
	mediaFilePath     string
	notifications     chan<- ChapterManagerNotification
}

func NewChaptersManager(mediaFilePath string, chaptersOrder []int64, api Api, notifications chan<- ChapterManagerNotification) (*ChaptersManager, error) {
	if len(chaptersOrder) < 1 {
		return nil, errChaptersListIncorrectSize
	}

	return &ChaptersManager{
		api:               api,
		chaptersOrder:     chaptersOrder,
		currentChapterIdx: -1,
		mediaFilePath:     mediaFilePath,
		notifications:     notifications,
	}, nil
}

func (t *ChaptersManager) Handler(change playback.Change) error {
	handleChange := change.Variant() == playback.CurrentChapterIdxChange || change.Variant() == playback.MediaFileChange
	if !handleChange {
		return nil
	}

	if change.Variant() == playback.MediaFileChange {
		newMediaFilePath, ok := change.Value.(string)
		if !ok {
			return errMediaFileNotString
		}

		if newMediaFilePath != t.mediaFilePath {
			t.notifications <- MediaFileChangedDuringIteration
			return nil
		}

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

	if currentChapter == newChapter {
		t.notifications <- NextChapterAlreadyPlaying
		return nil
	}

	if t.currentChapterIdx+1 >= len(t.chaptersOrder) {
		t.notifications <- ChaptersIterationDone
		return nil
	}

	t.currentChapterIdx += 1
	nextChapter := t.chaptersOrder[t.currentChapterIdx]
	if nextChapter == newChapter {
		t.notifications <- NextChapterAlreadyPlaying
		return nil
	}

	t.notifications <- TriggeringChapterChange
	return t.api.ChangeChapter(nextChapter)
}
