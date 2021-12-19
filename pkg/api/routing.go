package api

import (
	"fmt"
	"net/http"
)

func (s *Server) mainHandler() *http.ServeMux {
	mux := http.NewServeMux()
	for _, server := range s.pluginServers {
		mux.Handle(fmt.Sprintf("/%s/", server.PathBase()), server.Handler())
	}

	return mux
}
