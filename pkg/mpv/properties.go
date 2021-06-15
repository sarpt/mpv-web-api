package mpv

const (
	// ABLoopAProperty is used for setting custom looping in the specified timeframe. A is one of the two ends of the time range.
	ABLoopAProperty = "ab-loop-a"

	// ABLoopBProperty is used for setting custom looping in the specified timeframe. B is one of the two ends of the time range.
	ABLoopBProperty = "ab-loop-b"

	// AudioIDProperty is an option used to change the audio track.
	AudioIDProperty = "aid"

	// ChapterProperty is used for setting/reading currently played chapter.
	ChapterProperty = "chapter"

	// FullscreenProperty is used to inform about state of mpv being in full screen.
	FullscreenProperty = "fullscreen"

	// LoopFileProperty is used for looping currently played file.
	LoopFileProperty = "loop-file"

	// PathProperty is used to inform about path to file currently being played by mpv.
	PathProperty = "path"

	// PauseProperty is used for pausing or unpausing playback.
	PauseProperty = "pause"

	// PlaybackTimeProperty is used for reading and setting current time of playback in seconds.
	PlaybackTimeProperty = "playback-time"

	// PlaylistProperty is used for reading state of the playlist.
	PlaylistProperty = "playlist"

	// PlaylistPlayingPosProperty is used for reading currently playing position of playlist.
	PlaylistPlayingPosProperty = "playlist-playing-pos"

	// SubtitleIDProperty is an option used to change the subtitle track.
	SubtitleIDProperty = "sid"
)

var (
	// ObservableProperties specifies collection of properties that can be observed by 'property-change' event.
	ObservableProperties = []string{
		ABLoopAProperty,
		ABLoopBProperty,
		AudioIDProperty,
		ChapterProperty,
		FullscreenProperty,
		LoopFileProperty,
		PathProperty,
		PauseProperty,
		PlaylistProperty,
		PlaybackTimeProperty,
		PlaylistPlayingPosProperty,
		SubtitleIDProperty,
	}
)
