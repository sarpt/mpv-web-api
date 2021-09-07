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
	// TODO: probe.Directory probably should be changed to probe.Directories(paths) or removed altogether,
	// with probeDirectory from server taking walking the tree responsibilities
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

func (s *Server) probeFile(path string) {
	s.outLog.Printf("probing file %s\n", path)

	probeResult, err := probe.File(path)
	if err != nil {
		s.errLog.Printf("error while probing '%s' file: %s", path, err)
		return
	}

	if !probeResult.IsMediaFile() {
		return
	}

	mediaFile := state.MapProbeResultToMediaFile(probeResult)
	s.mediaFiles.Add(mediaFile)

	s.outLog.Printf("finished probing file %s\n", path)
}
