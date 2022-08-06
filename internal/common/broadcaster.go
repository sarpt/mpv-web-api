package common

import (
	"sync"
)

type Subscriber[CT any] interface {
	Receive(change CT)
}

type Broadcaster[CT any] struct {
	changes     chan CT
	lock        *sync.RWMutex
	subscribers []Subscriber[CT]
}

func NewBroadcaster[CT any]() *Broadcaster[CT] {
	return &Broadcaster[CT]{
		changes:     make(chan CT),
		lock:        &sync.RWMutex{},
		subscribers: []Subscriber[CT]{},
	}
}

func (cb *Broadcaster[CT]) Subscribe(sub Subscriber[CT]) {
	cb.lock.Lock()
	defer cb.lock.Unlock()

	cb.subscribers = append(cb.subscribers, sub)
}

func (cb *Broadcaster[CT]) Send(payload CT) {
	cb.changes <- payload
}

func (cb *Broadcaster[CT]) Broadcast() {
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
