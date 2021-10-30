package sse

import (
	"github.com/sarpt/mpv-web-api/pkg/state"
)

const (
	statusSSEChannelVariant state.SSEChannelVariant = "status"
)

func (s *Server) createStatusReplayHandler() sseReplayHandler {
	return func(res ResponseWriter) error {
		return res.SendChange(s.status, statusSSEChannelVariant, string(state.StatusReplay))
	}
}

func (s *Server) createStatusChangeHandler() sseChangeHandler {
	return func(res ResponseWriter, changes interface{}) error {
		statusChange, ok := changes.(state.StatusChange)
		if !ok {
			return errIncorrectChangesType
		}

		return res.SendChange(s.status, statusSSEChannelVariant, string(statusChange.Variant))
	}
}

func (s *Server) statusSSEChannel() channel {
	return channel{
		variant:       statusSSEChannelVariant,
		observers:     s.statusObservers,
		changeHandler: s.createStatusChangeHandler(),
		replayHandler: s.createStatusReplayHandler(),
	}
}

// statusChangesToChannelObserversDistributor is a fan-out dispatcher, which notifies all playback observers (subscribers from SSE etc.) when a playbackChange occurs.
func statusChangesToChannelObserversDistributor(channelObservers observers) func(change state.StatusChange) {
	return func(change state.StatusChange) {
		channelObservers.lock.RLock()
		for _, observer := range channelObservers.items {
			observer <- change
		}
		channelObservers.lock.RUnlock()
	}
}
