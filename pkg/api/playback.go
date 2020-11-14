package api

import (
	"encoding/json"
)

// Playback contains information about currently played movie file
type Playback struct {
	CurrentTime        float64
	CurrentChapterIdx  int
	Fullscreen         bool
	Movie              Movie
	SelectedAudioID    string
	SelectedSubtitleID string
	Paused             bool
	Loop               PlaybackLoop
	Changes            chan interface{}
}

type playbackJSON struct {
	CurrentTime        float64      `json:"CurrentTime"`
	CurrentChapterIdx  int          `json:"CurrentChapterIdx"`
	Fullscreen         bool         `json:"Fullscreen"`
	Movie              Movie        `json:"Movie"`
	SelectedAudioID    string       `json:"SelectedAudioID"`
	SelectedSubtitleID string       `json:"SelectedSubtitleID"`
	Paused             bool         `json:"Paused"`
	Loop               PlaybackLoop `json:"Loop"`
}

// MarshalJSON satisifes json.Marshaller
func (p *Playback) MarshalJSON() ([]byte, error) {
	pJSON := playbackJSON{
		CurrentTime:        p.CurrentTime,
		CurrentChapterIdx:  p.CurrentChapterIdx,
		Fullscreen:         p.Fullscreen,
		Movie:              p.Movie,
		SelectedAudioID:    p.SelectedAudioID,
		SelectedSubtitleID: p.SelectedSubtitleID,
		Paused:             p.Paused,
		Loop:               p.Loop,
	}
	return json.Marshal(pJSON)
}
