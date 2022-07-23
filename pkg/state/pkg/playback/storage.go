package playback

import (
	"encoding/json"

	"github.com/sarpt/mpv-web-api/pkg/state/internal/common"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/media_files"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
)

type Subscriber = func(change Change)

const (
	// FullscreenChange notifies about fullscreen state change.
	FullscreenChange sse.ChangeVariant = "fullscreenChange"

	// LoopFileChange notifies about change to the looping of current file.
	LoopFileChange sse.ChangeVariant = "loopFileChange"

	// PauseChange notifies about change to the playback pause state.
	PauseChange sse.ChangeVariant = "pauseChange"

	// AudioIDChange notifies about change of currently played audio.
	AudioIDChange sse.ChangeVariant = "audioIdChange"

	// PlaybackStoppedChange notifies about playbck being stopped completely.
	PlaybackStoppedChange sse.ChangeVariant = "playbackStoppedChange"

	// SubtitleIDChange notifies about change of currently shown subtitles.
	SubtitleIDChange sse.ChangeVariant = "subtitleIdChange"

	// CurrentChapterIdxChange notifies about change of currently played chapter.
	CurrentChapterIdxChange sse.ChangeVariant = "currentChapterIndexChange"

	// MediaFileChange notifies about change of currently played mediaFile.
	MediaFileChange sse.ChangeVariant = "mediaFileChange"

	// PlaybackTimeChange notifies about current timestamp change.
	PlaybackTimeChange sse.ChangeVariant = "playbackTimeChange"

	// PlaylistSelectionChange notifies about change of currently played playlist.
	PlaylistSelectionChange sse.ChangeVariant = "playlistSelectionChange"

	// PlaylistUnloadChange notifies about unload of playlist.
	PlaylistUnloadChange sse.ChangeVariant = "playlistUnloadChange"

	// PlaylistCurrentIdxChange notifies about change of currently played entry in a selected playlist.
	PlaylistCurrentIdxChange sse.ChangeVariant = "playlistCurrentIdxChange"
)

// Change is used to inform about changes to the Playback.
// TODO: implement playback change to carry information on the change (using either interfaces or generics in go2).
type Change struct {
	ChangeVariant sse.ChangeVariant
	Value         interface{}
}

// MarshalJSON returns change items in JSON format. Satisfies json.Marshaller.
func (d Change) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Value)
}

func (d Change) Variant() sse.ChangeVariant {
	return d.ChangeVariant
}

// Storage contains information about currently played media file.
type Storage struct {
	currentTime        float64
	currentChapterIdx  int64
	broadcaster        *common.ChangesBroadcaster
	fullscreen         bool
	loop               Loop
	mediaFilePath      string
	paused             bool
	playlistCurrentIdx int
	playlistUUID       string
	selectedAudioID    string
	selectedSubtitleID string
	Stopped            bool
}

type storageJSON struct {
	CurrentTime        float64 `json:"CurrentTime"`
	CurrentChapterIdx  int64   `json:"CurrentChapterIdx"`
	Fullscreen         bool    `json:"Fullscreen"`
	Loop               Loop    `json:"Loop"`
	MediaFilePath      string  `json:"MediaFilePath"`
	Paused             bool    `json:"Paused"`
	PlaylistCurrentIdx int     `json:"PlaylistCurrentIdx"`
	PlaylistUUID       string  `json:"PlaylistUUID"`
	SelectedAudioID    string  `json:"SelectedAudioID"`
	SelectedSubtitleID string  `json:"SelectedSubtitleID"`
}

// NewStorage constructs Playback state.
func NewStorage() *Storage {
	broadcaster := common.NewChangesBroadcaster()
	broadcaster.Broadcast()

	return &Storage{
		broadcaster:        broadcaster,
		playlistCurrentIdx: -1,
		Stopped:            true,
	}
}

// Clear clears all playback information.
func (p *Storage) Clear() {
	*p = Storage{
		broadcaster: p.broadcaster,
	}
}

