package api

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sarpt/mpv-web-api/pkg/probe"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/directories"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/media_files"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playlists"
)

// readDirectory tries to read and probe directory,
// adding found media files inside the directory.
func (s *Server) readDirectory(path string) error {
	pathFs := os.DirFS(path)
	dirEntries, err := fs.ReadDir(pathFs, ".")
	if err != nil {
		return err
	}

	s.outLog.Printf("reading directory %s\n", path)
	var playlistUUIDs []string
	var playlistEntries []playlists.PlaylistEntry
	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}

		entryPath := filepath.Join(path, entry.Name())

		if s.hasPlaylistFilePrefix(entryPath) {
			uuid, err := s.handlePlaylistFile(entryPath)
			if err == nil {
				playlistUUIDs = append(playlistUUIDs, uuid)

				continue // successfuly handled playlist files don't need to be probed or handled in any other way
			}

			s.errLog.Printf("could not handle file with playlist prefix: %s", err)
		}

		result := probe.File(entryPath)
		if !result.IsMediaFile() {
			continue
		}

		mediaFile := media_files.MapProbeResultToMediaFile(result)
		s.mediaFiles.Add(mediaFile)

		entry := playlists.PlaylistEntry{
			Path: mediaFile.Path(),
		}
		playlistEntries = append(playlistEntries, entry)
	}

	for _, uuid := range playlistUUIDs {
		playlist, err := s.playlists.ByUUID(uuid)
		if err != nil {
			s.errLog.Printf("could not find playlists with provided uuid '%s': %s", uuid, err)

			continue
		}

		if !playlist.DirectoryContentsAsEntries() {
			continue
		}

		s.playlists.SetPlaylistEntries(uuid, playlistEntries)
	}

	return err
}

// AddRootDirectories adds root directories with media files to be handled by the server.
// If the Directory entries are already present, they are overwritten along with their properties
// (watched, recursive, etc.).
// TODO2: at the moment no error is being returned from the directories adding,
// however some information about unsuccessful attempts should be returned
// in addition to just printing it in server (for example for REST responses).
func (s *Server) AddRootDirectories(rootDirectories []directories.Directory) {
	for _, rootDir := range rootDirectories {
		rootPath := directories.EnsureDirectoryPath(rootDir.Path)

		walkErr := filepath.WalkDir(rootPath, func(path string, dirEntry fs.DirEntry, err error) error {
			if err != nil {
				s.errLog.Printf("could not process entry '%s': %s\n", path, err)

				return err
			}

			if !dirEntry.IsDir() {
				return nil
			}

			if rootPath != path && !rootDir.Recursive {
				return fs.SkipDir
			}

			subDir := directories.Directory{
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

func (s *Server) AddDirectory(dir directories.Directory) error {
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

func (s *Server) TakeDirectory(path string) (directories.Directory, error) {
	dir, err := s.directories.ByPath(path)
	if err != nil {
		return directories.Directory{}, fmt.Errorf("could not remove directory '%s' - directory was not added", path)
	}

	if dir.Watched {
		if err := s.fsWatcher.Remove(dir.Path); err != nil {
			return directories.Directory{}, fmt.Errorf("could not stop watching fs changes for a directory '%s': %s", path, err)
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
