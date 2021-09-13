package api

import (
	"fmt"
	"os"
	"strings"

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
		err := s.AddDirectory(dir)
		if err != nil {
			s.errLog.Printf("could not add directory '%s': %s\n", dir.Path, err)
		}
	}

	return nil
}

func (s *Server) AddDirectory(dir common.Directory) error {
	prevDir, err := s.directories.ByPath(dir.Path)
	if err == nil && prevDir.Watched {
		err := s.fsWatcher.Remove(prevDir.Path)
		if err != nil {
			return err
		}
	}

	if dir.Watched {
		err := s.fsWatcher.Add(dir.Path)
		if err != nil {
			return err
		}
	}

	s.ReadDirectory(dir.Path)
	s.directories.Add(dir)

	return nil
}

func (s *Server) TakeDirectory(path string) (common.Directory, error) {
	dir, err := s.directories.ByPath(path)
	if err != nil {
		return common.Directory{}, fmt.Errorf("could not remove directory '%s' - directory was not added", path)
	}

	if dir.Watched {
		if err := s.fsWatcher.Remove(dir.Path); err != nil {
			return common.Directory{}, fmt.Errorf("could not stop watching fs changes for a directory '%s': %s", path, err)
		}
	}

	dir, err = s.directories.Take(path)
	if err != nil {
		return dir, fmt.Errorf("could not take directory '%s': %s", path, err)
	}

	filesToRemove := s.mediaFiles.PathsUnderParent(path)
	removedFiles, skippedFiles := s.mediaFiles.TakeMultiple(filesToRemove)
	if len(skippedFiles) != 0 {
		s.errLog.Printf("could not take following %d files: %s\n", len(skippedFiles), strings.Join(skippedFiles, ", "))
	}

	s.outLog.Printf("deleted directory '%s' and %d children media files\n", path, len(removedFiles))

	return dir, err
}
