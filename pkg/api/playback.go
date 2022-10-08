package api

import (
	"fmt"

	playbackTriggers "github.com/sarpt/mpv-web-api/pkg/api/internal/playback_triggers"
)

func (s *Server) ChangeAudio(audioId string) error {
	return s.mpvManager.ChangeAudio(audioId)
}

func (s *Server) ChangeChapter(idx int64) error {
	return s.mpvManager.ChangeChapter(idx)
}

func (s *Server) ChangeChaptersOrder(chapters []int64, force bool) error {
	mediaFilePath := s.statesRepository.Playback().MediaFilePath()
	notifications := make(chan playbackTriggers.ChapterManagerNotification)
	playbackTrigger, err := playbackTriggers.NewChaptersManager(mediaFilePath, chapters, s, notifications)
	if err != nil {
		return fmt.Errorf("could not change chapters order: %s", err)
	}

	if force {
		chapter := chapters[0]
		s.mpvManager.ChangeChapter(chapter)
	}

	unsub := s.addPlaybackTrigger(playbackTrigger)
	go func() {
		for {
			notif, more := <-notifications
			if notif == playbackTriggers.MediaFileChangedDuringIteration || !more {
				unsub()
				return
			}

			if notif != playbackTriggers.ChaptersIterationDone {
				continue
			}

			unsub()

			fileLoops := s.statesRepository.Playback().LoopFile()
			if fileLoops {
				s.ChangeChaptersOrder(chapters, true)
			}

			return
		}
	}()

	return nil
}

func (s *Server) WaitUntilMediaFileByUuid(uuid string) error {
	currentMediaFilePath := s.statesRepository.Playback().MediaFilePath()
	currentMediaFile, err := s.statesRepository.MediaFiles().ByPath(currentMediaFilePath)
	if err != nil {
		return fmt.Errorf("could not determine currently playing file: %s", err)
	}

	if currentMediaFile.Uuid() == uuid {
		return nil
	}

	targetMediaFile, err := s.statesRepository.MediaFiles().ByUuid(uuid)
	if err != nil {
		return fmt.Errorf("provided uuid does not match known media file: %s", err)
	}

	return s.WaitUntilMediaFileByPath(targetMediaFile.Path())
}

func (s *Server) WaitUntilMediaFileByPath(mediaFilePath string) error {
	if s.statesRepository.Playback().MediaFilePath() == mediaFilePath {
		return nil
	}

	notifications := make(chan playbackTriggers.MediaFileChangeNotification)
	mediaFiletrigger, err := playbackTriggers.NewMediaFileChange(mediaFilePath, notifications)
	if err != nil {
		return fmt.Errorf("could not change chapters order: %s", err)
	}

	unsub := s.addPlaybackTrigger(mediaFiletrigger)
	go func() {
		for {
			notif, more := <-notifications
			if notif == playbackTriggers.ChangedMediaFileMatches || !more {
				unsub()
				return
			}
		}
	}()

	return nil
}

func (s *Server) ChangeFullscreen(fullscreen bool) error {
	return s.mpvManager.ChangeFullscreen(fullscreen)
}

func (s *Server) ChangePause(paused bool) error {
	return s.mpvManager.ChangePause(paused)
}

func (s *Server) ChangeSubtitle(subtitleID string) error {
	return s.mpvManager.ChangeSubtitle(subtitleID)
}

func (s *Server) LoadFile(filePath string, append bool) error {
	return s.mpvManager.LoadFile(filePath, append)
}

func (s *Server) LoadFileByUuid(uuid string, append bool) error {
	mediaFile, err := s.statesRepository.MediaFiles().ByUuid(uuid)
	if err != nil {
		return err
	}

	filePath := mediaFile.Path()

	return s.mpvManager.LoadFile(filePath, append)
}

func (s *Server) LoopFile(looped bool) error {
	return s.mpvManager.LoopFile(looped)
}

func (s *Server) PlaylistPlayIndex(idx int) error {
	return s.mpvManager.PlaylistPlayIndex(idx)
}

func (s *Server) StopPlayback() error {
	return s.mpvManager.StopPlayback()
}
