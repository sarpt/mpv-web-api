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
	added   MoviesChangeVariant = "added"
	updated MoviesChangeVariant = "updated"
	removed MoviesChangeVariant = "removed"

	errNoMovieAvailable = errors.New("movie with specified path does not exist")
)

// MoviesChange holds information about changes to the list of movies being served.
type MoviesChange struct {
	variant MoviesChangeVariant
	items   []Movie
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
	Movies []Movie `json:"movies"`
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

func (s *Server) getSseMoviesHandler(res http.ResponseWriter, req *http.Request) {
	flusher, err := sseFlusher(res)
	if err != nil {
		res.WriteHeader(400)
		return
	}

	moviesChanges := make(chan MoviesChange, 1)
	s.moviesChangesObserversLock.Lock()
	s.moviesChangesObservers[req.RemoteAddr] = moviesChanges
	s.moviesChangesObserversLock.Unlock()

	for {
		select {
		case change, ok := <-moviesChanges:
			if !ok {
				return
			}

			out, err := json.Marshal(change)
			if err != nil {
				s.errLog.Println("could not create response")
				continue
			}

			_, err = res.Write(formatSseEvent(string(change.variant), out))
			if err != nil {
				s.errLog.Println("could not write to the client")
				continue
			}

			flusher.Flush()
		case <-req.Context().Done():
			s.moviesChangesObserversLock.Lock()
			delete(s.moviesChangesObservers, req.RemoteAddr)
			s.moviesChangesObserversLock.Unlock()
			return
		}
	}
}

func (s Server) movieByPath(path string) (Movie, error) {
	for _, movie := range s.movies {
		if movie.Path == path {
			return movie, nil
		}
	}

	return Movie{}, errNoMovieAvailable
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
