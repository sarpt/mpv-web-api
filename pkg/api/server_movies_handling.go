package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

type getMoviesRespone struct {
	Movies map[string]Movie `json:"movies"`
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
		return fmt.Errorf("%w: %s", errResponseJSONCreationFailed, err)
	}

	_, err = res.Write(formatSseEvent(string(added), out))
	if err != nil {
		return fmt.Errorf("sending movies failed: %w: %s", errClientWritingFailed, err)
	}

	return nil
}
