package state

import (
	"encoding/json"
)

// PlaybackChangeVariant specifies type of change that happened to playback.
type PlaybackChangeVariant string

const (
	// FullscreenChange notifies about fullscreen state change.
	FullscreenChange PlaybackChangeVariant = "fullscreenChange"

	// LoopFileChange notifies about change to the looping of current file.
	LoopFileChange PlaybackChangeVariant = "loopFileChange"

	// PauseChange notifies about change to the playback pause state.
	PauseChange PlaybackChangeVariant = "pauseChange"

	// AudioIDChange notifies about change of currently played audio.
	AudioIDChange PlaybackChangeVariant = "audioIdChange"

	// PlaybackStoppedChange notifies about playbck being stopped completely.
	PlaybackStoppedChange PlaybackChangeVariant = "playbackStoppedChange"

	// SubtitleIDChange notifies about change of currently shown subtitles.
	SubtitleIDChange PlaybackChangeVariant = "subtitleIdChange"

	// CurrentChapterIdxChange notifies about change of currently played chapter.
	CurrentChapterIdxChange PlaybackChangeVariant = "currentChapterIndexChange"

	// MovieChange notifies about change of currently played movie (TODO: change to "fileChange" in preparation for Music playback?).
	MovieChange PlaybackChangeVariant = "movieChange"

	// PlaybackTimeChange notifies about current timestamp change.
	PlaybackTimeChange PlaybackChangeVariant = "playbackTimeChange"

	// PlaylistSelectionChange notifies about change of currently played playlist.
	PlaylistSelectionChange PlaybackChangeVariant = "playlistSelectionChange"

	// PlaylistCurrentIdxChange notifies about change of currently played entry in a selected playlist.
	PlaylistCurrentIdxChange PlaybackChangeVariant = "playlistCurrentIdxChange"
)

// PlaybackChange is used to inform about changes to the Playback.
// TODO: implement playback change to carry information on the change (using either interfaces or generics in go2).
type PlaybackChange struct {
	Variant PlaybackChangeVariant
}

// Playback contains information about currently played movie file.
type Playback struct {
	changes            chan interface{}
	currentTime        float64
	currentChapterIdx  int64
	fullscreen         bool
	loop               PlaybackLoop
	moviePath          string
	paused             bool
	playlistCurrentIdx int
	playlistUUID       string
	selectedAudioID    string
	selectedSubtitleID string
	Stopped            bool
}

type playbackJSON struct {
	CurrentTime        float64      `json:"CurrentTime"`
	CurrentChapterIdx  int64        `json:"CurrentChapterIdx"`
	Fullscreen         bool         `json:"Fullscreen"`
	Loop               PlaybackLoop `json:"Loop"`
	MoviePath          string       `json:"MoviePath"`
	Paused             bool         `json:"Paused"`
	PlaylistCurrentIdx int          `json:"PlaylistCurrentIdx"`
	PlaylistUUID       string       `json:"PlaylistUUID"`
	SelectedAudioID    string       `json:"SelectedAudioID"`
	SelectedSubtitleID string       `json:"SelectedSubtitleID"`
}

// NewPlayback constructs Playback state.
func NewPlayback() *Playback {
	return &Playback{
		playlistCurrentIdx: -1,
		Stopped:            true,
		changes:            make(chan interface{}),
	}
}

// Changes returns read-only channel notifying of playback changes.
func (p *Playback) Changes() <-chan interface{} {
	return p.changes
}

// Clear clears all playback information.
func (p *Playback) Clear() {
	*p = Playback{
		changes: p.changes,
	}
}

