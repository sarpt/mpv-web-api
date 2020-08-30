package mpv

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	mpvName           = "mpv"
	idleArg           = "--idle"
	inputIpcServerArg = "--input-ipc-server"
)

// Manager handles dispatching of commands, while exposing a facade.
type Manager struct {
	mpvCmd     *exec.Cmd
	socketPath string
	cd         *CommandDispatcher
}

// NewManager starts mpv process and instantiates new command dispatcher, preparing new Manager for use
func NewManager(mpvSocketPath string) *Manager {
	m := &Manager{
		socketPath: mpvSocketPath,
	}

	go func(m *Manager) {
		var err error
		for {
			if m.mpvCmd != nil {
				fmt.Fprintf(os.Stdout, "watching for mpv process exit...\n")

				err = m.mpvCmd.Wait()
				if err != nil {
					fmt.Fprintf(os.Stderr, "mpv process finished with error: %s\n", err)
				} else {
					fmt.Fprintf(os.Stdout, "mpv process finished successfully\n")
				}

				m.cd.Close()
				fmt.Fprintf(os.Stdout, "restarting mpv process and command dispatcher...\n")
			}

			err = m.startMpv()
			if err != nil {
				fmt.Fprintf(os.Stdout, "could not start mpv process due to error: %s\n", err)
				return // TODO: add some handling of errors on the manager instance
			}
			fmt.Fprintf(os.Stdout, "mpv process started\n")

			err = m.startCommandDispatcher()
			if err != nil {
				fmt.Fprintf(os.Stdout, "could not start command dispatcher due to error: %s\n", err)
				return // TODO: add some handling of errors on the manager instance
			}
			fmt.Fprintf(os.Stdout, "command dispatcher started\n")
		}
	}(m)

	return m
}

func (m *Manager) startMpv() error {
	cmd := exec.Command(mpvName, idleArg, fmt.Sprintf("%s=%s", inputIpcServerArg, m.socketPath))
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("could not start mpv process: %w", err)
	}

	m.mpvCmd = cmd
	return nil
}

func (m *Manager) startCommandDispatcher() error {
	cd, err := NewCommandDispatcher(m.socketPath)
	if err != nil {
		return err
	}

	m.cd = cd
	return nil
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
