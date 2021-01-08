package mpv

// command holds the name and value of the command to be dispatched.
// It's a generic struct that is supposed to be properly constructed by a function.
type command struct {
	name     string
	elements []interface{} // i'm not conviced about this interface{} thing. Probably should do a reflection or type assertion whether it's an int or a string
}

// JSONIPCFormat returns the representation expected by the mpv in the JSON payload.
func (cmd command) JSONIPCFormat() []interface{} {
	return append([]interface{}{cmd.name}, cmd.elements...)
}
