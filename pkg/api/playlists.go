package api

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/sarpt/mpv-web-api/pkg/state"
)

var (
	ErrJSONFileNotAPlaylistFile = errors.New("a JSON file is not a valid playlist file - 'mpvWebApiPlaylist' either not specified or false")
)

const (
	defaultPlaylistName string = "default"
)

type PlaylistFile struct {
	CurrentEntryIdx            int                   `json:"CurrentEntryIdx"`
	DirectoryContentsAsEntries bool                  `json:"DirectoryContentsAsEntries"`
	Entries                    []state.PlaylistEntry `json:"Entries"`
	MpvWebApiPlaylist          bool                  `json:"MpvWebApiPlaylist"`
	Name                       string                `json:"Name"`
	Description                string                `json:"Description"`
}

func (s *Server) createDefaultPlaylist() (string, error) {
	defaultPlaylistCfg := state.PlaylistConfig{
		Name: defaultPlaylistName,
	}

	return s.playlists.AddPlaylist(state.NewPlaylist(defaultPlaylistCfg))
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

	playlistCfg := state.PlaylistConfig{
		Description:                playlistFile.Description,
		DirectoryContentsAsEntries: playlistFile.DirectoryContentsAsEntries,
		Entries:                    playlistFile.Entries,
		Name:                       playlistFile.Name,
	}

	uuid, err := s.playlists.AddPlaylist(state.NewPlaylist(playlistCfg))
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
