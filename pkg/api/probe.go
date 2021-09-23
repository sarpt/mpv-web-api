package api

import (
	"errors"
	"fmt"

	"github.com/sarpt/mpv-web-api/internal/state"
	"github.com/sarpt/mpv-web-api/pkg/probe"
)

var (
	// ErrPathNotDirectory occurs when provided path is not pointing to a directory.
	ErrPathNotDirectory = errors.New("path does not point to a directory")
)

func (s *Server) probeFile(path string) (state.MediaFile, error) {
	s.outLog.Printf("probing file %s\n", path)

	probeResult := probe.File(path)
	if probeResult.Err != nil {
		return state.MediaFile{}, fmt.Errorf("error while probing '%s' file: %s", path, probeResult.Err)
	}

	if !probeResult.IsMediaFile() {
		return state.MediaFile{}, fmt.Errorf("file '%s' is not a media file", path)
	}

	mediaFile := state.MapProbeResultToMediaFile(probeResult)
	s.outLog.Printf("finished probing file %s\n", path)

	return mediaFile, nil
}
