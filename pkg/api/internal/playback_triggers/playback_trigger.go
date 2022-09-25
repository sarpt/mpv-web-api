package playback_triggers

import "github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"

type PlaybackTrigger interface {
	Handler(change playback.Change) error
}
