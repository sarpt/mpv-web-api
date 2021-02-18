package sse

import (
	"encoding/json"

	"github.com/sarpt/mpv-web-api/internal/state"
)

const (
	moviesSSEChannelVariant state.SSEChannelVariant = "movies"
)

type moviesMapChange struct {
	Movies map[string]state.Movie
}

func (mmc moviesMapChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(mmc.Movies)
}

func (s *Server) createMoviesReplayHandler() sseReplayHandler {
	return func(res ResponseWriter) error {
		return res.SendChange(moviesMapChange{Movies: s.movies.All()}, moviesSSEChannelVariant, string(state.AddedMoviesChange))
	}
}

func (s *Server) createMoviesChangeHandler() sseChangeHandler {
	return func(res ResponseWriter, changes interface{}) error {
		moviesChange, ok := changes.(state.MoviesChange)
		if !ok {
			return errIncorrectChangesType
		}

		return res.SendChange(moviesChange, moviesSSEChannelVariant, string(state.AddedMoviesChange))
	}
}

func (s *Server) moviesSSEChannel() channel {
	return channel{
		variant:       moviesSSEChannelVariant,
		observers:     s.moviesObservers,
		changeHandler: s.createMoviesChangeHandler(),
		replayHandler: s.createMoviesReplayHandler(),
	}
}
