package api

import (
	"fmt"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/directories"
)

func (s *Server) addFsEventTarget(path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		mediaFile, err := s.probeFile(path)
		if err != nil {
			return err
		}

		s.mediaFiles.Add(mediaFile)

		return nil
	}

	parentDir, err := s.directories.ParentByPath(path)
	if err != nil {
		return fmt.Errorf("could not handle directory fs event for '%s' due to missing parent: %w", path, err)
	}

	if !parentDir.Recursive {
		return nil
	}

	dir := directories.Directory{
		Path:      path,
		Recursive: true,
		Watched:   true,
	}

	return s.AddDirectory(dir)
}

func (s *Server) removeFsEventTarget(path string) error {
	if s.mediaFiles.Exists(path) {
		s.outLog.Printf("removing media file '%s'\n", path)
		_, err := s.mediaFiles.Take(path)

		return err
	}

	if s.directories.Exists(path) {
		_, err := s.TakeDirectory(path)

		return err
	}

	return nil
}

func (s *Server) handleFsEvent(event fsnotify.Event) error {
	if shouldRemoveFsEventTarget(event.Op) {
		return s.removeFsEventTarget(event.Name)
	}

	if shouldAddFsEventTarget(event.Op) {
		return s.addFsEventTarget(event.Name)
	}

	return nil
}

func (s *Server) watchForFsChanges() {
	go func() {
		defer s.fsWatcher.Close()

		for {
			select {
			case event, ok := <-s.fsWatcher.Events:
				if !ok {
					return
				}

				err := s.handleFsEvent(event)
				if err != nil {
					s.errLog.Printf("could not handle event '%s' due to an error: %s\n", event, err)
				}
			case err, ok := <-s.fsWatcher.Errors:
				if !ok {
					return
				}

				s.outLog.Printf("fs watcher returned an error: %s\n", err)
			}
		}
	}()
}

func shouldAddFsEventTarget(op fsnotify.Op) bool {
	return op&fsnotify.Create == fsnotify.Create
}

func shouldRemoveFsEventTarget(op fsnotify.Op) bool {
	return op&(fsnotify.Rename|fsnotify.Remove) != 0
}
