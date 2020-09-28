package mpv

import (
	"fmt"
	"io"
	"log"
	"os/exec"
)

const (
	mpvName           = "mpv"
	idleArg           = "--idle"
	inputIpcServerArg = "--input-ipc-server"

	managerLogPrefix = "mpv.Manager#"
)

// Manager handles dispatching of commands, while exposing a facade.
type Manager struct {
	mpvCmd     *exec.Cmd
	socketPath string
	cd         *CommandDispatcher
	outLog     *log.Logger
	errLog     *log.Logger
}

// NewManager starts mpv process and instantiates new command dispatcher, preparing new Manager for use
func NewManager(mpvSocketPath string, outWriter io.Writer, errWriter io.Writer) *Manager {
	errLog := log.New(errWriter, managerLogPrefix, log.LstdFlags)

	m := &Manager{
		socketPath: mpvSocketPath,
		outLog:     log.New(outWriter, managerLogPrefix, log.LstdFlags),
		errLog:     errLog,
		cd:         NewCommandDispatcher(mpvSocketPath, errLog.Writer()),
	}

	go m.watchMpvProcess()

	return m
}

func (m *Manager) watchMpvProcess() {
	var err error
	for {
		if m.mpvCmd != nil {
			m.outLog.Println("watching for mpv process exit...")

			err = m.mpvCmd.Wait()
			if err != nil {
				m.errLog.Printf("mpv process finished with error: %s\n", err)
			} else {
				m.outLog.Println("mpv process finished successfully")
			}

			m.cd.Close()
			m.outLog.Println("restarting mpv process and command dispatcher...")
		}

		err = m.startMpv()
		if err != nil {
			m.errLog.Printf("could not start mpv process due to error: %s\n", err)
			return // TODO: add some handling of errors on the manager instance
		}
		m.outLog.Println("mpv process started")

		err = m.cd.Connect()
		if err != nil {
			m.errLog.Printf("command dispatcher could not connect to socket due to error: %s\n", err)
			return // TODO: add some handling of errors on the manager instance
		}
		m.outLog.Println("command dispatcher connected to socket")
	}
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

// ChangePause instructs mpv to change the pause state
func (m Manager) ChangePause(paused bool) error {
	_, err := m.cd.Request(NewSetPause(paused))
	return err
}

// LoopFile instructs mpv to change the loop state
func (m Manager) LoopFile(loop bool) error {
	_, err := m.cd.Request(NewSetLoopFile(loop))
	return err
}

// SubscribeToProperty instructs mpv to listen on property changes and send those changes on the out channel
func (m Manager) SubscribeToProperty(propertyName string, out chan<- ObservePropertyResponse) (int, error) {
	return m.cd.SubscribeToProperty(propertyName, out)
}

// Close cleans up manager's resources
func (m Manager) Close() {
	m.cd.Close()
}
