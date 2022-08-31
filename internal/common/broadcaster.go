package common

import (
	"sync"

	"github.com/google/uuid"
)

type Subscriber[T any] interface {
	Receive(payload T, unsub func())
}

type Broadcaster[T any] struct {
	payloads    chan T
	lock        *sync.RWMutex
	subscribers map[string]Subscriber[T]
}

func NewBroadcaster[T any]() *Broadcaster[T] {
	return &Broadcaster[T]{
		payloads:    make(chan T),
		lock:        &sync.RWMutex{},
		subscribers: map[string]Subscriber[T]{},
	}
}

// Subscribe adds subscriber to the broadcast listeneres pool.
// Returns unsubscriber function.
func (b *Broadcaster[T]) Subscribe(sub Subscriber[T]) func() {
	b.lock.Lock()
	defer b.lock.Unlock()

	subUuid := uuid.NewString()
	b.subscribers[subUuid] = sub

	return func() { b.unsubscribe(subUuid) }
}

func (b *Broadcaster[T]) unsubscribe(subUuid string) {
	b.lock.Lock()
	defer b.lock.Unlock()

	delete(b.subscribers, subUuid)
}

func (b *Broadcaster[T]) Send(payload T) {
	b.payloads <- payload
}

func (b *Broadcaster[T]) Broadcast() {
	go func() {
		for {
			payload, more := <-b.payloads
			if !more {
				return
			}

			b.lock.RLock()
			for subUuid, subscriber := range b.subscribers {
				go subscriber.Receive(payload, func() { b.unsubscribe(subUuid) })
			}
			b.lock.RUnlock()
		}
	}()
}
