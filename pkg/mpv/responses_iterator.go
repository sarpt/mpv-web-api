package mpv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
)

type responsesIterator struct {
	conn        net.Conn
	accumulator []byte
}

// NewResponsesIterator creates an iterator which returns ResponsePayload processed from
// provided connection.
func NewResponsesIterator(conn net.Conn) *responsesIterator {
	return &responsesIterator{
		conn: conn,
	}
}

// Next returns ResponsePayload fetched from a mpv socket connection.
// It blocks until a valid, newline-separated JSON is provided through the connection.
// If the newline-separated data does not form a correct JSON responses, it returns an error.
// Not every call to Next results in reading from a socket - if previous call to Next
// resulted in more than one newline-separated payloads being read, 'Next' will process
// payload right after the one returned on previous call to 'Next' without reading new data from socket.
func (ri *responsesIterator) Next() (ResponsePayload, error) {
	var result ResponsePayload

	chunk, err := ri.getNonEmptyChunkFromAccumulator()
	if err != nil {
		return result, err
	}

	chunkValid := json.Valid(chunk)
	if !chunkValid {
		return result, fmt.Errorf("received non-empty data from socket is not a correct json payload")
	}

	response, err := getResponsePayload(chunk)
	return response, err
}

func (ri *responsesIterator) fetchIntoAccumulator() (int, error) {
	buf := make([]byte, bufSize)

	nRead, err := ri.conn.Read(buf)
	if err == nil && nRead > 0 {
		ri.accumulator = append(ri.accumulator, buf[:nRead]...)
	}

	return nRead, err
}

// getNonEmptyChunkFromAccumulator reads accumulator until newline-separated non-empty chunk can be returned.
// Accumulator is read without making a read from socket as long as it contains any newlines.
// When accumulator does not contain any newlines anymore, socket is being read until it contains a non-empty newline.
func (ri *responsesIterator) getNonEmptyChunkFromAccumulator() ([]byte, error) {
	var firstNewByteIdx int = 0

	for {
		newlineIdx := bytes.Index(ri.accumulator[firstNewByteIdx:], newline)
		if newlineIdx != -1 {
			chunk := ri.takeFromAccumulator(newlineIdx)
			if len(chunk) == 0 {
				continue // consecutive newlines - discard and continue searching/fetching.
			}

			return chunk, nil
		}

		nRead, err := ri.fetchIntoAccumulator()
		if err != nil {
			return []byte{}, err
		}

		firstNewByteIdx = len(ri.accumulator) - nRead // A newline can only be on last nRead bytes, otherwise the read from socket would not occur.
	}
}

// takeFromAccumulator takes a newlineIdx, takes up to newlineIdx - 1 and
// discards from accumulator newline at newlineIdx.
func (ri *responsesIterator) takeFromAccumulator(newlineIdx int) []byte {
	result := append([]byte(nil), ri.accumulator[:newlineIdx]...)
	ri.accumulator = append([]byte(nil), ri.accumulator[newlineIdx+1:]...)

	return result
}