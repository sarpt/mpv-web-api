package api

import "encoding/json"

const (
	fileLoop loopVariant = "file"
	abLoop   loopVariant = "ab"
)

type loopVariant string

// PlaybackLoop contains information about playback loop
type PlaybackLoop struct {
	Variant loopVariant
	ATime   int
	BTime   int
}

type playbackLoopJSON struct {
	Variant loopVariant `json:"Variant"`
	ATime   int         `json:"ATime"`
	BTime   int         `json:"BTime"`
}

// MarshalJSON satisifes json.Marshaller
func (pl *PlaybackLoop) MarshalJSON() ([]byte, error) {
	plJSON := playbackLoopJSON{
		Variant: pl.Variant,
		ATime:   pl.ATime,
		BTime:   pl.BTime,
	}

	return json.Marshal(plJSON)
}
