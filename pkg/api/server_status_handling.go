package api

import "fmt"

func (s *Server) createStatusReplayHandler() sseReplayHandler {
	return func(res SSEResponseWriter) error {
		return s.status.sendStatus(statusReplay, res)
	}
}

func (s *Server) createStatusChangeHandler() sseChangeHandler {
	return func(res SSEResponseWriter, changes interface{}) error {
		statusChange, ok := changes.(StatusChange)
		if !ok {
			return errIncorrectChangesType
		}

		return s.status.sendStatus(statusChange.Variant, res)
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

func (s *Status) sendStatus(variant StatusChangeVariant, res SSEResponseWriter) error {
	out, err := s.jsonMarshal()
	if err != nil {
		return fmt.Errorf("%w: %s", errResponseJSONCreationFailed, err)
	}

	_, err = res.Write(formatSseEvent(string(variant), out))
	if err != nil {
		return fmt.Errorf("sending status failed: %w: %s", errClientWritingFailed, err)
	}

	return nil
}
