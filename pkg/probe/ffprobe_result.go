package probe

type chapter struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Tags      tags   `json:"tags"`
}

type tags struct {
	Language string `json:"language"`
	Title    string `json:"title"`
	Filename string `json:"filename"`
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
	Tags     tags   `json:"tags"`
}

type probeError struct {
	Code    int    `json:"code"`
	Message string `json:"string"`
}

type ffprobeResult struct {
	Chapters   []chapter  `json:"chapters"`
	Streams    []stream   `json:"streams"`
	ProbeError probeError `json:"error"`
	Format     format     `json:"format"`
}
