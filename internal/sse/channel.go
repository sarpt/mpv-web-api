package sse

import (
	"github.com/sarpt/mpv-web-api/pkg/state"
)

type channel interface {
	AddObserver(address string)
	RemoveObserver(address string)
	Replay(res ResponseWriter) error
	ServeObserver(address string, res ResponseWriter, done chan<- bool, errors chan<- error)
	Variant() state.SSEChannelVariant
}
