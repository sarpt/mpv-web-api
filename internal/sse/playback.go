package sse

import (
	"github.com/sarpt/mpv-web-api/internal/state"
)

const (
	playbackSSEChannelVariant state.SSEChannelVariant = "playback"

	playbackAllSseEvent    = "all"
	playbackReplaySseEvent = "replay"
)

func (s *Server) createPlaybackReplayHandler() sseReplayHandler {
	return func(res ResponseWriter) error {
		return res.sendChange(s.playback, playbackSSEChannelVariant, playbackReplaySseEvent)
	}
}

func (s *Server) createPlaybackChangesHandler() sseChangeHandler {
	return func(res ResponseWriter, changes interface{}) error {
		change, ok := changes.(state.PlaybackChange)
		if !ok {
			return errIncorrectChangesType
		}

		return res.sendChange(s.playback, playbackSSEChannelVariant, string(change.Variant))
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
