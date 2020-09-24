package api

import "github.com/sarpt/mpv-web-api/pkg/probe"

// AddDirectories executes probing of each directory concurrently.
func (s *Server) AddDirectories(directories []string) {
	for _, directory := range directories {
		go s.probeDirectory(directory)
	}
}

func (s *Server) probeDirectory(directory string) []probe.SkippedFile {
	var movies []Movie

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

	return skippedFiles
}
