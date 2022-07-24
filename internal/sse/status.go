package sse

import (
	state_sse "github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/status"
)

const (
	statusSSEChannelVariant state_sse.ChannelVariant = "status"

	// statusReplay notifies about replay of status state.
	statusReplay state_sse.ChangeVariant = "replay"
)

type statusChangesBroadcaster struct {
	status *status.Storage
	ChangesBroadcaster[status.Change]
}

func (sc *statusChangesBroadcaster) Replay(res ResponseWriter) error {
	return res.SendChange(sc.status, statusSSEChannelVariant, string(statusReplay))
}

func (sc *statusChangesBroadcaster) ChangeHandler(res ResponseWriter, change status.Change) error {
	return res.SendChange(sc.status, statusSSEChannelVariant, string(change.ChangeVariant))
}

func NewStatusChannel(storage *status.Storage) *StateChannel[status.Change] {
	return &StateChannel[status.Change]{
		&statusChangesBroadcaster{
			storage,
			NewChangesBroadcaster[status.Change](),
		},
		statusSSEChannelVariant,
	}
}
