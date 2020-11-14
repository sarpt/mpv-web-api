package api

import (
	"net/http"
	"sync"
)

// SSEResponseWriter is used to send data through keep-alive SSE connection.
// The writer and flusher are protected by lock since multiple go routines use the same connection to send events.
type SSEResponseWriter struct {
	res     http.ResponseWriter
	flusher http.Flusher
	lock    *sync.Mutex
}

// Write sends data through the connection
func (f *SSEResponseWriter) Write(data []byte) (int, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	n, err := f.res.Write(data)
	if err == nil {
		f.flusher.Flush()
	}

	return n, err
}
