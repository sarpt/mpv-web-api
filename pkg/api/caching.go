package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/media_files"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playlists"
	"github.com/ulikunitz/xz"
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
	directoriesCacheContent, err := json.Marshal(&cache)
	if err != nil {
		return fmt.Errorf("could not marshall cache as a JSON: %w\n", err)
	}

	err = os.WriteFile(directoriesCachePath, directoriesCacheContent, os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not write directories cache contents to a file \"%s\", %w", directoriesCachePath, err)
	}

	return nil
}

const xzMagicSize = 6

var xzMagic = []byte{0xfd, 0x37, 0x7A, 0x58, 0x5a, 0x00}

func loadDirectoriesCache(cacheDir string) (*DirectoriesCache, error) {
	directoriesCache := DirectoriesCache{
		Directories: map[string]*CacheDirEntry{},
	}
	directoriesCachePath := path.Join(cacheDir, "directories")
	directoriesCacheFile, err := os.Open(directoriesCachePath)
	if err != nil {
		return &directoriesCache, fmt.Errorf("could not open cache directoriesCacheFile: %w\n", err)
	}

	fileMagic := make([]byte, xzMagicSize)
	readN, err := directoriesCacheFile.Read(fileMagic)
	if err != nil || readN != xzMagicSize {
		return &directoriesCache, fmt.Errorf("could not check magic of cache file: %w\n", err)
	}

	_, err = directoriesCacheFile.Seek(0, 0)
	if err != nil {
		return &directoriesCache, fmt.Errorf("seeking to beggining of cache buffer failed: %w\n", err)
	}

	var directoriesCacheReader io.Reader = directoriesCacheFile
	// directoriesCacheJson := &bytes.Buffer{}
	if bytes.Equal(fileMagic, xzMagic) {
		xzReader, err := xz.NewReader(directoriesCacheFile)
		if err != nil {
			return nil, fmt.Errorf("reading xz cache content failed: %w", err)
		}

		directoriesCacheReader = xzReader
		// _, err = io.Copy(directoriesCacheJson, xzReader)
		// if err != nil {
		// 	return &directoriesCache, fmt.Errorf("decoding of xz content failed: %w", err)
		// }
	}
	// } else {
	// 	// support older cache files not compressed with xz
	// 	_, err = io.Copy(directoriesCacheJson, directoriesCacheFile)
	// 	if err != nil {
	// 		return &directoriesCache, fmt.Errorf("copy of json content failed: %w", err)
	// 	}
	// }

	err = json.NewDecoder(directoriesCacheReader).Decode(&directoriesCache)
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
