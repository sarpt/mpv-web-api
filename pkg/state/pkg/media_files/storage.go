package media_files

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/sarpt/mpv-web-api/internal/common"
	"github.com/sarpt/mpv-web-api/pkg/state/internal/revision"
)

var (
	errNoMediaFileAvailable = errors.New("media file with specified path does not exist")
)

const (
	// AddedMediaFilesChange notifies about addition of mediaFiles to the list of mediaFiles handled by the application.
	AddedMediaFilesChange common.ChangeVariant = "added"

	// UpdatedMediaFilesChange notifies about updates to the list of mediaFiles.
	UpdatedMediaFilesChange common.ChangeVariant = "updated"

	// RemovedMediaFilesChange notifies about removal of mediaFiles from the list.
	RemovedMediaFilesChange common.ChangeVariant = "removed"
)

type SubscriberCB = func(change Change)

type mediaFilesChangeSubscriber struct {
	cb SubscriberCB
}

func (s *mediaFilesChangeSubscriber) Receive(change Change) {
	s.cb(change)
}

// Change holds information about changes to the list of mediaFiles being served.
type Change struct {
	ChangeVariant common.ChangeVariant
	Items         map[string]Entry
}

// MarshalJSON returns change items in JSON format. Satisfies json.Marshaller.
func (mc Change) MarshalJSON() ([]byte, error) {
	return json.Marshal(mc.Items)
}

func (mc Change) Variant() common.ChangeVariant {
	return mc.ChangeVariant
}

// Storage is an aggregate state of the media files being served by the server instance.
// Any modification done on the state should be done by exposed methods which should guarantee goroutine access safety.
type Storage struct {
	broadcaster *common.ChangesBroadcaster[Change]
	items       map[string]Entry
	lock        *sync.RWMutex
	revision    *revision.Storage
}

// NewStorage counstructs MediaFiles state.
func NewStorage(broadcaster *common.ChangesBroadcaster[Change]) *Storage {
	return &Storage{
		broadcaster: broadcaster,
		items:       map[string]Entry{},
		lock:        &sync.RWMutex{},
		revision:    revision.NewStorage(),
	}
}

// Add appends a mediaFile to the list of mediaFiles served on current server instance.
func (m *Storage) Add(mediaFile Entry) {
	path := mediaFile.path

	go func() {
		m.lock.Lock()
		defer m.lock.Unlock()
		if _, ok := m.items[path]; ok {
			return
		}

		m.items[path] = mediaFile
	}()

	m.revision.Tick()
	addedMediaFiles := map[string]Entry{
		path: mediaFile,
	}
	m.broadcaster.Send(Change{
		ChangeVariant: AddedMediaFilesChange,
		Items:         addedMediaFiles,
	})
}

// All returns a copy of all MediaFiles being served by the instance of the server.
func (m *Storage) All() map[string]Entry {
	allMediaFiles := map[string]Entry{}

	m.lock.RLock()
	defer m.lock.RUnlock()

	for path, mediaFile := range m.items {
		allMediaFiles[path] = mediaFile
	}

	return allMediaFiles
}

// ByPath returns a MediaFile by a provided path.
// When media file cannot be found, the error is being reported.
func (m *Storage) ByPath(path string) (Entry, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	mediaFile, ok := m.items[path]
	if !ok {
		return Entry{}, errNoMediaFileAvailable
	}

	return mediaFile, nil
}

// ByParent returns media files with path under provided parent
// (path to directory).
func (m *Storage) ByParent(parentPath string) []Entry {
	m.lock.RLock()
	defer m.lock.RUnlock()

	var mediaFiles []Entry
	for _, mediaFile := range m.items {
		if strings.HasPrefix(mediaFile.path, parentPath) {
			mediaFiles = append(mediaFiles, mediaFile)
		}
	}

	return mediaFiles
}

// ByUuid returns a MediaFile by a provided uuid.
// When media file cannot be found, the error is being reported.
func (m *Storage) ByUuid(uuid string) (Entry, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	var foundMediaFile *Entry
	for _, mediaFile := range m.items {
		if mediaFile.uuid == uuid {
			foundMediaFile = &mediaFile
			break
		}
	}

	if foundMediaFile == nil {
		return Entry{}, errNoMediaFileAvailable
	}

	return *foundMediaFile, nil
}

// Exists checks whether media file with provided path exists.
func (m *Storage) Exists(path string) bool {
	_, err := m.ByPath(path)

	return err == nil
}

// PathsUnderParent returns paths of media files under provided parent
// (path to directory).
func (m *Storage) PathsUnderParent(parentPath string) []string {
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

func (m *Storage) Revision() revision.Identifier {
	return m.revision.Revision()
}

func (p *Storage) Subscribe(cb SubscriberCB, onError func(err error)) func() {
	subscriber := mediaFilesChangeSubscriber{
		cb,
	}

	return p.broadcaster.Subscribe(&subscriber)
}

// Take removes MediaFile by a provided path from the state,
// returning the object for use after removal.
// When media file cannot be found, the error is being reported.
func (m *Storage) Take(path string) (Entry, error) {
	mediaFile, err := m.ByPath(path)
	if err != nil {
		return Entry{}, err
	}

	m.lock.Lock()
	delete(m.items, path)
	m.lock.Unlock()

	m.revision.Tick()
	m.broadcaster.Send(Change{
		ChangeVariant: RemovedMediaFilesChange,
		Items: map[string]Entry{
			mediaFile.path: mediaFile,
		},
	})

	return mediaFile, nil
}

// TakeMultiple removed MediaFiles with provided paths from the state,
// returning objects for use after removal as first return value,
// and skipped paths (not found ones) as a second return value.
func (m *Storage) TakeMultiple(paths []string) ([]Entry, []string) {
	var skipped []string
	var taken []Entry

	change := Change{
		ChangeVariant: RemovedMediaFilesChange,
		Items:         map[string]Entry{},
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

	m.revision.Tick()
	m.broadcaster.Send(change)

	return taken, skipped
}
