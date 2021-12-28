package common

import (
	"context"
	"sync"
)

// RestartWithContext loops infinitely, calling handler and after loop until either error is resturned
// during handling, or ctx finishes.
func RestartWithContext(ctx context.Context, handler func() error, afterLoop func(), result chan<- error) {
	loopingDone := make(chan error, 1) // size of 1 since we don't want to leave goroutine blocked indifinitely if ctx.Done is caught earlier

	stopRestarting := false
	stopLock := &sync.RWMutex{}

	go func() {
		defer close(loopingDone)

		for {
			err := handler()
			if err != nil {
				loopingDone <- err
				break
			}

			stopLock.RLock()
			shouldStop := stopRestarting
			stopLock.RUnlock()

			if shouldStop {
				break
			}

			afterLoop()
		}
	}()

	select {
	case <-ctx.Done():
		stopLock.Lock()
		stopRestarting = true
		stopLock.Unlock()
		result <- nil
	case err := <-loopingDone:
		result <- err
	}
}
