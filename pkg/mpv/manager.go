package mpv

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"
)

const (
	mpvName           = "mpv"
	idleArg           = "--idle"
	inputIpcServerArg = "--input-ipc-server"

	managerLogPrefix = "mpv.Manager#"
)

type ManagerConfig struct {
	MpvSocketPath           string
	ErrWriter               io.Writer
	OutWriter               io.Writer
	SocketConnectionTimeout time.Duration
	StartMpvInstance        bool
}

// Manager handles dispatching of commands, while exposing MPV command API as a facade.
type Manager struct {
	cd               *commandDispatcher
	errLog           *log.Logger
	mpvCmd           *exec.Cmd
	outLog           *log.Logger
	socketPath       string
	startMpvInstance bool
}

// NewManager starts mpv process and instantiates new command dispatcher, preparing new Manager for use.
func NewManager(cfg ManagerConfig) (*Manager, error) {
	errLog := log.New(cfg.ErrWriter, managerLogPrefix, log.LstdFlags)
	outLog := log.New(cfg.OutWriter, managerLogPrefix, log.LstdFlags)

	cdCfg := commandDispatcherConfig{
		connectionTimeout: cfg.SocketConnectionTimeout,
		errWriter:         errLog.Writer(),
		socketPath:        cfg.MpvSocketPath,
		outWriter:         outLog.Writer(),
	}
	m := &Manager{
		cd:               newCommandDispatcher(cdCfg),
		errLog:           errLog,
		outLog:           outLog,
		socketPath:       cfg.MpvSocketPath,
		startMpvInstance: cfg.StartMpvInstance,
	}

	// TODO: all of the below should be handled be a separate method instead of doing it in NewManager.
	// A proper error management from both mpv process management and command dispatcher
	// command loop managmenet should be used to give client (API server in this case)
	// a control over how Manager should behave in case of any errors.
	// The issue - in case of own mpv process managment any issue related to crash/closure
	// of mpv instance should be autonatic when possible - trying to restart the instance and reconnect.
	// In case of connection to existing unmanaged instance restart is impossible, only reconnect.
	// In case of unmanaged instance there is also an issue that command dispatch loop will crash instead
	// of process management loop, which complicates a little the issue - there needs to be a single point of error handling
	// and a method of a graceful shutdown instead of just loosing connection or endless restart cycle.
	if m.startMpvInstance {
		go m.manageOwnMpvProcess()

		return m, nil
	}

	err := m.cd.Connect()
	if err != nil {
		return m, err // TODO: add some handling of errors on the manager instance
	}

	return m, nil
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
func (m Manager) PlaylistPlayIndex(idx int) error {
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

func (m *Manager) manageOwnMpvProcess() {
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
			return // TODO: add some handling of errors on the manager instance
		}
	}
}
