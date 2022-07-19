package sse

import (
	state_sse "github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
)

type channel interface {
	AddObserver(address string)
	RemoveObserver(address string)
	Replay(res ResponseWriter) error
	ServeObserver(address string, res ResponseWriter, done chan<- bool, errors chan<- error)
	Variant() state_sse.ChannelVariant
}
