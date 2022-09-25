package playback_triggers

import (
	"errors"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"
)

type MediaFileChangeNotification string

var (
	errMediaFileNotString = errors.New("media file path in change is not a string")

	ChangedMediaFileDoesNotMatch MediaFileChangeNotification = "Changed media file does not match provided target"
	ChangedMediaFileMatches      MediaFileChangeNotification = "Changed media file matches provided target"
)

type MediaFileChange struct {
	targetMediaFilePath string
	notifications       chan<- MediaFileChangeNotification
}

func NewMediaFileChange(targetMediaFilePath string, notifications chan<- MediaFileChangeNotification) (*MediaFileChange, error) {
	return &MediaFileChange{
		targetMediaFilePath: targetMediaFilePath,
		notifications:       notifications,
	}, nil
}

func (t *MediaFileChange) Handler(change playback.Change) error {
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
