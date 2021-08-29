package sse

import (
	"encoding/json"

	"github.com/sarpt/mpv-web-api/internal/state"
)

const (
	mediaFilesSSEChannelVariant state.SSEChannelVariant = "mediaFiles"
)

type mediaFilesMapChange struct {
	MediaFiles map[string]state.MediaFile
}

func (mmc mediaFilesMapChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(mmc.MediaFiles)
}

func (s *Server) createMediaFilesReplayHandler() sseReplayHandler {
	return func(res ResponseWriter) error {
		return res.SendChange(mediaFilesMapChange{MediaFiles: s.mediaFiles.All()}, mediaFilesSSEChannelVariant, string(state.AddedMediaFilesChange))
	}
}

func (s *Server) createMediaFilesChangeHandler() sseChangeHandler {
	return func(res ResponseWriter, changes interface{}) error {
		mediaFilesChange, ok := changes.(state.MediaFilesChange)
		if !ok {
			return errIncorrectChangesType
		}

		return res.SendChange(mediaFilesChange, mediaFilesSSEChannelVariant, string(state.AddedMediaFilesChange))
	}
}

func (s *Server) mediaFilesSSEChannel() channel {
	return channel{
		variant:       mediaFilesSSEChannelVariant,
		observers:     s.mediaFilesObservers,
		changeHandler: s.createMediaFilesChangeHandler(),
		replayHandler: s.createMediaFilesReplayHandler(),
	}
}
