package api

import (
	"encoding/json"

	"github.com/sarpt/mpv-web-api/pkg/probe"
)

// Movie specifies information about a movie file that can be played
type Movie struct {
	Title           string
	FormatName      string
	FormatLongName  string
	Chapters        []probe.Chapter
	AudioStreams    []probe.AudioStream
	Duration        float64
	Path            string
	SubtitleStreams []probe.SubtitleStream
	VideoStreams    []probe.VideoStream
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

// MarshalJSON satisifes json.Marshaller
func (m *Movie) MarshalJSON() ([]byte, error) {
	mJSON := movieJSON{
		Title:           m.Title,
		FormatName:      m.FormatName,
		FormatLongName:  m.FormatLongName,
		Chapters:        m.Chapters,
		AudioStreams:    m.AudioStreams,
		Duration:        m.Duration,
		Path:            m.Path,
		SubtitleStreams: m.SubtitleStreams,
		VideoStreams:    m.VideoStreams,
	}

	return json.Marshal(mJSON)
}

func mapProbeResultToMovie(result probe.Result) Movie {
	return Movie{
		Title:           result.Format.Title,
		FormatName:      result.Format.Name,
		FormatLongName:  result.Format.LongName,
		Chapters:        result.Chapters,
		Path:            result.Path,
		VideoStreams:    result.VideoStreams,
		AudioStreams:    result.AudioStreams,
		SubtitleStreams: result.SubtitleStreams,
		Duration:        result.Format.Duration,
	}
}
