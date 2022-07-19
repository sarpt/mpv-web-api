package playback

import "encoding/json"

const (
	abLoop   loopVariant = "ab"
	fileLoop loopVariant = "file"
	offLoop  loopVariant = "off"
)

type loopVariant string

// Loop contains information about playback loop
type Loop struct {
	variant loopVariant
	aTime   int
	bTime   int
}

type loopJSON struct {
	Variant loopVariant `json:"Variant"`
	ATime   int         `json:"ATime"`
	BTime   int         `json:"BTime"`
}

// MarshalJSON satisifes json.Marshaller
func (pl Loop) MarshalJSON() ([]byte, error) {
	plJSON := loopJSON{
		Variant: pl.variant,
		ATime:   pl.aTime,
		BTime:   pl.bTime,
	}

	return json.Marshal(plJSON)
}
