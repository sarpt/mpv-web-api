package sse

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/state"
)

const (
	mediaFilesSSEChannelVariant state.SSEChannelVariant = "mediaFiles"
)

type mediaFilesMapChange struct {
	MediaFiles map[string]state.MediaFile
}

func (mmc mediaFilesMapChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(mmc.MediaFiles)
}

type mediaFilesChannel struct {
	mediaFiles *state.MediaFiles
	lock       *sync.RWMutex
	observers  map[string]chan state.MediaFilesChange
}

func newMediaFilesChannel(mediaFiles *state.MediaFiles) *mediaFilesChannel {
	return &mediaFilesChannel{
		mediaFiles: mediaFiles,
		observers:  map[string]chan state.MediaFilesChange{},
		lock:       &sync.RWMutex{},
	}
}

func (mfc *mediaFilesChannel) AddObserver(address string) {
	changes := make(chan state.MediaFilesChange)

	mfc.lock.Lock()
	defer mfc.lock.Unlock()

	mfc.observers[address] = changes
}

func (mfc *mediaFilesChannel) RemoveObserver(address string) {
	mfc.lock.Lock()
	defer mfc.lock.Unlock()

	changes, ok := mfc.observers[address]
	if !ok {
		return
	}

	close(changes)
	delete(mfc.observers, address)
}

func (mfc *mediaFilesChannel) Replay(res ResponseWriter) error {
	return res.SendChange(mediaFilesMapChange{MediaFiles: mfc.mediaFiles.All()}, mfc.Variant(), string(state.AddedMediaFilesChange))
}

func (mfc *mediaFilesChannel) ServeObserver(address string, res ResponseWriter, done chan<- bool, errs chan<- error) {
	defer close(done)
	defer close(errs)

	changes, ok := mfc.observers[address]
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

		err := mfc.changeHandler(res, change)
		if err != nil {
			errs <- err
		}
	}
}

func (mfc *mediaFilesChannel) changeHandler(res ResponseWriter, change state.MediaFilesChange) error {
	return res.SendChange(change, mfc.Variant(), string(change.Variant))
}

func (mfc *mediaFilesChannel) BroadcastToChannelObservers(change state.MediaFilesChange) {
	mfc.lock.RLock()
	defer mfc.lock.RUnlock()

	for _, observer := range mfc.observers {
		observer <- change
	}
}

func (mfc mediaFilesChannel) Variant() state.SSEChannelVariant {
	return mediaFilesSSEChannelVariant
}
