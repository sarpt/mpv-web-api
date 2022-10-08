package media_files

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/sarpt/mpv-web-api/pkg/probe"
)

// Entry specifies information about a media file that can be played.
type Entry struct {
	audioStreams    []probe.AudioStream
	chapters        []probe.Chapter
	duration        float64
	formatName      string
	formatLongName  string
	path            string
	subtitleStreams []probe.SubtitleStream
	title           string
	uuid            string
	videoStreams    []probe.VideoStream
}

type entryJSON struct {
	AudioStreams    []probe.AudioStream    `json:"AudioStreams"`
	Chapters        []probe.Chapter        `json:"Chapters"`
	Duration        float64                `json:"Duration"`
	FormatName      string                 `json:"FormatName"`
	FormatLongName  string                 `json:"FormatLongName"`
	Path            string                 `json:"Path"`
	SubtitleStreams []probe.SubtitleStream `json:"SubtitleStreams"`
	Title           string                 `json:"Title"`
	UUID            string                 `json:"UUID"`
	VideoStreams    []probe.VideoStream    `json:"VideoStreams"`
}

// MarshalJSON satisifes json.Marshaller.
func (m Entry) MarshalJSON() ([]byte, error) {
	mJSON := entryJSON{
		Title:           m.title,
		FormatName:      m.formatName,
		FormatLongName:  m.formatLongName,
		Chapters:        m.chapters,
		AudioStreams:    m.audioStreams,
		Duration:        m.duration,
		Path:            m.path,
		SubtitleStreams: m.subtitleStreams,
		UUID:            m.uuid,
		VideoStreams:    m.videoStreams,
	}

	return json.Marshal(mJSON)
}

// Path returns mediaFile path.
func (m *Entry) Path() string {
	return m.path
}

// Uuid returns mediaFile UUID.
func (m *Entry) Uuid() string {
	return m.uuid
}

// MapProbeResultToMediaFile constructs new MediaFile from results returned by probing for media files.
func MapProbeResultToMediaFile(result probe.Result) Entry {
	uuid := uuid.NewString()

	return Entry{
		title:           result.Format.Title,
		formatName:      result.Format.Name,
		formatLongName:  result.Format.LongName,
		chapters:        result.Chapters,
		path:            result.Path,
		audioStreams:    result.AudioStreams,
		subtitleStreams: result.SubtitleStreams,
		duration:        result.Format.Duration,
		uuid:            uuid,
		videoStreams:    result.VideoStreams,
	}
}
