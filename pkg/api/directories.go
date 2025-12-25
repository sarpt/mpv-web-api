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
func (s *Server) readDirectory(path string, cacheEntry *CacheDirEntry) error {
	pathFs := os.DirFS(path)
	dirEntries, err := fs.ReadDir(pathFs, ".")
	if err != nil {
		return err
	}

	s.outLog.Printf("reading directory %s\n", path)
	var playlistUUIDs []string
	var playlistEntries []playlists.Entry
	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}

		entryPath := filepath.Join(path, entry.Name())

		if s.hasPlaylistFilePrefix(entryPath) {
			playlist, err := s.handlePlaylistFile(entryPath)
			if err == nil {
				if cacheEntry != nil {
					info, err := entry.Info()
					if err != nil {
						s.errLog.Printf("unable to read file info for path \"%s\": %s", entryPath, err)
					}

					cacheEntry.Playlists[entryPath] = CachePlaylistEntry{
						Mtime:    info.ModTime(), // TODO: currently ignored
						Playlist: *playlist,
					}
				}

				playlistUUIDs = append(playlistUUIDs, playlist.UUID())

				continue // successfuly handled playlist files don't need to be probed or handled in any other way
			}

			s.errLog.Printf("could not handle file with playlist prefix: %s", err)
		}

		result := probe.File(entryPath)
		if !result.IsMediaFile() {
			continue
		}

		mediaFile := media_files.MapProbeResultToMediaFile(result)
		s.statesRepository.MediaFiles().Add(mediaFile)
		if cacheEntry != nil {
			info, err := entry.Info()
			if err != nil {
				s.errLog.Printf("unable to read file info for path \"%s\": %s", entryPath, err)
			}

			cacheEntry.MediaFiles[entryPath] = CacheMediaFileEntry{
				Mtime: info.ModTime(), // TODO: currently ignored
				Entry: mediaFile,
			}
		}

		entry := playlists.Entry{
			Path: mediaFile.Path(),
		}
		playlistEntries = append(playlistEntries, entry)
	}

	// Update playlists content in with freshly read directory contents
	for _, uuid := range playlistUUIDs {
		playlist, err := s.statesRepository.Playlists().ByUUID(uuid)
		if err != nil {
			s.errLog.Printf("could not find playlists with provided uuid '%s': %s", uuid, err)

			continue
		}

		if !playlist.DirectoryContentsAsEntries() {
			continue
		}

		s.statesRepository.Playlists().SetPlaylistEntries(uuid, playlistEntries)
	}

	return err
}

func (s *Server) restoreDirectoryFromCache(cacheEntry *CacheDirEntry) {
	for _, cacheEntry := range cacheEntry.MediaFiles {
		s.statesRepository.MediaFiles().Add(cacheEntry.Entry)
	}

	for _, cacheEntry := range cacheEntry.Playlists {
		_, err := s.statesRepository.Playlists().AddPlaylist(&cacheEntry.Playlist)
		if err != nil {
			s.errLog.Printf("could not restore playlist '%s'", cacheEntry.Name())
		}
	}
}

// AddRootDirectories adds root directories with media files to be handled by the server.
// If the Directory entries are already present, they are overwritten along with their properties
// (watched, recursive, etc.).
// TODO2: at the moment no error is being returned from the directories adding,
// however some information about unsuccessful attempts should be returned
// in addition to just printing it in server (for example for REST responses).
func (s *Server) AddRootDirectories(rootDirectories []directories.Entry) {
	var (
		cache    *DirectoriesCache
		cacheErr error
	)

	if s.useCache {
		cache = &DirectoriesCache{
			Directories: map[string]*CacheDirEntry{},
		}

		if !s.clearCache {
			cache, cacheErr = loadDirectoriesCache(s.cacheDir)
			if cacheErr != nil {
				s.errLog.Printf("could not get cache for directory entries, using empty cache: %s", cacheErr)
			}
		} else {
			s.outLog.Println("using empty cache due to request to clear cache")
		}

		defer func() {
			s.outLog.Printf("saving directories entries cache to disk")
			err := saveDirectoriesCache(cache, s.cacheDir)
			if err != nil {
				s.errLog.Printf("could not save cache for directories: %s\n", err)
			}
		}()
	}

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

			cacheEntry := s.processCacheEntry(cache, path, dirEntry)
			subDir := directories.Entry{
				Path:      path,
				Recursive: rootDir.Recursive,
				Watched:   rootDir.Watched,
			}

			addDirErr := s.AddDirectory(subDir, cacheEntry)
			if addDirErr != nil {
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

func (s *Server) AddDirectory(dir directories.Entry, cacheEntry *CacheDirEntry) error {
	prevDir, err := s.statesRepository.Directories().ByPath(dir.Path)
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

	if s.useCache && cacheEntry != nil && !cacheEntry.stale {
		s.outLog.Printf("restoring \"%s\" from cache", dir.Path)
		s.restoreDirectoryFromCache(cacheEntry)
	} else {
		if s.useCache {
			s.outLog.Printf("cache unavailable or stale for entry \"%s\"", dir.Path)
		}

		err = s.readDirectory(dir.Path, cacheEntry)
		if err != nil {
			return err
		}
	}

	s.statesRepository.Directories().Add(dir)

	return nil
}

func (s *Server) TakeDirectory(path string) (directories.Entry, error) {
	dir, err := s.statesRepository.Directories().ByPath(path)
	if err != nil {
		return directories.Entry{}, fmt.Errorf("could not remove directory '%s' - directory was not added", path)
	}

	if dir.Watched {
		if err := s.fsWatcher.Remove(dir.Path); err != nil {
			return directories.Entry{}, fmt.Errorf("could not stop watching fs changes for a directory '%s': %s", path, err)
		}
	}

	dir, err = s.statesRepository.Directories().Take(path)
	if err != nil {
		return dir, fmt.Errorf("could not take directory '%s': %s", path, err)
	}

	filesToRemove := s.statesRepository.MediaFiles().PathsUnderParent(path)
	removedFiles, skippedFiles := s.statesRepository.MediaFiles().TakeMultiple(filesToRemove)
	if len(skippedFiles) != 0 {
		s.errLog.Printf("could not take following %d files: %s\n", len(skippedFiles), strings.Join(skippedFiles, ", "))
	}

	s.outLog.Printf("deleted directory '%s' and %d children media files\n", path, len(removedFiles))

	return dir, err
}
