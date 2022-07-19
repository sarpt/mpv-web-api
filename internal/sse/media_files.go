package sse

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/media_files"
	state_sse "github.com/sarpt/mpv-web-api/pkg/state/pkg/sse"
)

const (
	mediaFilesSSEChannelVariant state_sse.ChannelVariant = "mediaFiles"
)

type mediaFilesMapChange struct {
	MediaFiles map[string]media_files.MediaFile
}

func (mmc mediaFilesMapChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(mmc.MediaFiles)
}

type mediaFilesChannel struct {
	mediaFiles *media_files.MediaFiles
	lock       *sync.RWMutex
	observers  map[string]chan media_files.MediaFilesChange
}

func newMediaFilesChannel(mediaFilesStorage *media_files.MediaFiles) *mediaFilesChannel {
	return &mediaFilesChannel{
		mediaFiles: mediaFilesStorage,
		observers:  map[string]chan media_files.MediaFilesChange{},
		lock:       &sync.RWMutex{},
	}
}

func (mfc *mediaFilesChannel) AddObserver(address string) {
	changes := make(chan media_files.MediaFilesChange)

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
	return res.SendChange(mediaFilesMapChange{MediaFiles: mfc.mediaFiles.All()}, mfc.Variant(), string(media_files.AddedMediaFilesChange))
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

func (mfc *mediaFilesChannel) changeHandler(res ResponseWriter, change media_files.MediaFilesChange) error {
	return res.SendChange(change, mfc.Variant(), string(change.Variant))
}

func (mfc *mediaFilesChannel) BroadcastToChannelObservers(change media_files.MediaFilesChange) {
	mfc.lock.RLock()
	defer mfc.lock.RUnlock()

	for _, observer := range mfc.observers {
		observer <- change
	}
}

func (mfc mediaFilesChannel) Variant() state_sse.ChannelVariant {
	return mediaFilesSSEChannelVariant
}
