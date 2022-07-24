package sse

import (
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playlists"
	state_sse "github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
)

const (
	playlistsSSEChannelVariant state_sse.ChannelVariant = "playlists"

	playlistsReplay state_sse.ChangeVariant = "replay"
)

type playlistsChangesBroadcaster struct {
	playback  *playback.Storage
	playlists *playlists.Storage
	ChangesBroadcaster[playlists.Change]
}

func (pc *playlistsChangesBroadcaster) Replay(res ResponseWriter) error {
	return res.SendChange(pc.playlists, playlistsSSEChannelVariant, string(playlistsReplay))
}

func (pc *playlistsChangesBroadcaster) ChangeHandler(res ResponseWriter, change playlists.Change) error {
	if pc.playback.Stopped { // TODO: the changes are shot by state.Playback even after the mediaFilePath is cleared, as such it may be wasteful to push further changes through SSE. to think of a way to reduce number of those blank data calls after closing stopping playback
		return res.SendEmptyChange(playlistsSSEChannelVariant, string(change.ChangeVariant))
	}

	return res.SendChange(change.Playlist, playlistsSSEChannelVariant, string(change.ChangeVariant))
}

func NewPlaylistsChannel(playbackStorage *playback.Storage, playlistsStorage *playlists.Storage) *StateChannel[playlists.Change] {
	return &StateChannel[playlists.Change]{
		&playlistsChangesBroadcaster{
			playbackStorage,
			playlistsStorage,
			NewChangesBroadcaster[playlists.Change](),
		},
		playlistsSSEChannelVariant,
	}
}
