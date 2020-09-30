package api

import (
	"errors"

	"github.com/sarpt/mpv-web-api/pkg/probe"
)

var (
	// ErrPathNotDirectory occurs when provided path is not pointing to a directory.
	ErrPathNotDirectory = errors.New("path does not point to a directory")
)

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

	s.AddMovies(movies)

	s.outLog.Printf("finished probing directory %s\n", directory)
	return skippedFiles
}
