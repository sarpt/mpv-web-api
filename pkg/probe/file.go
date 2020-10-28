package probe

import (
	"encoding/json"
	"os/exec"
	"strconv"
)

const (
	videoCodecType    = "video"
	audioCodecType    = "audio"
	subtitleCodecType = "subtitle"

	ffprobeName     = "ffprobe"
	hideBannerArg   = "-hide_banner"
	logLevelArg     = "-loglevel"
	quietLogLevel   = "quiet"
	showErrorArg    = "-show_error"
	showStreamsArg  = "-show_streams"
	showFormatArg   = "-show_format"
	showChaptersArg = "-show_chapters"
	outputArg       = "-of"
	jsonOutput      = "json"
)

// Chapter specifies information about chapters included in the file
type Chapter struct {
	StartTime float64
	EndTime   float64
	Title     string
}

// SubtitleStream specifies information about subtitles inluded in the file
type SubtitleStream struct {
	Language   string
	SubtitleID string
	Title      string
}

// AudioStream specifies information about audio the file includes
type AudioStream struct {
	AudioID  string
	Channels int
	Language string
	Title    string
}

// VideoStream specifies information about video the file includes
type VideoStream struct {
	Height   int
	Language string
	Width    int
	Title    string
}

// Format specifies general information about media container file
type Format struct {
	Name     string
	LongName string
	Duration float64
	Title    string
}

// Result contains information about the file
type Result struct {
	Path            string
	Format          Format
	Chapters        []Chapter
	VideoStreams    []VideoStream
	AudioStreams    []AudioStream
	SubtitleStreams []SubtitleStream
	err             probeError
}

// File checks information about the file format, it's streams and whether it can be used as a media (movie)
// As of now it usses "ffprobe" ran as a separate process to get this information. May be changed to use libav go wrappers in the future
func File(filepath string) (Result, error) {
	result := Result{
		Path:            filepath,
		Format:          Format{},
		Chapters:        []Chapter{},
		VideoStreams:    []VideoStream{},
		AudioStreams:    []AudioStream{},
		SubtitleStreams: []SubtitleStream{},
	}

	ffprobeResult, err := probeWithFfprobe(filepath)
	if err != nil {
		return result, err
	}

	parsedDuration, err := strconv.ParseFloat(ffprobeResult.Format.Duration, 64)
	if err != nil {
		return result, err
	}

	result.Format = Format{
		Name:     ffprobeResult.Format.Name,
		LongName: ffprobeResult.Format.LongName,
		Duration: parsedDuration,
		Title:    ffprobeResult.Format.Tags.Title,
	}

	for _, chapter := range ffprobeResult.Chapters {
		startTime, err := strconv.ParseFloat(chapter.StartTime, 64)
		if err != nil {
			return result, err
		}

		endTime, err := strconv.ParseFloat(chapter.EndTime, 64)
		if err != nil {
			return result, err
		}

		result.Chapters = append(result.Chapters, Chapter{
			Title:     chapter.Tags.Title,
			StartTime: startTime,
			EndTime:   endTime,
		})
	}

	for _, str := range ffprobeResult.Streams {
		switch str.CodecType {
		case videoCodecType:
			result.VideoStreams = append(result.VideoStreams, VideoStream{
				Language: str.Tags.Language,
				Width:    str.Width,
				Height:   str.Height,
				Title:    str.Tags.Title,
			})
		case audioCodecType:
			result.AudioStreams = append(result.AudioStreams, AudioStream{
				AudioID:  strconv.FormatInt(int64(len(result.AudioStreams)+1), 10),
				Language: str.Tags.Language,
				Channels: str.Channels,
				Title:    str.Tags.Title,
			})
		case subtitleCodecType:
			result.SubtitleStreams = append(result.SubtitleStreams, SubtitleStream{
				SubtitleID: strconv.FormatInt(int64(len(result.SubtitleStreams)+1), 10),
				Language:   str.Tags.Language,
				Title:      str.Tags.Title,
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
		showChaptersArg,
		showFormatArg,
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
