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

type playlistsChannel struct {
	playback *playback.Storage
	StateChannel[*playlists.Storage, playlists.Change]
}

func newPlaylistsChannel(playbackStorage *playback.Storage, playlistsStorage *playlists.Storage) *playlistsChannel {
	return &playlistsChannel{
		playbackStorage,
		NewStateChannel[*playlists.Storage, playlists.Change](playlistsStorage, playlistsSSEChannelVariant),
	}
}

func (pc *playlistsChannel) Replay(res ResponseWriter) error {
	return res.SendChange(pc.state, pc.Variant(), string(playlistsReplay))
}

func (pc *playlistsChannel) changeHandler(res ResponseWriter, change playlists.Change) error {
	if pc.playback.Stopped { // TODO: the changes are shot by state.Playback even after the mediaFilePath is cleared, as such it may be wasteful to push further changes through SSE. to think of a way to reduce number of those blank data calls after closing stopping playback
		return res.SendEmptyChange(pc.Variant(), string(change.ChangeVariant))
	}

	return res.SendChange(change.Playlist, pc.Variant(), string(change.ChangeVariant))
}
