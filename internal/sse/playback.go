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

type playbackChangesBroadcaster struct {
	playback *playback.Storage
	ChangesBroadcaster[playback.Change]
}

func (pc *playbackChangesBroadcaster) Replay(res ResponseWriter) error {
	return res.SendChange(pc.playback, playbackSSEChannelVariant, playbackReplaySseEvent)
}

func (pc *playbackChangesBroadcaster) ChangeHandler(res ResponseWriter, change playback.Change) error {
	if pc.playback.Stopped { // TODO: the changes are shot by state.Playback even after the mediaFilePath is cleared, as such it may be wasteful to push further changes through SSE. to think of a way to reduce number of those blank data calls after closing stopping playback
		return res.SendEmptyChange(playbackSSEChannelVariant, string(change.ChangeVariant))
	}

	return res.SendChange(pc.playback, playbackSSEChannelVariant, string(change.ChangeVariant))
}

func NewPlaybackChannel(storage *playback.Storage) *StateChannel[playback.Change] {
	return &StateChannel[playback.Change]{
		&playbackChangesBroadcaster{
			storage,
			NewChangesBroadcaster[playback.Change](),
		},
		playbackSSEChannelVariant,
	}
}
