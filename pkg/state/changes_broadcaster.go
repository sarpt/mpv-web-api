package state

import "sync"

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

func (cb *ChangesBroadcaster) Broadcast() {
	go func() {
		for {
			change, done := <-cb.changes
			if done {
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
