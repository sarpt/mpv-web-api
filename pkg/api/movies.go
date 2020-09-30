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
	Variant MoviesChangeVariant
	Items   []Movie
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

// AddMovies appends movies to the list of movies served on current server instance
func (s *Server) AddMovies(movies []Movie) {
	s.moviesLock.Lock()
	s.movies = append(s.movies, movies...)
	s.moviesLock.Unlock()

	s.moviesChanges <- MoviesChange{
		Variant: added,
		Items:   movies,
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

	s.outLog.Printf("added /sse/movies observer with addr %s\n", req.RemoteAddr)

	for {
		select {
		case change, ok := <-moviesChanges:
			if !ok {
				return
			}

			out, err := json.Marshal(change.Items)
			if err != nil {
				s.errLog.Println("could not create response")
				continue
			}

			_, err = res.Write(formatSseEvent(string(change.Variant), out))
			if err != nil {
				s.errLog.Println("could not write to the client")
				continue
			}

			flusher.Flush()
		case <-req.Context().Done():
			s.moviesChangesObserversLock.Lock()
			delete(s.moviesChangesObservers, req.RemoteAddr)
			s.moviesChangesObserversLock.Unlock()
			s.outLog.Printf("removing /sse/movies observer with addr %s\n", req.RemoteAddr)

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

// watchMoviesChanges reads all moviesChanges done by path/event handlers.
// It's a fan-out dispatcher, which notifies all movies observers (subscribers from SSE etc.) when a moviesChange occurs.
// TODO: this method does not differ all that much from playbackChanges and seems like it's quite generic -> need to consider creating some abstraction over this
func (s Server) watchMoviesChanges() {
	for {
		changes, ok := <-s.moviesChanges
		if !ok {
			return
		}

		s.moviesChangesObserversLock.RLock()
		for _, observer := range s.moviesChangesObservers {
			observer <- changes
		}
		s.moviesChangesObserversLock.RUnlock()
	}
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
