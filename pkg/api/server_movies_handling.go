package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sarpt/mpv-web-api/internal/state"
)

const (
	moviesSSEChannelVariant state.SSEChannelVariant = "movies"
)

type getMoviesRespone struct {
	Movies map[string]state.Movie `json:"movies"`
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
		moviesChange, ok := changes.(state.MoviesChange)
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

func sendMovies(movies map[string]state.Movie, res SSEResponseWriter) error {
	out, err := json.Marshal(movies)
	if err != nil {
		return fmt.Errorf("%w: %s", errResponseJSONCreationFailed, err)
	}

	_, err = res.Write(formatSseEvent(moviesSSEChannelVariant, string(state.AddedMoviesChange), out))
	if err != nil {
		return fmt.Errorf("sending movies failed: %w: %s", errClientWritingFailed, err)
	}

	return nil
}
