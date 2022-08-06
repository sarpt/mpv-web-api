package common

import (
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
)

type Change interface {
	Variant() sse.ChangeVariant
}

// type ChangesSubscriber = func(change interface{})
type ChangesSubscriber[CT Change] interface {
	Receive(change CT)
}

type ChangesBroadcaster[CT Change] struct {
	Broadcaster[CT]
}

func NewChangesBroadcaster[CT Change]() *ChangesBroadcaster[CT] {
	return &ChangesBroadcaster[CT]{
		Broadcaster[CT]{
			changes:     make(chan CT),
			lock:        &sync.RWMutex{},
			subscribers: []Subscriber[CT]{},
		},
	}
}
