package mpv

// Manager handles dispatching of commands, while exposing a facade.
// TODO: Managed should also handle mpv binary detection and lifetime handling
type Manager struct {
	cd *CommandDispatcher
}

// NewManager instantiates new command dispatcher, preparing new Manager to be used
// TODO: add mpv binary detection and mpv process startup
func NewManager(mpvSocketPath string) (Manager, error) {
	cd, err := NewCommandDispatcher(mpvSocketPath)
	if err != nil {
		return Manager{}, err
	}

	return Manager{
		cd,
	}, nil
}

// ChangeFullscreen instructs mpv to change the fullscreen state
func (m Manager) ChangeFullscreen(enabled bool) error {
	_, err := m.cd.Request(NewFullscreen(enabled))
	return err
}

// LoadFile instructs mpv to start playing the file from provided filepath
func (m Manager) LoadFile(filePath string) error {
	_, err := m.cd.Request(NewLoadFile(filePath))
	return err
}

// ChangeSubtitle instructs mpv to change the subtitle to the one with specified id
func (m Manager) ChangeSubtitle(subtitleID string) error {
	_, err := m.cd.Request(NewSetSubtitleID(subtitleID))
	return err
}

// ChangeAudio instructs mpv to change the audio to the one with specified id
func (m Manager) ChangeAudio(audioID string) error {
	_, err := m.cd.Request(NewSetAudioID(audioID))
	return err
}

// ObserveProperty instructs mpv to listen on property changes and send those changes on the out channel
func (m Manager) ObserveProperty(propertyName string, out chan<- ObserveResponse) (int, error) {
	return m.cd.ObserveProperty(propertyName, out)
}

// Close cleans up manager's resources
func (m Manager) Close() {
	m.cd.Close()
}
