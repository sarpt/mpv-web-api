package api

import (
	"encoding/json"
	"fmt"

	"github.com/sarpt/mpv-web-api/internal/state"
)

var (
	statusSSEChannelVariant state.SSEChannelVariant = "status"
)

func (s *Server) createStatusReplayHandler() sseReplayHandler {
	return func(res SSEResponseWriter) error {
		return sendStatus(s.status, state.StatusReplay, res)
	}
}

func (s *Server) createStatusChangeHandler() sseChangeHandler {
	return func(res SSEResponseWriter, changes interface{}) error {
		statusChange, ok := changes.(state.StatusChange)
		if !ok {
			return errIncorrectChangesType
		}

		return sendStatus(s.status, statusChange.Variant, res)
	}
}

func (s *Server) statusSSEChannel() SSEChannel {
	return SSEChannel{
		Variant:       statusSSEChannelVariant,
		Observers:     s.statusSSEObservers,
		ChangeHandler: s.createStatusChangeHandler(),
		ReplayHandler: s.createStatusReplayHandler(),
	}
}

func sendStatus(status *state.Status, variant state.StatusChangeVariant, res SSEResponseWriter) error {
	out, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("%w: %s", errResponseJSONCreationFailed, err)
	}

	_, err = res.Write(formatSseEvent(statusSSEChannelVariant, string(variant), out))
	if err != nil {
		return fmt.Errorf("sending status failed: %w: %s", errClientWritingFailed, err)
	}

	return nil
}
