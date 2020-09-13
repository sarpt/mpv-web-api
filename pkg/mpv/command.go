package mpv

const (
	loadfileCommand        = "loadfile"
	setPropertyCommand     = "set_property"
	observePropertyCommand = "observe_property_string"

	propertyChangeEvent = "property-change"

	// FullscreenProperty is used to inform about state of mpv being in full screen
	FullscreenProperty = "fullscreen"
	// FullscreenEnabled is a value of fullscreen property indicating that mpv is in full screen
	FullscreenEnabled = "yes"
	// FullscreenDisabled is a value of fullscreen property indicating that mpv is not in full screen
	FullscreenDisabled = "no"

	// AudioID is an option used to change the audio track
	AudioID = "aid"

	// SubtitleID is an option used to change the subtitle track
	SubtitleID = "sid"

	// PathProperty is used to inform about path to file currently being played by mpv
	PathProperty = "path"

	// PlaybackTimeProperty is used for reading and setting current time of playback in seconds
	PlaybackTimeProperty = "playback-time"
)

var (
	// ObservableProperties specifies collection of properties that can be observed by 'property-change' event
	ObservableProperties = []string{
		FullscreenProperty,
		PathProperty,
		PlaybackTimeProperty,
	}
)

// Command holds the name and value of the command to be dispatched.
// It's a generic struct that is supposed to be properly constructed by a function.
type Command struct {
	name   string
	values []interface{} // i'm not conviced about this interface{} thing. Probably should do a reflection or type assertion whether it's an int or a string
}

// Get returns the representation expected by the mpv in the JSON payload
func (cmd Command) Get() []interface{} {
	return append([]interface{}{cmd.name}, cmd.values...)
}

// NewLoadFile returns command for the mpv to load a file under the path
func NewLoadFile(path string) Command {
	return Command{
		name:   loadfileCommand,
		values: []interface{}{path},
	}
}

// NewSetProperty returns command setting the property of the mpv.
// Probably not very useful on its own, rather it's used by other Command creators eg. NewFullscreen.
func NewSetProperty(property string, value string) Command {
	return Command{
		name:   setPropertyCommand,
		values: []interface{}{property, value},
	}
}

// NewFullscreen returns the command setting whether the fullscreen should be enabled
func NewFullscreen(enabled bool) Command {
	var fullscreen string = FullscreenDisabled
	if enabled {
		fullscreen = FullscreenEnabled
	}

	return NewSetProperty(FullscreenProperty, fullscreen)
}

// NewSetAudioID returns the command changing the audio track to the specidifed audio id
func NewSetAudioID(aid string) Command {
	return NewSetProperty(AudioID, aid)
}

// NewSetSubtitleID return the command changing the subtitle track to she specified subtitle id
func NewSetSubtitleID(sid string) Command {
	return NewSetProperty(SubtitleID, sid)
}

// NewObserveProperty returns the command that instructs mpv to report as event changes for the specific mpv state property
func NewObserveProperty(id int, propertyName string) Command {
	return Command{
		name:   observePropertyCommand,
		values: []interface{}{id, propertyName},
	}
}
