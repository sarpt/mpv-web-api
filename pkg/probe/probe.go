package probe

import (
	"encoding/json"
	"os/exec"
)

const (
	videoCodecType    = "video"
	audioCodecType    = "audio"
	subtitleCodecType = "subtitle"

	ffprobeName    = "ffprobe"
	hideBannerArg  = "-hide_banner"
	logLevelArg    = "-loglevel"
	quietLogLevel  = "quiet"
	showErrorArg   = "-show_error"
	showStreamsArg = "-show_streams"
	outputArg      = "-of"
	jsonOutput     = "json"
)

// SubtitleStream specifies information about subtitles inluded in the movie
type SubtitleStream struct {
	Language string
}

// AudioStream specifies information about audio the movie includes
type AudioStream struct {
	Language string
	Channels int
}

// VideoStream specifies information about video the movie includes
type VideoStream struct {
	Language string
	Width    int
	Height   int
}

// Result contains information about the file
type Result struct {
	VideoStreams    []VideoStream
	AudioStreams    []AudioStream
	SubtitleStreams []SubtitleStream
	err             probeError
}

// File checks information about the file format, it's streams and whether it can be used as a media (movie)
// As of now it usses "ffprobe" ran as a separate process to get this information. May be changed to use libav go wrappers in the future
func File(filepath string) (Result, error) {
	result := Result{}

	ffprobeResult, err := probeWithFfprobe(filepath)
	if err != nil {
		return result, err
	}

	for _, str := range ffprobeResult.Streams {
		switch str.CodecType {
		case videoCodecType:
			result.VideoStreams = append(result.VideoStreams, VideoStream{
				Language: str.Tags.Language,
				Width:    str.Width,
				Height:   str.Height,
			})
		case audioCodecType:
			result.AudioStreams = append(result.AudioStreams, AudioStream{
				Language: str.Tags.Language,
				Channels: str.Channels,
			})
		case subtitleCodecType:
			result.SubtitleStreams = append(result.SubtitleStreams, SubtitleStream{
				Language: str.Tags.Language,
			})
		}
	}

	return result, nil
}

func probeWithFfprobe(filepath string) (ffprobeResult, error) {
	result := ffprobeResult{}

	ffprobeargs := []string{
		hideBannerArg,
		logLevelArg, quietLogLevel,
		showErrorArg,
		showStreamsArg,
		outputArg, jsonOutput,
		filepath,
	}
	cmd := exec.Command(ffprobeName, ffprobeargs...)

	output, err := cmd.Output()
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(output, &result)
	return result, err
}

// IsMovieFile checks whether parsing of file was successful, and whether any video streams are present in the file (audio or subtitles optional)
func (res Result) IsMovieFile() bool {
	if res.err.Code != 0 {
		return false
	}

	return len(res.VideoStreams) != 0
}
