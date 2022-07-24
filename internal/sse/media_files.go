package sse

import (
	"encoding/json"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/media_files"
	state_sse "github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
)

const (
	mediaFilesSSEChannelVariant state_sse.ChannelVariant = "mediaFiles"
)

type mediaFilesMapChange struct {
	MediaFiles map[string]media_files.Entry
}

func (mmc mediaFilesMapChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(mmc.MediaFiles)
}

type mediaFilesChangesBroadcaster struct {
	mediaFiles *media_files.Storage
	ChangesBroadcaster[media_files.Change]
}

func (mfc *mediaFilesChangesBroadcaster) Replay(res ResponseWriter) error {
	return res.SendChange(mediaFilesMapChange{MediaFiles: mfc.mediaFiles.All()}, mediaFilesSSEChannelVariant, string(media_files.AddedMediaFilesChange))
}

func (mfc *mediaFilesChangesBroadcaster) ChangeHandler(res ResponseWriter, change media_files.Change) error {
	return res.SendChange(change, mediaFilesSSEChannelVariant, string(change.ChangeVariant))
}

func NewMediaFilesChannel(storage *media_files.Storage) *StateChannel[media_files.Change] {
	return &StateChannel[media_files.Change]{
		&mediaFilesChangesBroadcaster{
			storage,
			NewChangesBroadcaster[media_files.Change](),
		},
		mediaFilesSSEChannelVariant,
	}
}
