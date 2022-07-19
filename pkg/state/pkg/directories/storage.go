package directories

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/state/internal/common"
)

var (
	errNoDirectoryAvailable = errors.New("directory does not exist")
)

const (
	// AddedDirectoriesChange notifies about addition of mediaFiles to the list of mediaFiles handled by the application.
	AddedDirectoriesChange ChangeVariant = "added"

	// UpdatedDirectoriesChange notifies about updates to the list of mediaFiles.
	UpdatedDirectoriesChange ChangeVariant = "updated"

	// RemovedDirectoriesChange notifies about removal of mediaFiles from the list.
	RemovedDirectoriesChange ChangeVariant = "removed"
)

type Subscriber = func(change Change)

// Change holds information about changes to the collection of directories being handled.
type Change struct {
	variant ChangeVariant
	items   map[string]Entry
}

// MarshalJSON returns change items in JSON format. Satisfies json.Marshaller.
func (d Change) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.items)
}

// ChangeVariant specifies what type of change to directories collection belong to in a DirectoriesChange type.
type ChangeVariant string

type Storage struct {
	broadcaster *common.ChangesBroadcaster
	items       map[string]Entry
	lock        *sync.RWMutex
}

// NewStorage counstructs Directories state.
func NewStorage() *Storage {
	broadcaster := common.NewChangesBroadcaster()
	broadcaster.Broadcast()

	return &Storage{
		broadcaster: broadcaster,
		items:       map[string]Entry{},
		lock:        &sync.RWMutex{},
	}
}

// Add appends a directory to the collection of directories handled by current server instance.
func (d *Storage) Add(dir Entry) {
	path := EnsureDirectoryPath(dir.Path)

	func() {
		d.lock.Lock()
		defer d.lock.Unlock()

		if _, ok := d.items[path]; ok {
			return
		}

		d.items[path] = dir
	}()

	d.broadcaster.Send(Change{
		variant: AddedDirectoriesChange,
		items: map[string]Entry{
			path: dir,
		},
	})
}

// All returns a copy of all Directories being handled by the instance of the server.
func (d *Storage) All() map[string]Entry {
	allDirectories := map[string]Entry{}

	d.lock.RLock()
	defer d.lock.RUnlock()

	for _, dir := range d.items {
		allDirectories[dir.Path] = dir
	}

	return allDirectories
}

// ByPath returns a directory by a provided path.
// When directory cannot be found, the error is being reported.
func (d *Storage) ByPath(path string) (Entry, error) {
	keyPath := EnsureDirectoryPath(path)

	d.lock.RLock()
	defer d.lock.RUnlock()

	dir, ok := d.items[keyPath]
	if !ok {
		return Entry{}, errNoDirectoryAvailable
	}

	return dir, nil
}

// Exists checks wheter directory under path is handled.
func (d *Storage) Exists(path string) bool {
	_, err := d.ByPath(path)

	return err == nil
}

// ParentByPath returns direct parent of the path.
// If not found, returns error errNoDirectoryAvailable.
func (d *Storage) ParentByPath(path string) (Entry, error) {
	dir, err := d.ByPath(filepath.Dir(path))
	if err != nil {
		return Entry{}, err
	}

	return dir, nil
}

func (p *Storage) Subscribe(sub Subscriber, onError func(err error)) {
	p.broadcaster.Subscribe(func(change interface{}) {
		directoriesChange, ok := change.(Change)
		if !ok {
			onError(common.ErrIncorrectChangesType)

			return
		}

		sub(directoriesChange)
	})
}

// Take removes directory by a provided path from the state,
// returning the object for use after removal.
// When directory cannot be found, the error is being reported.
func (d *Storage) Take(path string) (Entry, error) {
	keyPath := EnsureDirectoryPath(path)

	dir, err := d.ByPath(keyPath)
	if err != nil {
		return Entry{}, err
	}

	d.lock.Lock()
	delete(d.items, keyPath)
	d.lock.Unlock()

	d.broadcaster.Send(Change{
		variant: RemovedDirectoriesChange,
		items: map[string]Entry{
			keyPath: dir,
		},
	})

	return dir, nil
}

func EnsureDirectoryPath(path string) string {
	if path[len(path)-1] == filepath.Separator {
		return path
	}

	return fmt.Sprintf("%s%c", path, filepath.Separator)
}