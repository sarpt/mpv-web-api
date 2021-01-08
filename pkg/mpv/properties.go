package mpv

const (
	// FullscreenProperty is used to inform about state of mpv being in full screen.
	FullscreenProperty = "fullscreen"

	// AudioIDProperty is an option used to change the audio track.
	AudioIDProperty = "aid"

	// SubtitleIDProperty is an option used to change the subtitle track.
	SubtitleIDProperty = "sid"

	// PathProperty is used to inform about path to file currently being played by mpv.
	PathProperty = "path"

	// PlaybackTimeProperty is used for reading and setting current time of playback in seconds.
	PlaybackTimeProperty = "playback-time"

	// PauseProperty is used for pausing or unpausing playback.
	PauseProperty = "pause"

	// LoopFileProperty is used for looping currently played file.
	LoopFileProperty = "loop-file"

	// ABLoopAProperty is used for setting custom looping in the specified timeframe. A is one of the two ends of the time range.
	ABLoopAProperty = "ab-loop-a"

	// ABLoopBProperty is used for setting custom looping in the specified timeframe. B is one of the two ends of the time range.
	ABLoopBProperty = "ab-loop-a"
)

var (
	// ObservableProperties specifies collection of properties that can be observed by 'property-change' event.
	ObservableProperties = []string{
		FullscreenProperty,
		PathProperty,
		PlaybackTimeProperty,
		PauseProperty,
		LoopFileProperty,
		ABLoopAProperty,
		ABLoopBProperty,
		AudioIDProperty,
		SubtitleIDProperty,
	}
)
