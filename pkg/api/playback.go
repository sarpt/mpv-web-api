package api

import (
	"encoding/json"
)

// PlaybackChangeVariant specifies type of change that happened to playback
type PlaybackChangeVariant string

const (
	// FullscreenChange notifies about fullscreen state change
	FullscreenChange PlaybackChangeVariant = "fullscreenChange"

	// LoopFileChange notifies about change to the looping of current file
	LoopFileChange PlaybackChangeVariant = "loopFileChange"

	// PauseChange notifies about change to the playback pause state
	PauseChange PlaybackChangeVariant = "pauseChange"

	// AudioIDChange notifies about change of currently played audio
	AudioIDChange PlaybackChangeVariant = "audioIdChange"

	// SubtitleIDChange notifies about change of currently shown subtitles
	SubtitleIDChange PlaybackChangeVariant = "subtitleIdChange"

	// CurrentChapterIdxChange notifies about change of currently played chapter
	CurrentChapterIdxChange PlaybackChangeVariant = "currentChapterIndexChange"

	// MovieChange notifies about change of currently played movie (TODO: change to "fileChange" in preparation for Music playback?)
	MovieChange PlaybackChangeVariant = "movieChange"

	// PlaybackTimeChange notifies about current timestamp change
	PlaybackTimeChange PlaybackChangeVariant = "playbackTimeChange"
)

// PlaybackChange is used to inform about changes to the Playback.
// TODO: implement playback change to carry information on the change (using either interfaces or generics in go2)
type PlaybackChange struct {
	Variant PlaybackChangeVariant
}

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
	changes            chan interface{}
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

// Changes returns read-only channel notifying of playback changes
func (p *Playback) Changes() <-chan interface{} {
	return p.changes
}

// SetFullscreen changes state of the fullscreen in playback
func (p *Playback) SetFullscreen(enabled bool) {
	p.Fullscreen = enabled
	p.changes <- PlaybackChange{
		Variant: FullscreenChange,
	}
}

// SetLoopFile changes whether file should be looped
func (p *Playback) SetLoopFile(enabled bool) {
	if enabled {
		p.Loop.Variant = fileLoop
	} else {
		p.Loop.Variant = ""
	}
	p.changes <- PlaybackChange{
		Variant: LoopFileChange,
	}
}

// SetPause changes whether playback should paused
func (p *Playback) SetPause(paused bool) {
	p.Paused = paused
	p.changes <- PlaybackChange{
		Variant: PauseChange,
	}
}

// SetAudioID changes played audio id
func (p *Playback) SetAudioID(aid string) {
	p.SelectedAudioID = aid
	p.changes <- PlaybackChange{
		Variant: AudioIDChange,
	}
}

// SetSubtitleID changes shown subtitles id
func (p *Playback) SetSubtitleID(sid string) {
	p.SelectedSubtitleID = sid
	p.changes <- PlaybackChange{
		Variant: SubtitleIDChange,
	}
}

// SetCurrentChapter changes currently played chapter index
func (p *Playback) SetCurrentChapter(idx int) {
	p.CurrentChapterIdx = idx
	p.changes <- PlaybackChange{
		Variant: CurrentChapterIdxChange,
	}
}

// SetMovie changes currently played movie
func (p *Playback) SetMovie(movie Movie) {
	p.Movie = movie
	p.changes <- PlaybackChange{
		Variant: MovieChange,
	}
}

// SetPlaybackTime changes current time of a playback
func (p *Playback) SetPlaybackTime(time float64) {
	p.CurrentTime = time
	p.changes <- PlaybackChange{
		Variant: PlaybackTimeChange,
	}
}
