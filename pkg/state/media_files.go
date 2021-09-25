package state

import (
	"encoding/json"
	"errors"
	"strings"
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
	Variant MediaFilesChangeVariant
	Items   map[string]MediaFile
}

// MarshalJSON returns change items in JSON format. Satisfies json.Marshaller.
func (mc MediaFilesChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(mc.Items)
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
	path := mediaFile.path

	m.lock.Lock()
	if _, ok := m.items[path]; ok {
		return
	}

	m.items[path] = mediaFile
	m.lock.Unlock()

	addedMediaFiles := map[string]MediaFile{
		path: mediaFile,
	}
	m.changes <- MediaFilesChange{
		Variant: AddedMediaFilesChange,
		Items:   addedMediaFiles,
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

	mediaFile, ok := m.items[path]
	if !ok {
		return MediaFile{}, errNoMediaFileAvailable
	}

	return mediaFile, nil
}

// ByParent returns media files with path under provided parent
// (path to directory).
func (m *MediaFiles) ByParent(parentPath string) []MediaFile {
	m.lock.RLock()
	defer m.lock.RUnlock()

	var mediaFiles []MediaFile
	for _, mediaFile := range m.items {
		if strings.HasPrefix(mediaFile.path, parentPath) {
			mediaFiles = append(mediaFiles, mediaFile)
		}
	}

	return mediaFiles
}

// Changes returns read-only channel notifying of mediaFiles changes.
func (m *MediaFiles) Changes() <-chan interface{} {
	return m.changes
}

// Exists checks whether media file with provided path exists.
func (m *MediaFiles) Exists(path string) bool {
	_, err := m.ByPath(path)

	return err == nil
}

// PathsUnderParent returns paths of media files under provided parent
// (path to directory).
func (m *MediaFiles) PathsUnderParent(parentPath string) []string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	var paths []string
	for _, mediaFile := range m.items {
		if strings.HasPrefix(mediaFile.path, parentPath) {
			paths = append(paths, mediaFile.path)
		}
	}

	return paths
}

// Take removes MediaFile by a provided path from the state,
// returning the object for use after removal.
// When media file cannot be found, the error is being reported.
func (m *MediaFiles) Take(path string) (MediaFile, error) {
	mediaFile, err := m.ByPath(path)
	if err != nil {
		return MediaFile{}, err
	}

	m.lock.Lock()
	delete(m.items, path)
	m.lock.Unlock()

	m.changes <- MediaFilesChange{
		Variant: RemovedMediaFilesChange,
		Items: map[string]MediaFile{
			mediaFile.path: mediaFile,
		},
	}

	return mediaFile, nil
}

// TakeMultiple removed MediaFiles with provided paths from the state,
// returning objects for use after removal as first return value,
// and skipped paths (not found ones) as a second return value.
func (m *MediaFiles) TakeMultiple(paths []string) ([]MediaFile, []string) {
	var skipped []string
	var taken []MediaFile

	change := MediaFilesChange{
		Variant: RemovedMediaFilesChange,
		Items:   map[string]MediaFile{},
	}

	for _, path := range paths {
		mediaFile, err := m.ByPath(path)
		if err != nil {
			skipped = append(skipped, path)
		}

		m.lock.Lock()
		delete(m.items, path)
		m.lock.Unlock()

		taken = append(taken, mediaFile)
		change.Items[mediaFile.path] = mediaFile
	}

	m.changes <- change

	return taken, skipped
}
