package probe

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

type format struct {
	Name     string `json:"format_name"`
	LongName string `json:"format_long_name"`
	Duration string `json:"duration"`
}

type probeError struct {
	Code    int    `json:"code"`
	Message string `json:"string"`
}

type ffprobeResult struct {
	Streams    []stream   `json:"streams"`
	ProbeError probeError `json:"error"`
	Format     format     `json:"format"`
}
