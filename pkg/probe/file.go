package probe

import (
	"encoding/json"
	"fmt"
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
	ChapterID string  `json:"ChapterID"`
	EndTime   float64 `json:"EndTime"`
	StartTime float64 `json:"StartTime"`
	Title     string  `json:"Title"`
}

// SubtitleStream specifies information about subtitles inluded in the file
type SubtitleStream struct {
	Language   string `json:"Language"`
	SubtitleID string `json:"SubtitleID"`
	Title      string `json:"Title"`
}

// AudioStream specifies information about audio the file includes
type AudioStream struct {
	AudioID  string `json:"AudioID"`
	Channels int    `json:"Channels"`
	Language string `json:"Language"`
	Title    string `json:"Title"`
}

// VideoStream specifies information about video the file includes
type VideoStream struct {
	Height   int    `json:"Height"`
	Language string `json:"Language"`
	Width    int    `json:"Width"`
	Title    string `json:"Title"`
}

// Format specifies general information about media container file
type Format struct {
	Name     string  `json:"Name"`
	LongName string  `json:"LongName"`
	Duration float64 `json:"Duration"`
	Title    string  `json:"Title"`
}

// Result contains information about the file
type Result struct {
	Path            string           `json:"Path"`
	Format          Format           `json:"Format"`
	Chapters        []Chapter        `json:"Chapters"`
	VideoStreams    []VideoStream    `json:"VideoStreams"`
	AudioStreams    []AudioStream    `json:"AudioStreams"`
	SubtitleStreams []SubtitleStream `json:"SubtitleStreams"`
	Err             error
}

// File checks information about the file format, it's streams and whether it can be used as a media file
// As of now it usses "ffprobe" ran as a separate process to get this information. May be changed to use libav go wrappers in the future
func File(filepath string) Result {
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
		result.Err = fmt.Errorf("probing error: %w", err)

		return result
	}

	if ffprobeResult.ProbeError.Code != 0 {
		result.Err = fmt.Errorf("probing error from ffprobe: %d (%s)", ffprobeResult.ProbeError.Code, ffprobeResult.ProbeError.Message)

		return result
	}

	parsedDuration, err := strconv.ParseFloat(ffprobeResult.Format.Duration, 64)
	if err != nil {
		result.Err = fmt.Errorf("could not parse duration: %w", err)

		return result
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
			result.Err = fmt.Errorf("could not parse start time: %w", err)

			return result
		}

		endTime, err := strconv.ParseFloat(chapter.EndTime, 64)
		if err != nil {
			result.Err = fmt.Errorf("could not parse end time: %w", err)

			return result
		}

		result.Chapters = append(result.Chapters, Chapter{
			ChapterID: strconv.FormatInt(int64(len(result.Chapters)+1), 10),
			EndTime:   endTime,
			StartTime: startTime,
			Title:     chapter.Tags.Title,
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

	return result
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

// IsMediaFile checks whether parsing of file was successful, and whether any video streams are present in the file (audio or subtitles optional)
func (res Result) IsMediaFile() bool {
	if res.Err != nil {
		return false
	}

	return len(res.VideoStreams) != 0 || len(res.AudioStreams) != 0
}
