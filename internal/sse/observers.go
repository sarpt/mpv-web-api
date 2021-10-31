package sse

import (
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/state"
)

// observers represents client observers that are currently connected to this instance of api server
// TODO: this thing should be a generic type (with type parameter instead of interface{}).
// Rewrite with go 1.18 to generics and remove the various observers structs implementing this interface.
type observers interface {
	Add(address string) <-chan interface{}
	Remove(address string)
}

type directoriesObserver struct {
	items map[string]chan interface{}
	lock  *sync.RWMutex
}

func newDirectoryObserver() *directoriesObserver {
	return &directoriesObserver{
		items: map[string]chan interface{}{},
		lock:  &sync.RWMutex{},
	}
}

func (do *directoriesObserver) Add(address string) <-chan interface{} {
	changes := make(chan interface{})

	do.lock.Lock()
	defer do.lock.Unlock()

	do.items[address] = changes

	return changes
}

func (do *directoriesObserver) Remove(address string) {
	do.lock.Lock()
	defer do.lock.Unlock()

	delete(do.items, address)
}

func (do *directoriesObserver) BroadcastToChannelObservers(change state.DirectoriesChange) {
	do.lock.RLock()
	defer do.lock.RUnlock()

	for _, observer := range do.items {
		observer <- change
	}
}

type playbackObservers struct {
	items map[string]chan interface{}
	lock  *sync.RWMutex
}

func newPlaybackObservers() *playbackObservers {
	return &playbackObservers{
		items: map[string]chan interface{}{},
		lock:  &sync.RWMutex{},
	}
}

func (do *playbackObservers) Add(address string) <-chan interface{} {
	changes := make(chan interface{})

	do.lock.Lock()
	defer do.lock.Unlock()

	do.items[address] = changes

	return changes
}

func (do *playbackObservers) Remove(address string) {
	do.lock.Lock()
	defer do.lock.Unlock()

	delete(do.items, address)
}

func (do *playbackObservers) BroadcastToChannelObservers(change state.PlaybackChange) {
	do.lock.RLock()
	defer do.lock.RUnlock()

	for _, observer := range do.items {
		observer <- change
	}
}

type playlistsObservers struct {
	items map[string]chan interface{}
	lock  *sync.RWMutex
}

func newPlaylistsObservers() *playlistsObservers {
	return &playlistsObservers{
		items: map[string]chan interface{}{},
		lock:  &sync.RWMutex{},
	}
}

func (po *playlistsObservers) Add(address string) <-chan interface{} {
	changes := make(chan interface{})

	po.lock.Lock()
	defer po.lock.Unlock()

	po.items[address] = changes

	return changes
}

func (po *playlistsObservers) Remove(address string) {
	po.lock.Lock()
	defer po.lock.Unlock()

	delete(po.items, address)
}

func (po *playlistsObservers) BroadcastToChannelObservers(change state.PlaylistsChange) {
	po.lock.RLock()
	defer po.lock.RUnlock()

	for _, observer := range po.items {
		observer <- change
	}
}

type mediaFilesObservers struct {
	items map[string]chan interface{}
	lock  *sync.RWMutex
}

func newMediaFilesObservers() *mediaFilesObservers {
	return &mediaFilesObservers{
		items: map[string]chan interface{}{},
		lock:  &sync.RWMutex{},
	}
}

func (mfo *mediaFilesObservers) Add(address string) <-chan interface{} {
	changes := make(chan interface{})

	mfo.lock.Lock()
	defer mfo.lock.Unlock()

	mfo.items[address] = changes

	return changes
}

func (mfo *mediaFilesObservers) Remove(address string) {
	mfo.lock.Lock()
	defer mfo.lock.Unlock()

	delete(mfo.items, address)
}

func (mfo *mediaFilesObservers) BroadcastToChannelObservers(change state.MediaFilesChange) {
	mfo.lock.RLock()
	defer mfo.lock.RUnlock()

	for _, observer := range mfo.items {
		observer <- change
	}
}

type statusObservers struct {
	items map[string]chan interface{}
	lock  *sync.RWMutex
}

func newStatusObservers() *statusObservers {
	return &statusObservers{
		items: map[string]chan interface{}{},
		lock:  &sync.RWMutex{},
	}
}

func (so *statusObservers) Add(address string) <-chan interface{} {
	changes := make(chan interface{})

	so.lock.Lock()
	defer so.lock.Unlock()

	so.items[address] = changes

	return changes
}

func (so *statusObservers) Remove(address string) {
	so.lock.Lock()
	defer so.lock.Unlock()

	delete(so.items, address)
}

func (so *statusObservers) BroadcastToChannelObservers(change state.StatusChange) {
	so.lock.RLock()
	defer so.lock.RUnlock()

	for _, observer := range so.items {
		observer <- change
	}
}
