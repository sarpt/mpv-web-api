package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playback"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/playlists"
)

var (
	ErrJSONFileNotAPlaylistFile = errors.New("a JSON file is not a valid playlist file - 'MpvWebApiPlaylist' either not specified or false")
)

const (
	defaultPlaylistName  string = "default"
	tempPlaylistFilename string = "mwa_playlist"
)

type PlaylistFile struct {
	CurrentEntryIdx            int               `json:"CurrentEntryIdx"`
	DirectoryContentsAsEntries bool              `json:"DirectoryContentsAsEntries"`
	Entries                    []playlists.Entry `json:"Entries"`
	MpvWebApiPlaylist          bool              `json:"MpvWebApiPlaylist"`
	Name                       string            `json:"Name"`
	Description                string            `json:"Description"`
}

func (s *Server) DefaultPlaylistSelected() bool {
	return s.statesRepository.Playback().PlaylistUUID() == s.defaultPlaylistUUID
}

// LoadPlaylist instructs mpv to add entries of a playlist to the mpv internal playlist.
// UUID is a key of a playlist that is unique in the scope of a server's instance.
// Append specifies whether the playlist should be added to the end of the currently played playlist.
// When append is false, the new playlist overwrites current playlist and starts playing it immediately.
// When append is true, a default playlist will be selected and updated with entries from both previously
// selected playlist and a new appended one (mpv will emit change to playlist property which will set the entries
// on the default playlist).
func (s *Server) LoadPlaylist(uuid string, append bool) error {
	playlist, err := s.statesRepository.Playlists().ByUUID(uuid)
	if err != nil {
		return err
	}

	// TODO: if default playlist selected and conditions for default playlist saving met, save the current playist into a file before modyfing

	filename := fmt.Sprintf("%s_%s", tempPlaylistFilename, uuid)
	filepath := filepath.Join(os.TempDir(), filename)
	err = s.createTempPlaylistFile(filepath, playlist.All())
	if err != nil {
		return err
	}

	if append {
		s.statesRepository.Playback().SelectPlaylist(s.defaultPlaylistUUID)
	} else {
		s.statesRepository.Playback().SelectPlaylist(uuid)
	}

	err = s.mpvManager.LoadList(filepath, append)
	if err != nil {
		return err
	}

	if !append && playlist.CurrentEntryIdx() != 0 {
		s.mpvManager.PlaylistPlayIndex(playlist.CurrentEntryIdx())
	}

	return nil
}

func (s *Server) createTempPlaylistFile(filename string, entries []playlists.Entry) error {
	fileData := []byte{}
	for _, entry := range entries {
		fileData = append(fileData, []byte(fmt.Sprintln(entry.Path))...)
	}

	return os.WriteFile(filename, fileData, os.ModePerm)
}

func (s *Server) createTempPlaylist() (string, error) {
	defaultPlaylistCfg := playlists.Config{
		Name:   defaultPlaylistName,
		Origin: playlists.TempOrigin,
	}

	return s.statesRepository.Playlists().AddPlaylist(playlists.NewPlaylist(defaultPlaylistCfg))
}

func (s *Server) hasPlaylistFilePrefix(path string) bool {
	filename := filepath.Base(path)

	for _, prefix := range s.playlistFilesPrefixes {
		if strings.HasPrefix(filename, prefix) {
			return true
		}
	}

	return false
}

func (s *Server) handlePlaylistFile(path string) (string, error) {
	playlistFile, err := s.readPlaylistFile(path)
	if err != nil {
		return "", err
	}

	playlistCfg := playlists.Config{
		CurrentEntryIdx:            playlistFile.CurrentEntryIdx,
		Description:                playlistFile.Description,
		DirectoryContentsAsEntries: playlistFile.DirectoryContentsAsEntries,
		Entries:                    playlistFile.Entries,
		Name:                       playlistFile.Name,
		Origin:                     playlists.ExternalOrigin,
		Path:                       path,
	}

	uuid, err := s.statesRepository.Playlists().AddPlaylist(playlists.NewPlaylist(playlistCfg))
	if err == nil {
		s.outLog.Printf("added playlist '%s' at path '%s'", playlistFile.Name, path)
	}

	return uuid, err
}

func (s *Server) readPlaylistFile(path string) (PlaylistFile, error) {
	var Playlist PlaylistFile

	filePayload, err := os.ReadFile(path)
	if err != nil {
		return Playlist, err
	}

	err = json.Unmarshal(filePayload, &Playlist)
	if err != nil {
		return Playlist, err
	}

	if !Playlist.MpvWebApiPlaylist {
		return Playlist, ErrJSONFileNotAPlaylistFile
	}

	return Playlist, nil
}

func (s *Server) handlePlaylistRelatedPlaybackChanges(change playback.Change) {
	if change.ChangeVariant != playback.PlaylistUnloadChange && change.ChangeVariant != playback.PlaylistCurrentIdxChange {
		return
	}

	if change.ChangeVariant == playback.PlaylistCurrentIdxChange {
		uuid := s.statesRepository.Playback().PlaylistUUID()
		if uuid == s.defaultPlaylistUUID {
			return
		}

		err := s.statesRepository.Playlists().SetPlaylistCurrentEntryIdx(uuid, s.statesRepository.Playback().PlaylistCurrentIdx())
		if err != nil {
			s.errLog.Println(err)
		}
	} else if change.ChangeVariant == playback.PlaylistUnloadChange {
		uuid, ok := change.Value.(string)
		if !ok || uuid == s.defaultPlaylistUUID {
			return
		}

		err := s.savePlaylist(uuid)
		if err != nil {
			s.errLog.Println(err)
		}
	}
}

func (s *Server) saveCurrentPlaylist() error {
	uuid := s.statesRepository.Playback().PlaylistUUID()
	if s.DefaultPlaylistSelected() { // TODO: default/unnamed playlist could be saved to a home directory to be restored when mpv-web-api is ran again, to be considered
		return fmt.Errorf("save of current playlist unsuccessful - cannot save current playlist")
	}

	return s.savePlaylist(uuid)
}

func (s *Server) savePlaylist(uuid string) error {
	playlist, err := s.statesRepository.Playlists().ByUUID(uuid)
	if err != nil {
		return err
	}

	playlistFile := &PlaylistFile{
		CurrentEntryIdx:            s.statesRepository.Playback().PlaylistCurrentIdx(),
		DirectoryContentsAsEntries: playlist.DirectoryContentsAsEntries(),
		Entries:                    playlist.All(),
		MpvWebApiPlaylist:          true,
		Name:                       playlist.Name(),
		Description:                playlist.Description(),
	}

	filePayload, err := json.Marshal(playlistFile)
	if err != nil {
		return err
	}

	return os.WriteFile(playlist.Path(), filePayload, 0)
}
