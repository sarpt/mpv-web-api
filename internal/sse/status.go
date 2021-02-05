package sse

import (
	"github.com/sarpt/mpv-web-api/internal/state"
)

const (
	statusSSEChannelVariant state.SSEChannelVariant = "status"
)

func (s *Server) createStatusReplayHandler() sseReplayHandler {
	return func(res ResponseWriter) error {
		return res.sendChange(s.status, statusSSEChannelVariant, string(state.StatusReplay))
	}
}

func (s *Server) createStatusChangeHandler() sseChangeHandler {
	return func(res ResponseWriter, changes interface{}) error {
		statusChange, ok := changes.(state.StatusChange)
		if !ok {
			return errIncorrectChangesType
		}

		return res.sendChange(s.status, statusSSEChannelVariant, string(statusChange.Variant))
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
