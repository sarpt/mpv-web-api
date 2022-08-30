package api

import "fmt"

func (s *Server) ChangeAudio(audioId string) error {
	return s.mpvManager.ChangeAudio(audioId)
}

func (s *Server) ChangeChapter(idx int64) error {
	return s.mpvManager.ChangeChapter(idx)
}

func (s *Server) ChangeChaptersOrder(chapters []int64, force bool) error {
	playbackTrigger, err := newChaptersManagerPlaybackTrigger(chapters)
	if err != nil {
		return fmt.Errorf("could not change chapters order: %s", err)
	}

	if force {
		chapter := chapters[0]
		s.mpvManager.ChangeChapter(chapter)
	}

	s.addPlaybackTrigger(s.statesRepository.Playback().MediaFilePath(), playbackTrigger)
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

func (s *Server) LoopFile(looped bool) error {
	return s.mpvManager.LoopFile(looped)
}

func (s *Server) PlaylistPlayIndex(idx int) error {
	return s.mpvManager.PlaylistPlayIndex(idx)
}

func (s *Server) StopPlayback() error {
	return s.mpvManager.StopPlayback()
}
