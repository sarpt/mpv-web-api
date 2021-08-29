package state

import (
	"encoding/json"
	"errors"
	"sync"
)

var (
	errNoMediaFileAvailable = errors.New("media file with specified path does not exist")
)

const (
	// AddedMediaFilesChange notifies about addition of mediaFiles to the list of mediaFiles handled by the application.
	AddedMediaFilesChange MediaFilesChangeVariant = "added"

	// UpdatedMediaFilesChange notifies about updates to the list of mediaFiles.
	UpdatedMediaFilesChange MediaFilesChangeVariant = "updated"

	// RemovedMediaFilesChange notifies about removal of mediaFiles from the list.
	RemovedMediaFilesChange MediaFilesChangeVariant = "removed"
)

// MediaFilesChange holds information about changes to the list of mediaFiles being served.
type MediaFilesChange struct {
	variant MediaFilesChangeVariant
	items   map[string]MediaFile
}

// MarshalJSON returns change items in JSON format. Satisfies json.Marshaller.
func (mc MediaFilesChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(mc.items)
}

// MediaFilesChangeVariant specifies what type of change to mediaFiles list items belong to in a MediaFilesChange type.
type MediaFilesChangeVariant string

// MediaFiles is an aggregate state of the media files being served by the server instance.
// Any modification done on the state should be done by exposed methods which should guarantee goroutine access safety.
type MediaFiles struct {
	items   map[string]MediaFile
	changes chan interface{}
	lock    *sync.RWMutex
}

// NewMediaFiles counstructs MediaFiles state.
func NewMediaFiles() *MediaFiles {
	return &MediaFiles{
		items:   map[string]MediaFile{},
		changes: make(chan interface{}),
		lock:    &sync.RWMutex{},
	}
}

// Add appends a mediaFile to the list of mediaFiles served on current server instance.
func (m *MediaFiles) Add(mediaFile MediaFile) {
	addedMediaFiles := map[string]MediaFile{}
	path := mediaFile.path

	m.lock.Lock()
	if _, ok := m.items[path]; ok {
		return
	}

	m.items[path] = mediaFile
	m.lock.Unlock()

	addedMediaFiles[path] = mediaFile
	m.changes <- MediaFilesChange{
		variant: AddedMediaFilesChange,
		items:   addedMediaFiles,
	}
}

// All returns a copy of all MediaFiles being served by the instance of the server.
func (m *MediaFiles) All() map[string]MediaFile {
	allMediaFiles := map[string]MediaFile{}

	m.lock.RLock()
	defer m.lock.RUnlock()

	for path, mediaFile := range m.items {
		allMediaFiles[path] = mediaFile
	}

	return allMediaFiles
}

// ByPath returns a MediaFile by a provided path.
// When media file cannot be found, the error is being reported.
func (m *MediaFiles) ByPath(path string) (MediaFile, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, mediaFile := range m.items {
		if mediaFile.path == path {
			return mediaFile, nil
		}
	}

	return MediaFile{}, errNoMediaFileAvailable
}

// Changes returns read-only channel notifying of mediaFiles changes.
func (m *MediaFiles) Changes() <-chan interface{} {
	return m.changes
}