// MarshalJSON satisifes json.Marshaller.
func (p *Playback) MarshalJSON() ([]byte, error) {
	pJSON := playbackJSON{
		CurrentTime:        p.currentTime,
		CurrentChapterIdx:  p.currentChapterIdx,
		Fullscreen:         p.fullscreen,
		MoviePath:          p.moviePath,
		SelectedAudioID:    p.selectedAudioID,
		SelectedSubtitleID: p.selectedSubtitleID,
		PlaylistCurrentIdx: p.playlistCurrentIdx,
		PlaylistUUID:       p.playlistUUID,
		Paused:             p.paused,
		Loop:               p.loop,
	}
	return json.Marshal(pJSON)
}

func (p *Playback) PlaylistUUID() string {
	return p.playlistUUID
}

func (p *Playback) PlaylistSelected() bool {
	return p.PlaylistUUID() != ""
}

// SetAudioID changes played audio id.
func (p *Playback) SetAudioID(aid string) {
	p.selectedAudioID = aid
	p.changes <- PlaybackChange{
		Variant: AudioIDChange,
	}
}

// SetCurrentChapter changes currently played chapter index.
func (p *Playback) SetCurrentChapter(idx int64) {
	p.currentChapterIdx = idx
	p.changes <- PlaybackChange{
		Variant: CurrentChapterIdxChange,
	}
}

// SetFullscreen changes state of the fullscreen in playback.
func (p *Playback) SetFullscreen(enabled bool) {
	p.fullscreen = enabled
	p.changes <- PlaybackChange{
		Variant: FullscreenChange,
	}
}

// SetLoopFile changes whether file should be looped.
func (p *Playback) SetLoopFile(enabled bool) {
	if enabled {
		p.loop.variant = fileLoop
	} else {
		p.loop.variant = offLoop
	}
	p.changes <- PlaybackChange{
		Variant: LoopFileChange,
	}
}

// SetMovie changes currently played movie, changing playback to not stopped.
func (p *Playback) SetMovie(movie Movie) {
	p.moviePath = movie.path
	p.Stopped = false
	p.changes <- PlaybackChange{
		Variant: MovieChange,
	}
}

// SetPause changes whether playback should paused.
func (p *Playback) SetPause(paused bool) {
	p.paused = paused
	p.changes <- PlaybackChange{
		Variant: PauseChange,
	}
}

// SelectPlaylist sets currently played uuid of a playlist.
func (p *Playback) SelectPlaylist(uuid string) {
	p.playlistUUID = uuid

	p.changes <- PlaybackChange{
		Variant: PlaylistSelectionChange,
	}
}

// SelectPlaylistCurrentIdx sets currently played idx of a selected playlist.
func (p *Playback) SelectPlaylistCurrentIdx(idx int) {
	p.playlistCurrentIdx = idx

	p.changes <- PlaybackChange{
		Variant: PlaylistCurrentIdxChange,
	}
}

// SetPlaybackTime changes current time of a playback.
func (p *Playback) SetPlaybackTime(time float64) {
	p.currentTime = time
	p.changes <- PlaybackChange{
		Variant: PlaybackTimeChange,
	}
}

// SetSubtitleID changes shown subtitles id.
func (p *Playback) SetSubtitleID(sid string) {
	p.selectedSubtitleID = sid
	p.changes <- PlaybackChange{
		Variant: SubtitleIDChange,
	}
}

// Stop clears outdated playback information related to played movie and sets playback to stopped.
// The method preservers information about played playlist, since the playlist might not have been saved for a default (unnamed) playlist.
// Change is being propagated before setting the state of Stopped, to inform observers about clear state of the playback,
// and before suppressing further changes playback changes to stopped playback.
// TODO: to consider not clearing the outdated information, since it will be updated after new media playback change,
// as such the clearing of playback method seems redundant, and the result potentialy unwanted
// (the payload will not be sent when Stopped is true, so the outdated information will not be sent on changes chan).
func (p *Playback) Stop() {
	playlistUUID := p.playlistUUID

	p.Clear()
	p.playlistCurrentIdx = -1
	p.playlistUUID = playlistUUID

	p.changes <- PlaybackChange{
		Variant: PlaybackStoppedChange,
	}

	p.Stopped = true
}
