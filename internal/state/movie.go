package state

import (
	"encoding/json"

	"github.com/sarpt/mpv-web-api/pkg/probe"
)

// Movie specifies information about a movie file that can be played.
// TODO: Add id to movie - currently "path" is assumed to be unique,
// but in case in future mutliple mpv-web-api servers are serving from different
// machines, the path may not be unique from the api user pov
// (either randomly generate one or sth else)
type Movie struct {
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

type movieJSON struct {
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
func (m Movie) MarshalJSON() ([]byte, error) {
	mJSON := movieJSON{
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

// Path returns movie path.
func (m *Movie) Path() string {
	return m.path
}

// MapProbeResultToMovie constructs new Movie from results returned by probing for movie files.
func MapProbeResultToMovie(result probe.Result) Movie {
	return Movie{
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
