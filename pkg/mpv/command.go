package mpv

const (
	loadfileCommand    = "loadfile"
	setPropertyCommand = "set_property"

	fullscreenProperty = "fullscreen"
	fullscreenEnabled  = "yes"
	fullscreenDisabled = "no"
)

// Command holds the name and value of the command to be dispatched
// It's a generic struct that is supposed to be properly constructed by a function
type Command struct {
	name   string
	values []string
}

// Get returns the representation expected by the mpv in the JSON payload
func (cmd Command) Get() []string {
	return append([]string{cmd.name}, cmd.values...)
}

// NewLoadFile returns command for the mpv to load a file under the path
func NewLoadFile(path string) Command {
	return Command{
		name:   loadfileCommand,
		values: []string{path},
	}
}

// NewSetProperty returns command setting the property of the mpv
// Probably not very usefull on its own, rather it's used by other Command creators eg. NewFullscreen
func NewSetProperty(property string, value string) Command {
	return Command{
		name:   setPropertyCommand,
		values: []string{property, value},
	}
}

// NewFullscreen returns the command setting whether the fullscreen should be enabled
func NewFullscreen(enabled bool) Command {
	var fullscreen string = fullscreenDisabled
	if enabled {
		fullscreen = fullscreenEnabled
	}

	return NewSetProperty(fullscreenProperty, fullscreen)
}
