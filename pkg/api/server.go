package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/sarpt/mpv-web-api/pkg/mpv"
	"github.com/sarpt/mpv-web-api/pkg/probe"
)

const (
	logPrefix             = "api.Server#"
	fileLoop  loopVariant = "file"
	abLoop    loopVariant = "ab"
)

var (
	sseEventEnd = []byte("\n\n")
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

type loopVariant string

// PlaybackLoop contains information about playback loop
type PlaybackLoop struct {
	Variant loopVariant
	ATime   int
	BTime   int
}

// Playback contains information about currently played movie file
type Playback struct {
	CurrentTime        float64
	CurrentChapterIdx  int
	Fullscreen         bool
	Movie              Movie
	SelectedAudioID    int
	SelectedSubtitleID int
	Paused             bool
	Loop               PlaybackLoop
}

// Server is used to serve API and hold state accessible to the API
type Server struct {
	address               string
	allowCors             bool
	movies                []Movie
	moviesLock            *sync.RWMutex
	mpvManager            *mpv.Manager
	mpvSocketPath         string
	playback              *Playback
	playbackChanges       chan Playback
	playbackObservers     map[string]chan Playback
	playbackObserversLock *sync.RWMutex
	outLog                *log.Logger
	errLog                *log.Logger
}

// Config controls behaviour of the api serve
type Config struct {
	Address       string
	AllowCors     bool
	MpvSocketPath string
	outWriter     io.Writer
	errWriter     io.Writer
}

// NewServer prepares and returns a server that can be used to handle API
func NewServer(cfg Config) (*Server, error) {
	if cfg.outWriter == nil {
		cfg.outWriter = os.Stdout
	}
	if cfg.errWriter == nil {
		cfg.errWriter = os.Stderr
	}

	mpvManager := mpv.NewManager(cfg.MpvSocketPath, cfg.outWriter, cfg.errWriter)

	playback := &Playback{}

	return &Server{
		cfg.Address,
		cfg.AllowCors,
		[]Movie{},
		&sync.RWMutex{},
		mpvManager,
		cfg.MpvSocketPath,
		playback,
		make(chan Playback),
		map[string]chan Playback{},
		&sync.RWMutex{},
		log.New(cfg.outWriter, logPrefix, log.LstdFlags),
		log.New(cfg.errWriter, logPrefix, log.LstdFlags),
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

	s.outLog.Printf("running server at %s\n", s.address)
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
		mpv.LoopFileProperty:     s.handleLoopFileEvent,
		mpv.PauseProperty:        s.handlePauseEvent,
		mpv.PathProperty:         s.handlePathEvent,
		mpv.PlaybackTimeProperty: s.handlePlaybackTimeEvent,
	}

	go s.watchPlaybackChanges()
	go s.watchObservePropertyResponses(observeHandlers, observeResponses)

	return s.observeProperties(observeResponses)
}

func (s Server) watchPlaybackChanges() {
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
}

func (s Server) watchObservePropertyResponses(observeHandlers map[string]observeHandler, observeResponses chan mpv.ObserveResponse) {
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
			s.errLog.Printf("could not handle property '%s' observer handling: %s\n", observeResponse.Property, err)
		}
		s.playbackChanges <- *s.playback
	}
}

func (s Server) observeProperties(observeResponses chan mpv.ObserveResponse) error {
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

	out = append(out, sseEventEnd...)
	return out
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
