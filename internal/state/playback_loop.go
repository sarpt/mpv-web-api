package state

import "encoding/json"

const (
	fileLoop loopVariant = "file"
	abLoop   loopVariant = "ab"
)

type loopVariant string

// PlaybackLoop contains information about playback loop
type PlaybackLoop struct {
	variant loopVariant
	aTime   int
	bTime   int
}

type playbackLoopJSON struct {
	Variant loopVariant `json:"Variant"`
	ATime   int         `json:"ATime"`
	BTime   int         `json:"BTime"`
}

// MarshalJSON satisifes json.Marshaller
func (pl *PlaybackLoop) MarshalJSON() ([]byte, error) {
	plJSON := playbackLoopJSON{
		Variant: pl.variant,
		ATime:   pl.aTime,
		BTime:   pl.bTime,
	}

	return json.Marshal(plJSON)
}
