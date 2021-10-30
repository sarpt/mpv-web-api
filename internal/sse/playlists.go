package sse

import "github.com/sarpt/mpv-web-api/pkg/state"

const (
	playlistsSSEChannelVariant state.SSEChannelVariant = "playlists"
)

func (s *Server) createPlaylistsReplayHandler() sseReplayHandler {
	return func(res ResponseWriter) error {
		return res.SendChange(s.playlists, playlistsSSEChannelVariant, string(state.PlaylistsReplay))
	}
}

func (s *Server) createPlaylistsChangesHandler() sseChangeHandler {
	return func(res ResponseWriter, changes interface{}) error {
		change, ok := changes.(state.PlaylistsChange)
		if !ok {
			return errIncorrectChangesType
		}

		// TODO: playlists changes always send the whole state of the playlists state - hardly ideal.
		// To consider: most of the time changes would revolve around a single playlist - probably sending only playlist in question would be enough.
		return res.SendChange(s.playlists, playlistsSSEChannelVariant, string(change.Variant))
	}
}

func (s *Server) playlistsSSEChannel() channel {
	return channel{
		variant:       playlistsSSEChannelVariant,
		observers:     s.playlistsObservers,
		changeHandler: s.createPlaylistsChangesHandler(),
		replayHandler: s.createPlaylistsReplayHandler(),
	}
}

// playlistsChangesToChannelObserversDistributor is a fan-out dispatcher, which notifies all playback observers (subscribers from SSE etc.) when a playbackChange occurs.
func playlistsChangesToChannelObserversDistributor(channelObservers observers) func(change state.PlaylistsChange) {
	return func(change state.PlaylistsChange) {
		channelObservers.lock.RLock()
		for _, observer := range channelObservers.items {
			observer <- change
		}
		channelObservers.lock.RUnlock()
	}
}
