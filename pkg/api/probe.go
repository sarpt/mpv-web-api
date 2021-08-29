package api

import (
	"errors"

	"github.com/sarpt/mpv-web-api/internal/state"
	"github.com/sarpt/mpv-web-api/pkg/probe"
)

var (
	// ErrPathNotDirectory occurs when provided path is not pointing to a directory.
	ErrPathNotDirectory = errors.New("path does not point to a directory")
)

func (s *Server) probeDirectory(directory string) {
	s.outLog.Printf("probing directory %s\n", directory)

	results := make(chan probe.Result)

	go probe.Directory(directory, results)
	for probeResult := range results {
		if !probeResult.IsMediaFile() {
			continue
		}

		mediaFile := state.MapProbeResultToMediaFile(probeResult)
		s.mediaFiles.Add(mediaFile)
	}

	s.outLog.Printf("finished probing directory %s\n", directory)
}