// MarshalJSON satisifes json.Marshaller.
func (p *Storage) MarshalJSON() ([]byte, error) {
	pJSON := storageJSON{
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

func (p *Storage) PlaylistCurrentIdx() int {
	return p.playlistCurrentIdx
}

func (p *Storage) PlaylistUUID() string {
	return p.playlistUUID
}

func (p *Storage) PlaylistSelected() bool {
	return p.PlaylistUUID() != ""
}

// SetAudioID changes played audio id.
func (p *Storage) SetAudioID(aid string) {
	p.selectedAudioID = aid
	p.broadcaster.Send(Change{
		ChangeVariant: AudioIDChange,
	})
}

// SetCurrentChapter changes currently played chapter index.
func (p *Storage) SetCurrentChapter(idx int64) {
	p.currentChapterIdx = idx
	p.broadcaster.Send(Change{
		ChangeVariant: CurrentChapterIdxChange,
	})
}

// SetFullscreen changes state of the fullscreen in playback.
func (p *Storage) SetFullscreen(enabled bool) {
	p.fullscreen = enabled
	p.broadcaster.Send(Change{
		ChangeVariant: FullscreenChange,
	})
}

// SetLoopFile changes whether file should be looped.
func (p *Storage) SetLoopFile(enabled bool) {
	if enabled {
		p.loop.variant = fileLoop
	} else {
		p.loop.variant = offLoop
	}
	p.broadcaster.Send(Change{
		ChangeVariant: LoopFileChange,
	})
}

// SetMediaFile changes currently played mediaFile, changing playback to not stopped.
func (p *Storage) SetMediaFile(mediaFile media_files.Entry) {
	p.mediaFilePath = mediaFile.Path()
	p.Stopped = false
	p.broadcaster.Send(Change{
		ChangeVariant: MediaFileChange,
	})
}

// SetPause changes whether playback should paused.
func (p *Storage) SetPause(paused bool) {
	p.paused = paused
	p.broadcaster.Send(Change{
		ChangeVariant: PauseChange,
	})
}

// SelectPlaylist sets currently played uuid of a playlist.
func (p *Storage) SelectPlaylist(uuid string) {
	p.broadcaster.Send(Change{
		ChangeVariant: PlaylistUnloadChange,
		Value:         p.playlistUUID,
	})

	p.playlistUUID = uuid

	p.broadcaster.Send(Change{
		ChangeVariant: PlaylistSelectionChange,
	})
}

// SelectPlaylistCurrentIdx sets currently played idx of a selected playlist.
func (p *Storage) SelectPlaylistCurrentIdx(idx int) {
	p.playlistCurrentIdx = idx

	p.broadcaster.Send(Change{
		ChangeVariant: PlaylistCurrentIdxChange,
	})
}

// SetPlaybackTime changes current time of a playback.
func (p *Storage) SetPlaybackTime(time float64) {
	p.currentTime = time
	p.broadcaster.Send(Change{
		ChangeVariant: PlaybackTimeChange,
	})
}

// SetSubtitleID changes shown subtitles id.
func (p *Storage) SetSubtitleID(sid string) {
	p.selectedSubtitleID = sid
	p.broadcaster.Send(Change{
		ChangeVariant: SubtitleIDChange,
	})
}

// Stop clears outdated playback information related to played mediaFile and sets playback to stopped.
// The method preservers information about played playlist, since the playlist might not have been saved for a default (unnamed) playlist.
// Change is being propagated before setting the state of Stopped, to inform observers about clear state of the playback,
// and before suppressing further changes playback changes to stopped playback.
// TODO: to consider not clearing the outdated information, since it will be updated after new media playback change,
// as such the clearing of playback method seems redundant, and the result potentialy unwanted
// (the payload will not be sent when Stopped is true, so the outdated information will not be sent on changes chan).
func (p *Storage) Stop() {
	playlistUUID := p.playlistUUID

	p.Clear()
	p.playlistCurrentIdx = -1
	p.playlistUUID = playlistUUID

	p.broadcaster.Send(Change{
		ChangeVariant: PlaybackStoppedChange,
	})

	p.Stopped = true
}

func (p *Storage) Subscribe(sub Subscriber, onError func(err error)) {
	p.broadcaster.Subscribe(func(change interface{}) {
		playbackChange, ok := change.(Change)
		if !ok {
			onError(common.ErrIncorrectChangesType)

			return
		}

		sub(playbackChange)
	})
}
