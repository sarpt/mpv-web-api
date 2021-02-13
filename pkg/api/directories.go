package api

import (
	"fmt"
	"os"
)

// AddDirectories executes probing of each directory concurrently.
func (s *Server) AddDirectories(directories []string) error {
	for _, directory := range directories {
		info, err := os.Stat(directory)
		if err != nil {
			return err // TODO: directories added before will still be added, so it needs to be refactored for directories to be checked before probing (or aggregate probing errors)
		}

		if !info.IsDir() {
			return fmt.Errorf("%w: %s", ErrPathNotDirectory, directory)
		}

		go s.probeDirectory(directory)
	}

	return nil
}
