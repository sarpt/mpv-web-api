package api

import (
	"os"
	"path/filepath"

	"github.com/sarpt/mpv-web-api/pkg/probe"
)

func moviesInDirectories(directories []string) []Movie {
	var movies []Movie

	for _, directory := range directories {
		filepath.Walk(directory, func(path string, info os.FileInfo, err error) error { // TODO: add some kind of error handling
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			pr, err := probe.File(path)
			if err != nil {
				return err
			}

			if !pr.IsMovieFile() {
				return nil
			}

			movie := Movie{
				Path:            path,
				VideoStreams:    pr.VideoStreams,
				AudioStreams:    pr.AudioStreams,
				SubtitleStreams: pr.SubtitleStreams,
			}
			movies = append(movies, movie)

			return nil
		})
	}

	return movies
}
