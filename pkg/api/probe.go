package api

import (
	"errors"
	"fmt"

	"github.com/sarpt/mpv-web-api/pkg/probe"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/media_files"
)

var (
	// ErrPathNotDirectory occurs when provided path is not pointing to a directory.
	ErrPathNotDirectory = errors.New("path does not point to a directory")
)

func (s *Server) probeFile(path string) (media_files.MediaFile, error) {
	s.outLog.Printf("probing file %s\n", path)

	probeResult := probe.File(path)
	if probeResult.Err != nil {
		return media_files.MediaFile{}, fmt.Errorf("error while probing '%s' file: %s", path, probeResult.Err)
	}

	if !probeResult.IsMediaFile() {
		return media_files.MediaFile{}, fmt.Errorf("file '%s' is not a media file", path)
	}

	mediaFile := media_files.MapProbeResultToMediaFile(probeResult)
	s.outLog.Printf("finished probing file %s\n", path)

	return mediaFile, nil
}
