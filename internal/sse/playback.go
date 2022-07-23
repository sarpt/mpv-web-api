package sse

import (
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"
	state_sse "github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
)

const (
	playbackSSEChannelVariant state_sse.ChannelVariant = "playback"

	playbackAllSseEvent    = "all"
	playbackReplaySseEvent = "replay"
)

type playbackChannel struct {
	StateChannel[*playback.Storage, playback.Change]
}

func newPlaybackChannel(playbackStorage *playback.Storage) *playbackChannel {
	return &playbackChannel{
		NewStateChannel[*playback.Storage, playback.Change](playbackStorage, playbackSSEChannelVariant),
	}
}

func (pc *playbackChannel) Replay(res ResponseWriter) error {
	return res.SendChange(pc.state, pc.Variant(), playbackReplaySseEvent)
}

func (pc *playbackChannel) changeHandler(res ResponseWriter, change playback.Change) error {
	if pc.state.Stopped { // TODO: the changes are shot by state.Playback even after the mediaFilePath is cleared, as such it may be wasteful to push further changes through SSE. to think of a way to reduce number of those blank data calls after closing stopping playback
		return res.SendEmptyChange(pc.Variant(), string(change.ChangeVariant))
	}

	return res.SendChange(pc.state, pc.Variant(), string(change.ChangeVariant))
}
