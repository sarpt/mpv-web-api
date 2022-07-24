package sse

import (
	"encoding/json"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/directories"
	state_sse "github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
)

const (
	directoriesSSEChannelVariant state_sse.ChannelVariant = "directories"
)

type directoriesMapChange struct {
	Directories map[string]directories.Entry
}

func (dmc directoriesMapChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(dmc.Directories)
}

type directoriesChangesBroadcaster struct {
	directories *directories.Storage
	ChangesBroadcaster[directories.Change]
}

func (dc *directoriesChangesBroadcaster) Replay(res ResponseWriter) error {
	return res.SendChange(directoriesMapChange{Directories: dc.directories.All()}, directoriesSSEChannelVariant, string(directories.AddedDirectoriesChange))
}

func (dc *directoriesChangesBroadcaster) ChangeHandler(res ResponseWriter, change directories.Change) error {
	return res.SendChange(change, directoriesSSEChannelVariant, string(directories.AddedDirectoriesChange))
}

func NewDirectoriesChannel(storage *directories.Storage) *StateChannel[directories.Change] {
	return &StateChannel[directories.Change]{
		&directoriesChangesBroadcaster{
			storage,
			NewChangesBroadcaster[directories.Change](),
		},
		directoriesSSEChannelVariant,
	}
}
