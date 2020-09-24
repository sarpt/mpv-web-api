package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

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
		res.WriteHeader(400)
		res.Write([]byte(fmt.Sprintf("could not prepare output: %s\n", err))) // good enough for poc

		return
	}

	res.WriteHeader(200)
	res.Write(response)
}
