package media_files

import (
	"encoding/json"

	"github.com/sarpt/mpv-web-api/pkg/probe"
)

// MediaFile specifies information about a media file that can be played.
// TODO: Add id to mediaFile - currently "path" is assumed to be unique,
// but in case in future mutliple mpv-web-api servers are serving from different
// machines, the path may not be unique from the api user pov
// (either randomly generate one or sth else)
type MediaFile struct {
	title           string
	formatName      string
	formatLongName  string
	chapters        []probe.Chapter
	audioStreams    []probe.AudioStream
	duration        float64
	path            string
	subtitleStreams []probe.SubtitleStream
	videoStreams    []probe.VideoStream
}

type mediaFileJSON struct {
	Title           string                 `json:"Title"`
	FormatName      string                 `json:"FormatName"`
	FormatLongName  string                 `json:"FormatLongName"`
	Chapters        []probe.Chapter        `json:"Chapters"`
	AudioStreams    []probe.AudioStream    `json:"AudioStreams"`
	Duration        float64                `json:"Duration"`
	Path            string                 `json:"Path"`
	SubtitleStreams []probe.SubtitleStream `json:"SubtitleStreams"`
	VideoStreams    []probe.VideoStream    `json:"VideoStreams"`
}

// MarshalJSON satisifes json.Marshaller.
func (m MediaFile) MarshalJSON() ([]byte, error) {
	mJSON := mediaFileJSON{
		Title:           m.title,
		FormatName:      m.formatName,
		FormatLongName:  m.formatLongName,
		Chapters:        m.chapters,
		AudioStreams:    m.audioStreams,
		Duration:        m.duration,
		Path:            m.path,
		SubtitleStreams: m.subtitleStreams,
		VideoStreams:    m.videoStreams,
	}

	return json.Marshal(mJSON)
}

// Path returns mediaFile path.
func (m *MediaFile) Path() string {
	return m.path
}

// MapProbeResultToMediaFile constructs new MediaFile from results returned by probing for media files.
func MapProbeResultToMediaFile(result probe.Result) MediaFile {
	return MediaFile{
		title:           result.Format.Title,
		formatName:      result.Format.Name,
		formatLongName:  result.Format.LongName,
		chapters:        result.Chapters,
		path:            result.Path,
		videoStreams:    result.VideoStreams,
		audioStreams:    result.AudioStreams,
		subtitleStreams: result.SubtitleStreams,
		duration:        result.Format.Duration,
	}
}
