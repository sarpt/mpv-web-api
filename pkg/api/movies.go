package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/probe"
)

// MoviesChangeVariant specifies what type of change to movies list items belong to in a MoviesChange type.
type MoviesChangeVariant string

var (
	errNoMovieAvailable = errors.New("movie with specified path does not exist")
)

const (
	moviesSSEChannelVariant SSEChannelVariant = "movies"

	added   MoviesChangeVariant = "added"
	updated MoviesChangeVariant = "updated"
	removed MoviesChangeVariant = "removed"
)

// MoviesChange holds information about changes to the list of movies being served.
type MoviesChange struct {
	Variant MoviesChangeVariant
	Items   map[string]Movie
}

// Movie specifies information about a movie file that can be played
type Movie struct {
	Title           string
	FormatName      string
	FormatLongName  string
	Chapters        []probe.Chapter
	AudioStreams    []probe.AudioStream
	Duration        float64
	Path            string
	SubtitleStreams []probe.SubtitleStream
	VideoStreams    []probe.VideoStream
}

type getMoviesRespone struct {
	Movies map[string]Movie `json:"movies"`
}

// Movies is an aggregate state of the movies being served by the server instance.
// Any modification done on the state should be done by exposed methods which should guarantee goroutine access safety.
type Movies struct {
	items   map[string]Movie
	Changes chan interface{}
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
		m.Changes <- MoviesChange{
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

func (s *Server) getMoviesHandler(res http.ResponseWriter, req *http.Request) {
	moviesResponse := getMoviesRespone{
		Movies: s.movies.All(),
	}

	response, err := json.Marshal(&moviesResponse)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintln("could not prepare output")))

		return
	}

	res.WriteHeader(200)
	res.Write(response)
}

func (s *Server) createMoviesReplayHandler() sseReplayHandler {
	return func(res SSEResponseWriter) error {
		return sendMovies(s.movies.All(), res)
	}
}

func (s *Server) createMoviesChangeHandler() sseChangeHandler {
	return func(res SSEResponseWriter, changes interface{}) error {
		moviesChange, ok := changes.(MoviesChange)
		if !ok {
			return errIncorrectChangesType
		}

		return sendMovies(moviesChange.Items, res)
	}
}

func (s *Server) moviesSSEChannel() SSEChannel {
	return SSEChannel{
		Variant:       moviesSSEChannelVariant,
		Observers:     s.moviesSSEObservers,
		ChangeHandler: s.createMoviesChangeHandler(),
		ReplayHandler: s.createMoviesReplayHandler(),
	}
}

func sendMovies(movies map[string]Movie, res SSEResponseWriter) error {
	out, err := json.Marshal(movies)
	if err != nil {
		return errResponseJSONCreationFailed
	}

	_, err = res.Write(formatSseEvent(string(added), out))
	if err != nil {
		return fmt.Errorf("sending movies failed: %s: %w", errClientWritingFailed.Error(), err)
	}

	return nil
}

func mapProbeResultToMovie(result probe.Result) Movie {
	return Movie{
		Title:           result.Format.Title,
		FormatName:      result.Format.Name,
		FormatLongName:  result.Format.LongName,
		Chapters:        result.Chapters,
		Path:            result.Path,
		VideoStreams:    result.VideoStreams,
		AudioStreams:    result.AudioStreams,
		SubtitleStreams: result.SubtitleStreams,
		Duration:        result.Format.Duration,
	}
}
