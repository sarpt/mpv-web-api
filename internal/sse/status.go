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

type statusChannel struct {
	StateChannel[*status.Storage, status.Change]
}

func newStatusChannel(statusStorage *status.Storage) *statusChannel {
	return &statusChannel{
		NewStateChannel[*status.Storage, status.Change](statusStorage, statusSSEChannelVariant),
	}
}

func (sc *statusChannel) Replay(res ResponseWriter) error {
	return res.SendChange(sc.state, sc.Variant(), string(statusReplay))
}

func (sc *statusChannel) changeHandler(res ResponseWriter, change status.Change) error {
	return res.SendChange(sc.state, sc.Variant(), string(change.ChangeVariant))
}
