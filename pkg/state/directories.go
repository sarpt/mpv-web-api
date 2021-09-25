package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
)

var (
	errNoDirectoryAvailable = errors.New("directory does not exist")
)

const (
	// AddedDirectoriesChange notifies about addition of mediaFiles to the list of mediaFiles handled by the application.
	AddedDirectoriesChange DirectoriesChangeVariant = "added"

	// UpdatedDirectoriesChange notifies about updates to the list of mediaFiles.
	UpdatedDirectoriesChange DirectoriesChangeVariant = "updated"

	// RemovedDirectoriesChange notifies about removal of mediaFiles from the list.
	RemovedDirectoriesChange DirectoriesChangeVariant = "removed"
)

// DirectoriesChange holds information about changes to the collection of directories being handled.
type DirectoriesChange struct {
	variant DirectoriesChangeVariant
	items   map[string]Directory
}

// MarshalJSON returns change items in JSON format. Satisfies json.Marshaller.
func (d DirectoriesChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.items)
}

// DirectoriesChangeVariant specifies what type of change to directories collection belong to in a DirectoriesChange type.
type DirectoriesChangeVariant string

type Directories struct {
	changes chan interface{}
	items   map[string]Directory
	lock    *sync.RWMutex
}

// NewDirectories counstructs Directories state.
func NewDirectories() *Directories {
	return &Directories{
		changes: make(chan interface{}),
		items:   map[string]Directory{},
		lock:    &sync.RWMutex{},
	}
}

// Add appends a directory to the collection of directories handled by current server instance.
func (d *Directories) Add(dir Directory) {
	path := ensureDirectoryPath(dir.Path)

	func() {
		d.lock.Lock()
		defer d.lock.Unlock()

		if _, ok := d.items[path]; ok {
			return
		}

		d.items[path] = dir
	}()

	d.changes <- DirectoriesChange{
		variant: AddedDirectoriesChange,
		items: map[string]Directory{
			path: dir,
		},
	}
}

// All returns a copy of all Directories being handled by the instance of the server.
func (d *Directories) All() map[string]Directory {
	allDirectories := map[string]Directory{}

	d.lock.RLock()
	defer d.lock.RUnlock()

	for _, dir := range d.items {
		allDirectories[dir.Path] = dir
	}

	return allDirectories
}

// ByPath returns a directory by a provided path.
// When directory cannot be found, the error is being reported.
func (d *Directories) ByPath(path string) (Directory, error) {
	keyPath := ensureDirectoryPath(path)

	d.lock.RLock()
	defer d.lock.RUnlock()

	dir, ok := d.items[keyPath]
	if !ok {
		return Directory{}, errNoDirectoryAvailable
	}

	return dir, nil
}

// Exists checks wheter directory under path is handled.
func (d *Directories) Exists(path string) bool {
	_, err := d.ByPath(path)

	return err == nil
}

// ParentByPath returns direct parent of the path.
// If not found, returns error errNoDirectoryAvailable.
func (d *Directories) ParentByPath(path string) (Directory, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	dir, err := d.ByPath(filepath.Dir(path))
	if err != nil {
		return Directory{}, err
	}

	return dir, nil
}

// Take removes directory by a provided path from the state,
// returning the object for use after removal.
// When directory cannot be found, the error is being reported.
func (d *Directories) Take(path string) (Directory, error) {
	keyPath := ensureDirectoryPath(path)

	dir, err := d.ByPath(keyPath)
	if err != nil {
		return Directory{}, err
	}

	d.lock.Lock()
	delete(d.items, keyPath)
	d.lock.Unlock()

	d.changes <- DirectoriesChange{
		variant: RemovedDirectoriesChange,
		items: map[string]Directory{
			keyPath: dir,
		},
	}

	return dir, nil
}

// Changes returns read-only channel notifying of mediaFiles changes.
func (d *Directories) Changes() <-chan interface{} {
	return d.changes
}

func ensureDirectoryPath(path string) string {
	if path[len(path)-1] == filepath.Separator {
		return path
	}

	return fmt.Sprintf("%s%c", path, filepath.Separator)
}
