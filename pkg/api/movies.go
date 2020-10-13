package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sarpt/mpv-web-api/pkg/probe"
)

// MoviesChangeVariant specifies what type of change to movies list items belong to in a MoviesChange type.
type MoviesChangeVariant string

var (
	errNoMovieAvailable = errors.New("movie with specified path does not exist")
)

const (
	moviesObserverVariant StatusObserverVariant = "movies"

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

// AddMovies appends movies to the list of movies served on current server instance
func (s *Server) AddMovies(movies map[string]Movie) {
	addedMovies := map[string]Movie{}

	s.moviesLock.Lock()
	for path, movie := range movies {
		if _, ok := s.movies[path]; ok {
			continue
		}

		s.movies[path] = movie
		addedMovies[path] = movie
	}
	s.moviesLock.Unlock()

	if len(addedMovies) > 0 {
		s.moviesChanges <- MoviesChange{
			Variant: added,
			Items:   addedMovies,
		}
	}
}

func (s *Server) getMoviesHandler(res http.ResponseWriter, req *http.Request) {
	s.moviesLock.Lock()
	moviesResponse := getMoviesRespone{
		Movies: s.movies,
	}
	s.moviesLock.Unlock()

	response, err := json.Marshal(&moviesResponse)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(fmt.Sprintln("could not prepare output")))

		return
	}

	res.WriteHeader(200)
	res.Write(response)
}

func (s Server) movieByPath(path string) (Movie, error) {
	for _, movie := range s.movies {
		if movie.Path == path {
			return movie, nil
		}
	}

	return Movie{}, errNoMovieAvailable
}

func (s *Server) createMoviesReplayHandler() sseReplayHandler {
	return func(res http.ResponseWriter, flusher http.Flusher) error {
		return sendMovies(s.movies, res, flusher)
	}
}

func (s *Server) createMoviesChangeHandler() sseChangeHandler {
	return func(res http.ResponseWriter, flusher http.Flusher, changes interface{}) error {
		moviesChange, ok := changes.(MoviesChange)
		if !ok {
			return errIncorrectChangesType
		}

		return sendMovies(moviesChange.Items, res, flusher)
	}
}

func (s *Server) createGetSseMoviesHandler() getSseHandler {
	cfg := SseHandlerConfig{
		ObserverVariant: moviesObserverVariant,
		Observers:       s.moviesChangesObservers,
		ChangeHandler:   s.createMoviesChangeHandler(),
		ReplayHandler:   s.createMoviesReplayHandler(),
	}

	return s.createGetSseHandler(cfg)
}

func sendMovies(movies map[string]Movie, res http.ResponseWriter, flusher http.Flusher) error {
	out, err := json.Marshal(movies)
	if err != nil {
		return errResponseJSONCreationFailed
	}

	_, err = res.Write(formatSseEvent(string(added), out))
	if err != nil {
		return errClientWritingFailed
	}

	flusher.Flush()
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
