package common

import (
	"fmt"
	"path/filepath"
)

type Directory struct {
	Path      string
	Recursive bool
	Watched   bool
}

// EnsureDirectoryPath
func EnsureDirectoryPath(path string) string {
	if path[len(path)-1] == filepath.Separator {
		return path
	}

	return fmt.Sprintf("%s%c", path, filepath.Separator)
}
