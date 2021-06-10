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

// Manager handles dispatching of commands, while exposing MPV command API as a facade.
type Manager struct {
	mpvCmd     *exec.Cmd
	socketPath string
	cd         *commandDispatcher
	outLog     *log.Logger
	errLog     *log.Logger
}

// NewManager starts mpv process and instantiates new command dispatcher, preparing new Manager for use.
func NewManager(mpvSocketPath string, outWriter io.Writer, errWriter io.Writer) *Manager {
	errLog := log.New(errWriter, managerLogPrefix, log.LstdFlags)

	m := &Manager{
		socketPath: mpvSocketPath,
		outLog:     log.New(outWriter, managerLogPrefix, log.LstdFlags),
		errLog:     errLog,
		cd:         newCommandDispatcher(mpvSocketPath, errLog.Writer()),
	}

	go m.watchMpvProcess()

	return m
}

// ChangeFullscreen instructs mpv to change the fullscreen state.
// Enabled argument specifies whether fullscrren should be enabled or disabled.
func (m Manager) ChangeFullscreen(enabled bool) error {
	var fullscreen string = NoValue
	if enabled {
		fullscreen = YesValue
	}

	_, err := m.SetProperty(FullscreenProperty, fullscreen)
	return err
}

// ChangeSubtitle instructs mpv to change the subtitle to the one with specified id.
func (m Manager) ChangeSubtitle(subtitleID string) error {
	_, err := m.SetProperty(SubtitleIDProperty, subtitleID)

	return err
}

// ChangeAudio instructs mpv to change the audio to the one with specified id.
func (m Manager) ChangeAudio(audioID string) error {
	_, err := m.SetProperty(AudioIDProperty, audioID)

	return err
}

// ChangeChapter instructs mpv to change the chapter to the one with specified idx.
func (m Manager) ChangeChapter(idx int64) error {
	_, err := m.SetProperty(ChapterProperty, idx)

	return err
}

// ChangePause instructs mpv to change the pause state.
// Paused argument specifies whether playback should be paused or unpaused.
func (m Manager) ChangePause(paused bool) error {
	_, err := m.SetProperty(PauseProperty, paused)

	return err
}

// Close cleans up manager's resources.
func (m Manager) Close() {
	m.cd.Close()
}

// LoadFile instructs mpv to start playing the file from provided filepath.
// Second argument (append) controls whether filepath playback should be appended to the current playlist (instead of playback replacement).
func (m Manager) LoadFile(filePath string, append bool) error {
	var loadFileArg string
	if append {
		loadFileArg = AppendValue
	} else {
		loadFileArg = ReplaceValue
	}

	cmd := command{
		name:     loadfileCommand,
		elements: []interface{}{filePath, loadFileArg},
	}
	_, err := m.cd.Request(cmd)

	return err
}

// LoopFile instructs mpv to change the loop state.
func (m Manager) LoopFile(looped bool) error {
	var loopedVal string = NoValue
	if looped {
		loopedVal = InfValue
	}

	_, err := m.SetProperty(LoopFileProperty, loopedVal)

	return err
}

// PlaylistClear removies all entries from playlist.
func (m Manager) PlaylistClear() error {
	cmd := command{
		name:     playlistClearCommand,
		elements: []interface{}{},
	}
	_, err := m.cd.Request(cmd)

	return err
}

// PlaylistNext changes playback to the next entry in the playlist.
// Force set to true instructs mpv to stop the playback when currently playing item is last in the playlist.
func (m Manager) PlaylistNext(force bool) error {
	var forceVal string = WeakValue
	if force {
		forceVal = ForceValue
	}

	cmd := command{
		name:     playlistNextCommand,
		elements: []interface{}{forceVal},
	}
	_, err := m.cd.Request(cmd)

	return err
}

// PlaylistPlayIndex changes playback to the playlist item under the provided index.
func (m Manager) PlaylistPlayIndex(idx uint) error {
	cmd := command{
		name:     playlistPlayIdxCommand,
		elements: []interface{}{idx},
	}
	_, err := m.cd.Request(cmd)

	return err
}

// PlaylistPrev changes playback to the previous entry in the playlist.
// Force set to true instructs mpv to stop the playback when currently playing item is first in the playlist.
func (m Manager) PlaylistPrev(force bool) error {
	var forceVal string = WeakValue
	if force {
		forceVal = ForceValue
	}

	cmd := command{
		name:     playlistPrevCommand,
		elements: []interface{}{forceVal},
	}
	_, err := m.cd.Request(cmd)

	return err
}

// PlaylistRestartCurrent starts playing current playlist item from the beginning.
func (m Manager) PlaylistRestartCurrent() error {
	cmd := command{
		name:     playlistPlayIdxCommand,
		elements: []interface{}{CurrentValue},
	}
	_, err := m.cd.Request(cmd)

	return err
}

// PlaylistRemove removes element under the index from the playlist.
func (m Manager) PlaylistRemove(idx uint) error {
	cmd := command{
		name:     playlistRemoveCommand,
		elements: []interface{}{idx},
	}
	_, err := m.cd.Request(cmd)

	return err
}

// PlaylistMove moves element in the playlist from "fromIdx" to index "toIdx".
func (m Manager) PlaylistMove(fromIdx uint, toIdx uint) error {
	cmd := command{
		name:     playlistMoveCommand,
		elements: []interface{}{fromIdx, toIdx},
	}
	_, err := m.cd.Request(cmd)

	return err
}

// SetProperty sets the value of a property.
// Value is of any type since various mpv commands expect different types of values.
// TODO: rewrite to generics when those are out.
func (m Manager) SetProperty(property string, value interface{}) (Response, error) {
	cmd := command{
		name:     setPropertyCommand,
		elements: []interface{}{property, value},
	}

	return m.cd.Request(cmd)
}

// Stop instructs mpv to stop the playback without quitting.
func (m Manager) Stop() error {
	cmd := command{
		name:     stopCommand,
		elements: []interface{}{},
	}
	_, err := m.cd.Request(cmd)

	return err
}

// SubscribeToProperty instructs mpv to listen on property changes and send those changes on the out channel.
func (m Manager) SubscribeToProperty(propertyName string, out chan<- ObservePropertyResponse) (int, error) {
	return m.cd.SubscribeToProperty(propertyName, out)
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
