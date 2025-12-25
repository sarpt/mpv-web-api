package api

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/media_files"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playlists"
)

type DirectoriesCache struct {
	Directories map[string]*CacheDirEntry `json:"Directories"`
}

type CacheDirEntry struct {
	Mtime      time.Time                      `json:"Mtime"`
	MediaFiles map[string]CacheMediaFileEntry `json:"MediaFiles"`
	Playlists  map[string]CachePlaylistEntry  `json:"Playlists"`
	stale      bool
}

type CacheMediaFileEntry struct {
	Mtime time.Time `json:"Mtime"`
	media_files.Entry
}

type CachePlaylistEntry struct {
	Mtime time.Time `json:"Mtime"`
	playlists.Playlist
}

func saveDirectoriesCache(cache *DirectoriesCache, directoriesCacheDir string) error {
	err := os.MkdirAll(directoriesCacheDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not create cache dir at path \"%s\": %w\n", directoriesCacheDir, err)
	}

	directoriesCachePath := path.Join(directoriesCacheDir, "directories")
	directoriesCacheJson, err := json.Marshal(&cache)
	if err != nil {
		return fmt.Errorf("could not marshall cache as a JSON: %w\n", err)
	}

	err = os.WriteFile(directoriesCachePath, directoriesCacheJson, os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not write directories cache contents to a file \"%s\", %w", directoriesCachePath, err)
	}

	return nil
}

func loadDirectoriesCache(cacheDir string) (*DirectoriesCache, error) {
	directoriesCache := DirectoriesCache{}
	directoriesCachePath := path.Join(cacheDir, "directories")
	directoriesCacheJson, err := os.ReadFile(directoriesCachePath)
	if err != nil {
		return &directoriesCache, fmt.Errorf("could not open cache file: %w\n", err)
	}

	err = json.Unmarshal(directoriesCacheJson, &directoriesCache)
	if err != nil {
		return &directoriesCache, fmt.Errorf("parsing cache entry failed: %w", err)
	}

	return &directoriesCache, nil
}

func (s *Server) processCacheEntry(cache *DirectoriesCache, path string, dirEntry fs.DirEntry) *CacheDirEntry {
	if cache == nil {
		return nil
	}

	var cacheEntry *CacheDirEntry
	dirInfo, err := dirEntry.Info()
	if err != nil {
		s.errLog.Printf("could not read directory \"%s\" information for modification time comparision", path)
	}

	entryMtime := dirInfo.ModTime()
	cacheEntry = cache.Directories[path]
	if cacheEntry != nil {
		restoreFromCache := !entryMtime.After(cacheEntry.Mtime)
		if !restoreFromCache {
			cacheEntry.Mtime = entryMtime
			cacheEntry.stale = true
		}

		return cacheEntry
	}

	cache.Directories[path] = &CacheDirEntry{
		Mtime:      entryMtime,
		MediaFiles: map[string]CacheMediaFileEntry{},
		Playlists:  map[string]CachePlaylistEntry{},
		stale:      true,
	}
	return cache.Directories[path]
}
