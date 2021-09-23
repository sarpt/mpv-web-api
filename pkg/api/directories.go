package api

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sarpt/mpv-web-api/internal/common"
	"github.com/sarpt/mpv-web-api/internal/state"
	"github.com/sarpt/mpv-web-api/pkg/probe"
)

// readDirectory tries to read and probe directory,
// adding found media files inside the directory.
func (s *Server) readDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrPathNotDirectory, path)
	}

	s.outLog.Printf("probing directory %s\n", path)
	err = probe.Directory(path, func(result probe.Result) {
		if !result.IsMediaFile() {
			return
		}

		mediaFile := state.MapProbeResultToMediaFile(result)
		s.mediaFiles.Add(mediaFile)
	})

	return err
}

// AddDirectories adds root directories with media files to be handled by the server.
// If the Directory entries are already present, they are overwritten along with their properties
// (watched, recursive, etc.).
// TODO: add posibility to disable recursivity
// TODO2: at the moment no error is being returned from the directories adding,
// however some information about unsuccessful attempts should be returned
// in addition to just printing it in server (for example for REST responses).
func (s *Server) AddDirectories(rootDirectories []common.Directory) {
	for _, rootDir := range rootDirectories {
		walkErr := filepath.WalkDir(rootDir.Path, func(path string, dirEntry fs.DirEntry, err error) error {
			if err != nil {
				s.errLog.Printf("could not process entry '%s': %s\n", path, err)
			}

			if !dirEntry.IsDir() {
				return nil
			}

			subDir := common.Directory{
				Path:      path,
				Recursive: rootDir.Recursive,
				Watched:   rootDir.Watched,
			}
			addDirErr := s.AddDirectory(subDir)
			if err != nil {
				s.errLog.Printf("could not add directory '%s': %s\n", path, addDirErr)
			} else {
				s.outLog.Printf("directory added %s\n", path)
			}

			return nil
		})

		if walkErr != nil {
			s.errLog.Printf("could not walk through the root directory '%s': %s\n", rootDir.Path, walkErr)
		}
	}
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

	err = s.readDirectory(dir.Path)
	if err != nil {
		return err
	}

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
