package common

import (
	"errors"
	"sync"
)

var (
	ErrIncorrectChangesType = errors.New("changes of incorrect type provided to the change handler")
)

// type ChangesSubscriber = func(change interface{})
type ChangesSubscriber[CT any] interface {
	Receive(change CT)
}

type ChangesBroadcaster[CT any] struct {
	changes     chan CT
	lock        *sync.RWMutex
	subscribers []ChangesSubscriber[CT]
}

func NewChangesBroadcaster[CT any]() *ChangesBroadcaster[CT] {
	return &ChangesBroadcaster[CT]{
		changes:     make(chan CT),
		lock:        &sync.RWMutex{},
		subscribers: []ChangesSubscriber[CT]{},
	}
}

func (cb *ChangesBroadcaster[CT]) Subscribe(sub ChangesSubscriber[CT]) {
	cb.lock.Lock()
	defer cb.lock.Unlock()

	cb.subscribers = append(cb.subscribers, sub)
}

func (cb *ChangesBroadcaster[CT]) Send(payload CT) {
	cb.changes <- payload
}

func (cb *ChangesBroadcaster[CT]) Broadcast() {
	go func() {
		for {
			change, more := <-cb.changes
			if !more {
				return
			}

			cb.lock.RLock()
			for _, subscriber := range cb.subscribers {
				subscriber.Receive(change)
			}
			cb.lock.RUnlock()
		}
	}()
}
