package sse

import "github.com/sarpt/mpv-web-api/internal/state"

const (
	playlistSSEChannelVariant state.SSEChannelVariant = "playlist"
)

func (s *Server) createPlaylistReplayHandler() sseReplayHandler {
	return func(res ResponseWriter) error {
		return res.SendChange(s.playlist, playlistSSEChannelVariant, string(state.PlaylistReplay))
	}
}

func (s *Server) createPlaylistChangesHandler() sseChangeHandler {
	return func(res ResponseWriter, changes interface{}) error {
		change, ok := changes.(state.PlaylistChange)
		if !ok {
			return errIncorrectChangesType
		}

		return res.SendChange(s.playlist, playlistSSEChannelVariant, string(change.Variant))
	}
}

func (s *Server) playlistSSEChannel() channel {
	return channel{
		variant:       playlistSSEChannelVariant,
		observers:     s.playlistObservers,
		changeHandler: s.createPlaylistChangesHandler(),
		replayHandler: s.createPlaylistReplayHandler(),
	}
}
