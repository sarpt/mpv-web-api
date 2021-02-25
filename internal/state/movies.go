package state

import (
	"encoding/json"
	"errors"
	"sync"
)

var (
	errNoMovieAvailable = errors.New("movie with specified path does not exist")
)

const (
	// AddedMoviesChange notifies about addition of movies to the list of movies handled by the application.
	AddedMoviesChange MoviesChangeVariant = "added"

	// UpdatedMoviesChange notifies about updates to the list of movies.
	UpdatedMoviesChange MoviesChangeVariant = "updated"

	// RemovedMoviesChange notifies about removal of movies from the list.
	RemovedMoviesChange MoviesChangeVariant = "removed"
)

// MoviesChange holds information about changes to the list of movies being served.
type MoviesChange struct {
	variant MoviesChangeVariant
	items   map[string]Movie
}

// MarshalJSON returns change items in JSON format. Satisfies json.Marshaller.
func (mc MoviesChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(mc.items)
}

// MoviesChangeVariant specifies what type of change to movies list items belong to in a MoviesChange type.
type MoviesChangeVariant string

// Movies is an aggregate state of the movies being served by the server instance.
// Any modification done on the state should be done by exposed methods which should guarantee goroutine access safety.
type Movies struct {
	items   map[string]Movie
	changes chan interface{}
	lock    *sync.RWMutex
}

// NewMovies counstructs Movies state.
func NewMovies() *Movies {
	return &Movies{
		items:   map[string]Movie{},
		changes: make(chan interface{}),
		lock:    &sync.RWMutex{},
	}
}

// Add appends a movie to the list of movies served on current server instance.
func (m *Movies) Add(movie Movie) {
	addedMovies := map[string]Movie{}
	path := movie.path

	m.lock.Lock()
	if _, ok := m.items[path]; ok {
		return
	}

	m.items[path] = movie
	m.lock.Unlock()

	addedMovies[path] = movie
	m.changes <- MoviesChange{
		variant: AddedMoviesChange,
		items:   addedMovies,
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
		if movie.path == path {
			return movie, nil
		}
	}

	return Movie{}, errNoMovieAvailable
}

// Changes returns read-only channel notifying of movies changes.
func (m *Movies) Changes() <-chan interface{} {
	return m.changes
}
