package api

import (
	"fmt"
	"os"

	"github.com/sarpt/mpv-web-api/internal/common"
)

// ReadDirectory executes probing of directory concurrently.
func (s *Server) ReadDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrPathNotDirectory, path)
	}

	go s.probeDirectory(path)

	return nil
}

// AddDirectories adds root directories with media files to be handled by the server.
// If the Directory entries are already present, they are overwritten along with their properties
// (watched, recursive, etc.).
// TODO: add posibility to disable recursivity
func (s *Server) AddDirectories(directories []common.Directory) error {
	for _, dir := range directories {
		if dir.Watched {
			err := s.fsWatcher.Add(dir.Path)
			if err != nil {
				s.errLog.Println(err)
			}

			s.ReadDirectory(dir.Path)
		}
	}

	return nil
}
