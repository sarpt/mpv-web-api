package api

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/mpv"
	"github.com/sarpt/mpv-web-api/pkg/probe"
)

type observeHandler = func(res mpv.ObserveResponse) error

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

// Playback contains information about currently played movie file
type Playback struct {
	CurrentTime        float64
	CurrentChapterIdx  int
	Fullscreen         bool
	Movie              Movie
	SelectedAudioID    int
	SelectedSubtitleID int
}

// Server is used to serve API and hold state accessible to the API
type Server struct {
	address               string
	allowCors             bool
	movies                []Movie
	mpvManager            *mpv.Manager
	mpvSocketPath         string
	playback              *Playback
	playbackChanges       chan Playback
	playbackObservers     map[string]chan Playback
	playbackObserversLock *sync.RWMutex
}

// Config controls behaviour of the api serve
type Config struct {
	Address           string
	AllowCors         bool
	MoviesDirectories []string
	MpvSocketPath     string
}

// NewServer prepares and returns a server that can be used to handle API
func NewServer(cfg Config) (*Server, error) {
	mpvManager := mpv.NewManager(cfg.MpvSocketPath)

	movies := probeDirectories(cfg.MoviesDirectories)
	playback := &Playback{}

	return &Server{
		cfg.Address,
		cfg.AllowCors,
		movies,
		mpvManager,
		cfg.MpvSocketPath,
		playback,
		make(chan Playback),
		map[string]chan Playback{},
		&sync.RWMutex{},
	}, nil
}

// Serve starts handling requests to the API endpoints. Blocks until canceled
func (s *Server) Serve() error {
	serv := http.Server{
		Addr:    s.address,
		Handler: s.mainHandler(),
	}

	err := s.initWatchers()
	if err != nil {
		return errors.New("could not start watching for properties")
	}

	fmt.Fprintf(os.Stdout, "running server at %s\n", s.address)
	return serv.ListenAndServe()
}

// Close closes server, along with closing necessary helpers
func (s Server) Close() {
	s.mpvManager.Close()
}

func (s *Server) initWatchers() error {
	observeResponses := make(chan mpv.ObserveResponse)
	observeHandlers := map[string]observeHandler{
		mpv.FullscreenProperty:   s.handleFullscreenEvent,
		mpv.PathProperty:         s.handlePathEvent,
		mpv.PlaybackTimeProperty: s.handlePlaybackTimeEvent,
	}

	go func() {
		for {
			playback, ok := <-s.playbackChanges
			if !ok {
				return
			}

			s.playbackObserversLock.RLock()
			for _, observer := range s.playbackObservers {
				observer <- playback
			}
			s.playbackObserversLock.RUnlock()
		}
	}()

	go func() {
		for {
			observeResponse, open := <-observeResponses
			if !open {
				return
			}

			observeHandler, ok := observeHandlers[observeResponse.Property]
			if !ok {
				continue
			}

			err := observeHandler(observeResponse)
			if err != nil {
				fmt.Fprintf(os.Stdout, "could not handle property '%s' observer handling: %s\n", observeResponse.Property, err)
			}
			s.playbackChanges <- *s.playback
		}
	}()

	for _, propertyName := range mpv.ObservableProperties {
		_, err := s.mpvManager.SubscribeToProperty(propertyName, observeResponses)
		if err != nil {
			return fmt.Errorf("could not initialize watchers due to error when observing property: %w", err)
		}
	}

	return nil
}

func (s Server) movieByPath(path string) (Movie, error) {
	for _, movie := range s.movies {
		if movie.Path == path {
			return movie, nil
		}
	}

	return Movie{}, errNoMovieAvailable
}

func formatSseEvent(eventName string, data []byte) []byte {
	var out []byte

	out = append(out, []byte(fmt.Sprintf("event:%s\n", eventName))...)

	dataEntries := bytes.Split(data, []byte("\n"))
	for _, dataEntry := range dataEntries {
		out = append(out, []byte(fmt.Sprintf("data:%s\n", dataEntry))...)
	}

	out = append(out, []byte("\n\n")...)
	return out
}

func probeDirectories(directories []string) []Movie {
	var movies []Movie

	probeResults, _ := probe.Directories(directories)
	for _, probeResult := range probeResults {
		if !probeResult.IsMovieFile() {
			continue
		}

		movie := Movie{
			Title:           probeResult.Format.Title,
			FormatName:      probeResult.Format.Name,
			FormatLongName:  probeResult.Format.LongName,
			Chapters:        probeResult.Chapters,
			Path:            probeResult.Path,
			VideoStreams:    probeResult.VideoStreams,
			AudioStreams:    probeResult.AudioStreams,
			SubtitleStreams: probeResult.SubtitleStreams,
			Duration:        probeResult.Format.Duration,
		}

		movies = append(movies, movie)
	}

	return movies
}
