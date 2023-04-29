package revision

import "sync"

type Identifier = uint64

type Storage struct {
	lock     *sync.RWMutex
	revision Identifier
}

func NewStorage() *Storage {
	return &Storage{
		lock:     &sync.RWMutex{},
		revision: 0,
	}
}

func (rs *Storage) Revision() Identifier {
	rs.lock.RLock()
	defer rs.lock.RUnlock()

	return rs.revision
}

func (rs *Storage) Tick() {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	rs.revision += 1
}
