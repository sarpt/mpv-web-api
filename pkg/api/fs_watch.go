package api

import "github.com/fsnotify/fsnotify"

func (s *Server) handleFsEvent(event fsnotify.Event) error {
	if shouldRemoveMediaPath(event.Op) {
		s.outLog.Printf("removing media file '%s'\n", event.Name)
		_, err := s.mediaFiles.Take(event.Name)

		return err
	}

	if shouldProbeMediaPath(event.Op) {
		go s.probeFile(event.Name) // TODO: this does not necesarilly be run as a goroutine, change in the next commit with rest of probing refactor

		return nil
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

func shouldProbeMediaPath(op fsnotify.Op) bool {
	return op&fsnotify.Create == fsnotify.Create
}

func shouldRemoveMediaPath(op fsnotify.Op) bool {
	return op&(fsnotify.Rename|fsnotify.Remove) != 0
}
