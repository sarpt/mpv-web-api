package api

import (
	"net/http"
)

const (
	ssePath  = "/sse/"
	restPath = "/rest/"
)

func (s *Server) mainHandler() *http.ServeMux {
	sseHandlers := s.sseServer.Handler()
	restHandler := s.restServer.Handler()

	mux := http.NewServeMux()
	mux.Handle(ssePath, sseHandlers)
	mux.Handle(restPath, restHandler)

	return mux
}
