package sse

import (
	"encoding/json"

	"github.com/sarpt/mpv-web-api/internal/common"
	"github.com/sarpt/mpv-web-api/internal/state"
)

const (
	directoriesSSEChannelVariant state.SSEChannelVariant = "directories"
)

type directoriesMapChange struct {
	Directories map[string]common.Directory
}

func (dmc directoriesMapChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(dmc.Directories)
}

func (s *Server) createDirectoriesReplayHandler() sseReplayHandler {
	return func(res ResponseWriter) error {
		return res.SendChange(directoriesMapChange{Directories: s.directories.All()}, directoriesSSEChannelVariant, string(state.AddedDirectoriesChange))
	}
}

func (s *Server) createDirectoriesChangeHandler() sseChangeHandler {
	return func(res ResponseWriter, changes interface{}) error {
		directoriesChange, ok := changes.(state.DirectoriesChange)
		if !ok {
			return errIncorrectChangesType
		}

		return res.SendChange(directoriesChange, directoriesSSEChannelVariant, string(state.AddedDirectoriesChange))
	}
}

func (s *Server) directoriesSSEChannel() channel {
	return channel{
		variant:       directoriesSSEChannelVariant,
		observers:     s.mediaFilesObservers,
		changeHandler: s.createDirectoriesChangeHandler(),
		replayHandler: s.createDirectoriesReplayHandler(),
	}
}
