package api

import (
	"errors"
	"fmt"
	"os"

	"github.com/sarpt/mpv-web-api/pkg/probe"
)

var (
	// ErrPathNotDirectory occurs when provided path is not pointing to a directory.
	ErrPathNotDirectory = errors.New("path does not point to a directory")
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

func (s *Server) probeDirectory(directory string) []probe.SkippedFile {
	var movies []Movie
	s.outLog.Printf("probing directory %s\n", directory)

	probeResults, skippedFiles := probe.Directory(directory)
	for _, probeResult := range probeResults {
		if !probeResult.IsMovieFile() {
			continue
		}

		movie := mapProbeResultToMovie(probeResult)
		movies = append(movies, movie)
	}

	s.moviesLock.Lock()
	s.movies = append(s.movies, movies...)
	s.moviesLock.Unlock()

	s.outLog.Printf("finished probing directory %s\n", directory)
	return skippedFiles
}
