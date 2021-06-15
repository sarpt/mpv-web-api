package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/sarpt/mpv-web-api/internal/state"
)

// ResponseWriter is used to send data through keep-alive SSE connection.
// The writer and flusher are protected by lock since multiple go routines use the same connection to send events.
type ResponseWriter struct {
	res     http.ResponseWriter
	flusher http.Flusher
	lock    *sync.Mutex
}

// Write sends data through the connection
func (f *ResponseWriter) Write(data []byte) (int, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	n, err := f.res.Write(data)
	if err == nil {
		f.flusher.Flush()
	}

	return n, err
}

// SendChange is responsible for propgating change payload through SSE connection.
func (f *ResponseWriter) SendChange(changePayload json.Marshaler, channelVariant state.SSEChannelVariant, changeVariant string) error {
	out, err := json.Marshal(changePayload)
	if err != nil {
		return fmt.Errorf("%w: %s", errResponseJSONCreationFailed, err)
	}

	_, err = f.writeChange(out, channelVariant, changeVariant)
	return err
}

// SendEmptyChange is responsible for propagating change without any payload (without "data") through SSE connection.
func (f *ResponseWriter) SendEmptyChange(channelVariant state.SSEChannelVariant, changeVariant string) error {
	_, err := f.writeChange([]byte{}, channelVariant, changeVariant)
	return err
}

func (f *ResponseWriter) writeChange(out []byte, channelVariant state.SSEChannelVariant, changeVariant string) (int, error) {
	n, err := f.Write(formatSseEvent(channelVariant, string(changeVariant), out))
	if err != nil {
		return n, fmt.Errorf("writing change %s on %s channel failed: %w: %s", changeVariant, channelVariant, errClientWritingFailed, err)
	}

	return n, nil
}
