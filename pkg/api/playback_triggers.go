package api

import (
	playbackTriggers "github.com/sarpt/mpv-web-api/pkg/api/internal/playback_triggers"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"
)

func (s *Server) addPlaybackTrigger(trigger playbackTriggers.PlaybackTrigger) func() {
	return s.statesRepository.Playback().Subscribe(func(change playback.Change) {
		err := trigger.Handler(change)
		if err != nil {
			s.errLog.Printf("playback trigger for media file returned error: %s", err)
		}
	}, func(err error) {})
}
