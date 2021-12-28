package mpv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"syscall"
	"time"

	"github.com/sarpt/mpv-web-api/internal/common"
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
	stopServing      chan string
	serveStop        chan error
	errLog           *log.Logger
	mpvCmd           *exec.Cmd
	outLog           *log.Logger
	socketPath       string
	startMpvInstance bool
}

// NewManager starts mpv process and instantiates new command dispatcher, preparing new Manager for use.
func NewManager(cfg ManagerConfig) *Manager {
	errLog := log.New(cfg.ErrWriter, managerLogPrefix, log.LstdFlags)
	outLog := log.New(cfg.OutWriter, managerLogPrefix, log.LstdFlags)

	cdCfg := commandDispatcherConfig{
		connectionTimeout: cfg.SocketConnectionTimeout,
		errWriter:         errLog.Writer(),
		socketPath:        cfg.MpvSocketPath,
		outWriter:         outLog.Writer(),
	}

	return &Manager{
		cd:               newCommandDispatcher(cdCfg),
		errLog:           errLog,
		outLog:           outLog,
		socketPath:       cfg.MpvSocketPath,
		startMpvInstance: cfg.StartMpvInstance,
	}
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

// StopServing instructs Manager to stop serving.
// Stopping a running manager results in command dispatcher being closed,
// and if manager handles an mpv instance, stopping the mpv instance.
// Stopping a not running manager results in an error.
func (m *Manager) StopServing(reason string) error {
	if m.stopServing == nil {
		return fmt.Errorf("stop unsuccessful - manager is not running")
	}

	m.serveStop = make(chan error)
	m.stopServing <- reason

	return <-m.serveStop
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

// LoadList instructs mpv to start playing the playliust from provided file.
// Second argument (append) controls whether playlist should be appended to the current playlist (instead of playlist replacement).
func (m Manager) LoadList(filePath string, append bool) error {
	var loadlistArg string
	if append {
		loadlistArg = AppendValue
	} else {
		loadlistArg = ReplaceValue
	}

	cmd := command{
		name:     loadlistCommand,
		elements: []interface{}{filePath, loadlistArg},
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

// PlaylistClear remediaFiles all entries from playlist.
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

// Serve starts handling requests to and responses from mpv.
// If necessary, Serve also spawns and handles mpv process lifetime.
func (m *Manager) Serve() error {
	mpvErrors := make(chan error)
	cdErrors := make(chan error)

	m.stopServing = make(chan string)
	defer func() { m.stopServing = nil }()

	serveCtx, serveCancel := context.WithCancel(context.Background())
	go common.RestartWithContext(serveCtx, m.manageOwnMpvProcess, func() { m.outLog.Println("restarting mpv process...") }, mpvErrors)
	go common.RestartWithContext(serveCtx, m.serveCommandDispatcher, func() { m.outLog.Println("restarting command dispatcher...") }, cdErrors)

	select {
	case reason := <-m.stopServing:
		m.outLog.Printf("stopping the manager, reason: %s", reason)
	case err := <-mpvErrors:
		if err != nil {
			m.errLog.Printf("stopping the manager due to mpv error: %s", err)
		}
	case err := <-cdErrors:
		if err != nil {
			m.errLog.Printf("stopping the manager due to command dispatcher error: %s", err)
		}
	}

	serveCancel()
	if m.mpvCmd != nil {
		err := m.mpvCmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			m.errLog.Printf("SIGTERM to mpv resulted with error: %s", err)
		}
	}

	err := m.cd.Close()
	m.serveStop <- err

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

// StopPlayback instructs mpv to stop the playback without quitting.
func (m Manager) StopPlayback() error {
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

func (m *Manager) manageOwnMpvProcess() error {
	err := m.startMpv()
	if err != nil {
		return fmt.Errorf("could not start mpv process due to error: %w", err)
	}
	m.outLog.Println("mpv process started")

	m.outLog.Println("watching for mpv process exit...")

	err = m.mpvCmd.Wait()
	if err != nil {
		return fmt.Errorf("mpv process finished with error: %w", err)
	}

	m.outLog.Println("mpv process finished successfully (closed by user)")
	return nil
}

func (m *Manager) serveCommandDispatcher() error {
	m.outLog.Println("connecting command dispatcher...")

	err := m.cd.Connect()
	if errors.Is(err, ErrCheckConnectionFailure) {
		return nil
	} else if err != nil {
		return err
	}

	err = m.cd.Serve()
	if err != nil {
		return err
	}

	m.cd.Close()
	return nil
}
