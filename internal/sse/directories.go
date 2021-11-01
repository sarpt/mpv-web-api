package sse

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/state"
)

const (
	directoriesSSEChannelVariant state.SSEChannelVariant = "directories"
)

type directoriesMapChange struct {
	Directories map[string]state.Directory
}

func (dmc directoriesMapChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(dmc.Directories)
}

type directoriesChannel struct {
	directories *state.Directories
	lock        *sync.RWMutex
	observers   map[string]chan state.DirectoriesChange
}

func newDirectoriesChannel(directories *state.Directories) *directoriesChannel {
	return &directoriesChannel{
		directories: directories,
		observers:   map[string]chan state.DirectoriesChange{},
		lock:        &sync.RWMutex{},
	}
}

func (dc *directoriesChannel) AddObserver(address string) {
	changes := make(chan state.DirectoriesChange)

	dc.lock.Lock()
	defer dc.lock.Unlock()

	dc.observers[address] = changes
}

func (dc *directoriesChannel) RemoveObserver(address string) {
	dc.lock.Lock()
	defer dc.lock.Unlock()

	changes, ok := dc.observers[address]
	if !ok {
		return
	}

	close(changes)
	delete(dc.observers, address)
}

func (dc *directoriesChannel) Replay(res ResponseWriter) error {
	return res.SendChange(directoriesMapChange{Directories: dc.directories.All()}, dc.Variant(), string(state.AddedDirectoriesChange))
}

func (dc *directoriesChannel) ServeObserver(address string, res ResponseWriter, done chan<- bool, errs chan<- error) {
	defer close(done)
	defer close(errs)

	changes, ok := dc.observers[address]
	if !ok {
		errs <- errors.New("no observer found for provided address")
		done <- true

		return
	}

	for {
		change, more := <-changes
		if !more {
			done <- true

			return
		}

		err := dc.changeHandler(res, change)
		if err != nil {
			errs <- err
		}
	}
}

func (dc *directoriesChannel) changeHandler(res ResponseWriter, change state.DirectoriesChange) error {
	return res.SendChange(change, dc.Variant(), string(state.AddedDirectoriesChange))
}

func (dc *directoriesChannel) BroadcastToChannelObservers(change state.DirectoriesChange) {
	dc.lock.RLock()
	defer dc.lock.RUnlock()

	for _, observer := range dc.observers {
		observer <- change
	}
}

func (dc directoriesChannel) Variant() state.SSEChannelVariant {
	return directoriesSSEChannelVariant
}
