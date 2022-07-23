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

type directoriesChannel struct {
	StateChannel[*directories.Storage, directories.Change]
}

func newDirectoriesChannel(directoriesStorage *directories.Storage) *directoriesChannel {
	return &directoriesChannel{
		NewStateChannel[*directories.Storage, directories.Change](directoriesStorage, directoriesSSEChannelVariant),
	}
}

func (dc *directoriesChannel) Replay(res ResponseWriter) error {
	return res.SendChange(directoriesMapChange{Directories: dc.state.All()}, dc.Variant(), string(directories.AddedDirectoriesChange))
}

func (dc *directoriesChannel) changeHandler(res ResponseWriter, change directories.Change) error {
	return res.SendChange(change, dc.Variant(), string(directories.AddedDirectoriesChange))
}
