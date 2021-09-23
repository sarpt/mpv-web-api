package probe

import (
	"io/fs"
	"os"
	"path/filepath"
)

type ProbeResultHandler = func(Result)

// Directory takes a directory path that should be checked for media files,
// and a handler for each handler directory entry.
// Returns error in case reading of the directory does not succeed.
func Directory(path string, handler ProbeResultHandler) error {
	pathFs := os.DirFS(path)
	dirEntries, err := fs.ReadDir(pathFs, ".")
	if err != nil {
		return err
	}

	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}

		entryPath := filepath.Join(path, entry.Name())
		result := File(entryPath)
		handler(result)
	}

	return err
}
