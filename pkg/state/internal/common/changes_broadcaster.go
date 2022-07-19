package common

import (
	"errors"
	"sync"
)

var (
	ErrIncorrectChangesType = errors.New("changes of incorrect type provided to the change handler")
)

type ChangesSubscriber = func(change interface{})

type ChangesBroadcaster struct {
	changes     chan interface{}
	lock        *sync.RWMutex
	subscribers []ChangesSubscriber
}

func NewChangesBroadcaster() *ChangesBroadcaster {
	return &ChangesBroadcaster{
		changes:     make(chan interface{}),
		lock:        &sync.RWMutex{},
		subscribers: []ChangesSubscriber{},
	}
}

func (cb *ChangesBroadcaster) Subscribe(sub ChangesSubscriber) {
	cb.lock.Lock()
	defer cb.lock.Unlock()

	cb.subscribers = append(cb.subscribers, sub)
}

func (cb *ChangesBroadcaster) Send(payload any) {
	cb.changes <- payload
}

func (cb *ChangesBroadcaster) Broadcast() {
	go func() {
		for {
			change, more := <-cb.changes
			if !more {
				return
			}

			cb.lock.RLock()
			for _, sub := range cb.subscribers {
				sub(change)
			}
			cb.lock.RUnlock()
		}
	}()
}
