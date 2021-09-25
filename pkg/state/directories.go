package state

import (
	"encoding/json"
	"errors"
	"sync"
)

var (
	errNoDirectoryAvailable = errors.New("directory with specified path does not exist")
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
	path := dir.Path

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

	for path, dir := range d.items {
		allDirectories[path] = dir
	}

	return allDirectories
}

// ByPath returns a directory by a provided path.
// When directory cannot be found, the error is being reported.
func (d *Directories) ByPath(path string) (Directory, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	for _, dir := range d.items {
		if dir.Path == path {
			return dir, nil
		}
	}

	return Directory{}, errNoDirectoryAvailable
}

// Exists checks wheter directory under path is handled.
func (d *Directories) Exists(path string) bool {
	_, err := d.ByPath(path)

	return err == nil
}

// Take removes directory by a provided path from the state,
// returning the object for use after removal.
// When directory cannot be found, the error is being reported.
func (d *Directories) Take(path string) (Directory, error) {
	dir, err := d.ByPath(path)
	if err != nil {
		return Directory{}, err
	}

	d.lock.Lock()
	delete(d.items, path)
	d.lock.Unlock()

	d.changes <- DirectoriesChange{
		variant: RemovedDirectoriesChange,
		items: map[string]Directory{
			path: dir,
		},
	}

	return dir, nil
}

// Changes returns read-only channel notifying of mediaFiles changes.
func (d *Directories) Changes() <-chan interface{} {
	return d.changes
}
