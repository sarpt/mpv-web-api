package state

import (
	"encoding/json"
)

type PlaybackSubscriber = func(change PlaybackChange)

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

	// MediaFileChange notifies about change of currently played mediaFile.
	MediaFileChange PlaybackChangeVariant = "mediaFileChange"

	// PlaybackTimeChange notifies about current timestamp change.
	PlaybackTimeChange PlaybackChangeVariant = "playbackTimeChange"

	// PlaylistSelectionChange notifies about change of currently played playlist.
	PlaylistSelectionChange PlaybackChangeVariant = "playlistSelectionChange"

	// PlaylistUnloadChange notifies about unload of playlist.
	PlaylistUnloadChange PlaybackChangeVariant = "playlistUnloadChange"

	// PlaylistCurrentIdxChange notifies about change of currently played entry in a selected playlist.
	PlaylistCurrentIdxChange PlaybackChangeVariant = "playlistCurrentIdxChange"
)

// PlaybackChange is used to inform about changes to the Playback.
// TODO: implement playback change to carry information on the change (using either interfaces or generics in go2).
type PlaybackChange struct {
	Variant PlaybackChangeVariant
	Value   interface{}
}

// Playback contains information about currently played media file.
type Playback struct {
	currentTime        float64
	currentChapterIdx  int64
	broadcaster        *ChangesBroadcaster
	fullscreen         bool
	loop               PlaybackLoop
	mediaFilePath      string
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
	MediaFilePath      string       `json:"MediaFilePath"`
	Paused             bool         `json:"Paused"`
	PlaylistCurrentIdx int          `json:"PlaylistCurrentIdx"`
	PlaylistUUID       string       `json:"PlaylistUUID"`
	SelectedAudioID    string       `json:"SelectedAudioID"`
	SelectedSubtitleID string       `json:"SelectedSubtitleID"`
}

// NewPlayback constructs Playback state.
func NewPlayback() *Playback {
	broadcaster := NewChangesBroadcaster()
	broadcaster.Broadcast()

	return &Playback{
		broadcaster:        broadcaster,
		playlistCurrentIdx: -1,
		Stopped:            true,
	}
}

// Clear clears all playback information.
func (p *Playback) Clear() {
	*p = Playback{
		broadcaster: p.broadcaster,
	}
}

// MarshalJSON satisifes json.Marshaller.
func (p *Playback) MarshalJSON() ([]byte, error) {
	pJSON := playbackJSON{
		CurrentTime:        p.currentTime,
		CurrentChapterIdx:  p.currentChapterIdx,
		Fullscreen:         p.fullscreen,
		MediaFilePath:      p.mediaFilePath,
		SelectedAudioID:    p.selectedAudioID,
		SelectedSubtitleID: p.selectedSubtitleID,
		PlaylistCurrentIdx: p.playlistCurrentIdx,
		PlaylistUUID:       p.playlistUUID,
		Paused:             p.paused,
		Loop:               p.loop,
	}
	return json.Marshal(pJSON)
}

func (p *Playback) PlaylistCurrentIdx() int {
	return p.playlistCurrentIdx
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
	p.broadcaster.changes <- PlaybackChange{
		Variant: AudioIDChange,
	}
}

// SetCurrentChapter changes currently played chapter index.
func (p *Playback) SetCurrentChapter(idx int64) {
	p.currentChapterIdx = idx
	p.broadcaster.changes <- PlaybackChange{
		Variant: CurrentChapterIdxChange,
	}
}

// SetFullscreen changes state of the fullscreen in playback.
func (p *Playback) SetFullscreen(enabled bool) {
	p.fullscreen = enabled
	p.broadcaster.changes <- PlaybackChange{
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
	p.broadcaster.changes <- PlaybackChange{
		Variant: LoopFileChange,
	}
}

// SetMediaFile changes currently played mediaFile, changing playback to not stopped.
func (p *Playback) SetMediaFile(mediaFile MediaFile) {
	p.mediaFilePath = mediaFile.path
	p.Stopped = false
	p.broadcaster.changes <- PlaybackChange{
		Variant: MediaFileChange,
	}
}

// SetPause changes whether playback should paused.
func (p *Playback) SetPause(paused bool) {
	p.paused = paused
	p.broadcaster.changes <- PlaybackChange{
		Variant: PauseChange,
	}
}

// SelectPlaylist sets currently played uuid of a playlist.
func (p *Playback) SelectPlaylist(uuid string) {
	p.broadcaster.changes <- PlaybackChange{
		Variant: PlaylistUnloadChange,
		Value:   p.playlistUUID,
	}

	p.playlistUUID = uuid

	p.broadcaster.changes <- PlaybackChange{
		Variant: PlaylistSelectionChange,
	}
}

// SelectPlaylistCurrentIdx sets currently played idx of a selected playlist.
func (p *Playback) SelectPlaylistCurrentIdx(idx int) {
	p.playlistCurrentIdx = idx

	p.broadcaster.changes <- PlaybackChange{
		Variant: PlaylistCurrentIdxChange,
	}
}

// SetPlaybackTime changes current time of a playback.
func (p *Playback) SetPlaybackTime(time float64) {
	p.currentTime = time
	p.broadcaster.changes <- PlaybackChange{
		Variant: PlaybackTimeChange,
	}
}

// SetSubtitleID changes shown subtitles id.
func (p *Playback) SetSubtitleID(sid string) {
	p.selectedSubtitleID = sid
	p.broadcaster.changes <- PlaybackChange{
		Variant: SubtitleIDChange,
	}
}

// Stop clears outdated playback information related to played mediaFile and sets playback to stopped.
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

	p.broadcaster.changes <- PlaybackChange{
		Variant: PlaybackStoppedChange,
	}

	p.Stopped = true
}

func (p *Playback) Subscribe(sub PlaybackSubscriber, onError func(err error)) {
	p.broadcaster.Subscribe(func(change interface{}) {
		playbackChange, ok := change.(PlaybackChange)
		if !ok {
			onError(errIncorrectChangesType)

			return
		}

		sub(playbackChange)
	})
}
