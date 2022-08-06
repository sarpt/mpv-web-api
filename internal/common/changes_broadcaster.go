package common

import (
	"sync"
)

type ChangeVariant string

type Change interface {
	Variant() ChangeVariant
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
