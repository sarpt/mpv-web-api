package sse

import (
	"github.com/sarpt/mpv-web-api/pkg/state"
)

const (
	playbackSSEChannelVariant state.SSEChannelVariant = "playback"

	playbackAllSseEvent    = "all"
	playbackReplaySseEvent = "replay"
)

func (s *Server) createPlaybackReplayHandler() sseReplayHandler {
	return func(res ResponseWriter) error {
		return res.SendChange(s.playback, playbackSSEChannelVariant, playbackReplaySseEvent)
	}
}

func (s *Server) createPlaybackChangesHandler() sseChangeHandler {
	return func(res ResponseWriter, changes interface{}) error {
		change, ok := changes.(state.PlaybackChange)
		if !ok {
			return errIncorrectChangesType
		}

		if s.playback.Stopped { // TODO: the changes are shot by state.Playback even after the mediaFilePath is cleared, as such it may be wasteful to push further changes through SSE. to think of a way to reduce number of those blank data calls after closing stopping playback
			return res.SendEmptyChange(playbackSSEChannelVariant, string(change.Variant))
		}

		return res.SendChange(s.playback, playbackSSEChannelVariant, string(change.Variant))
	}
}

func (s *Server) playbackSSEChannel() channel {
	return channel{
		variant:       playbackSSEChannelVariant,
		observers:     s.playbackObservers,
		changeHandler: s.createPlaybackChangesHandler(),
		replayHandler: s.createPlaybackReplayHandler(),
	}
}

// playbackChangesToChannelObserversDistributor is a fan-out dispatcher, which notifies all playback observers (subscribers from SSE etc.) when a playbackChange occurs.
func playbackChangesToChannelObserversDistributor(channelObservers observers) func(change state.PlaybackChange) {
	return func(change state.PlaybackChange) {
		channelObservers.lock.RLock()
		for _, observer := range channelObservers.items {
			observer <- change
		}
		channelObservers.lock.RUnlock()
	}
}
