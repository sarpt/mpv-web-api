package probe

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

type tags struct {
	Language string `json:"language"`
}

type stream struct {
	Index     int    `json:"index"`
	CodecType string `json:"codec_type"`
	Tags      tags   `json:"tags"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Channels  int    `json:"channels"`
}

type probeError struct {
	Code    int    `json:"code"`
	Message string `json:"string"`
}

type ffprobeResult struct {
	Streams    []stream   `json:"streams"`
	ProbeError probeError `json:"error"`
}
