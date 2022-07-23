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

type mediaFilesChannel struct {
	StateChannel[*media_files.Storage, media_files.Change]
}

func newMediaFilesChannel(mediaFilesStorage *media_files.Storage) *mediaFilesChannel {
	return &mediaFilesChannel{
		NewStateChannel[*media_files.Storage, media_files.Change](mediaFilesStorage, mediaFilesSSEChannelVariant),
	}
}

func (mfc *mediaFilesChannel) Replay(res ResponseWriter) error {
	return res.SendChange(mediaFilesMapChange{MediaFiles: mfc.state.All()}, mfc.Variant(), string(media_files.AddedMediaFilesChange))
}

func (mfc *mediaFilesChannel) changeHandler(res ResponseWriter, change media_files.Change) error {
	return res.SendChange(change, mfc.Variant(), string(change.ChangeVariant))
}
