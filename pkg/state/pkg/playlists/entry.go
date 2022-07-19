package playlists

type Entry struct {
	Path              string  `json:"Path"`
	PlaybackTimestamp float64 `json:"PlaybackTimestamp"`
	AudioID           string  `json:"AudioId"`
	SubtitleID        string  `json:"SubtitleId"`
}
