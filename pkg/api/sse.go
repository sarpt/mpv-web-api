package api

import (
	"bytes"
	"fmt"
	"net/http"
)

var (
	sseEventEnd = []byte("\n\n")
)

func sseFlusher(res http.ResponseWriter) (http.Flusher, error) {
	flusher, ok := res.(http.Flusher)
	if !ok {
		return flusher, fmt.Errorf("could not instantiate http sse flusher")
	}

	res.Header().Set("Connection", "keep-alive")
	res.Header().Set("Content-Type", "text/event-stream")
	res.Header().Set("Access-Control-Allow-Origin", "*")

	return flusher, nil
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
