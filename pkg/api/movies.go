package api

import "sync"

// Movies is an aggregate state of the movies being served by the server instance.
// Any modification done on the state should be done by exposed methods which should guarantee goroutine access safety.
type Movies struct {
	items   map[string]Movie
	changes chan interface{}
	lock    *sync.RWMutex
}

// Add appends movies to the list of movies served on current server instance
func (m *Movies) Add(movies map[string]Movie) {
	addedMovies := map[string]Movie{}

	m.lock.Lock()
	for path, movie := range movies {
		if _, ok := m.items[path]; ok {
			continue
		}

		m.items[path] = movie
		addedMovies[path] = movie
	}
	m.lock.Unlock()

	if len(addedMovies) > 0 {
		m.changes <- MoviesChange{
			Variant: added,
			Items:   addedMovies,
		}
	}
}

// All returns a copy of all Movies being served by the instance of the server.
func (m *Movies) All() map[string]Movie {
	allMovies := map[string]Movie{}

	m.lock.RLock()
	defer m.lock.RUnlock()

	for path, movie := range m.items {
		allMovies[path] = movie
	}

	return allMovies
}

// ByPath returns a Movie by a provided path.
// When movie cannot be found, the error is being reported.
func (m *Movies) ByPath(path string) (Movie, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, movie := range m.items {
		if movie.Path == path {
			return movie, nil
		}
	}

	return Movie{}, errNoMovieAvailable
}

// Changes returns read-only channel notifying of movies changes
func (m *Movies) Changes() <-chan interface{} {
	return m.changes
}
