package mpv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

const (
	socketType = "unix"
	bufSize    = 512
)

// Command represents command payload sent to the mpv
type Command struct {
	Command   []string `json:"command"`
	RequestID int      `json:"request_id"`
}

// Result holds data returned after command executon
type Result struct {
	Err       string      `json:"error"`
	RequestID int         `json:"request_id"`
	Data      interface{} `json:"data"`
}

// CommandDispatcher connects to the provided socket path and handles sending commands and handling results
type CommandDispatcher struct {
	conn      net.Conn
	requestID int
	requests  map[int]chan Result
}

// NewCommandDispatcher returns dispatcher connected to the socket
// Error is returned when connection to the socket failed
func NewCommandDispatcher(socketPath string) (*CommandDispatcher, error) {
	var conn net.Conn
	var err error

	for {
		conn, err = net.Dial(socketType, socketPath)
		if err == nil {
			break
		}

		time.Sleep(1 * time.Second) // mpv takes a longer moment to start listening on the socket, repeat until connection succesful; TODO: add timeout
	}

	cd := &CommandDispatcher{
		conn:      conn,
		requests:  make(map[int]chan Result),
		requestID: 1,
	}

	cd.listenUnixSocket()
	return cd, nil
}

// Dispatch sends a commmand to the mpv using socket in path provided during construction
// Returns result sent back by mpv
func (cd *CommandDispatcher) Dispatch(commands []string) (Result, error) {
	var result Result

	payload, err := prepareCommandPayload(commands, cd.requestID)
	if err != nil {
		return result, err
	}

	requestResult := make(chan Result)
	cd.requests[cd.requestID] = requestResult

	written, err := cd.conn.Write(payload)
	if err != nil || len(payload) != written {
		return result, err
	}

	cd.requestID++

	result = <-requestResult
	return result, err
}

// Close makes connection by ipc to the mpv closed
func (cd CommandDispatcher) Close() {
	cd.conn.Close()
}

func (cd CommandDispatcher) listenUnixSocket() {
	go func() {
		for {
			var result Result

			payload, err := readUntilNewline(cd.conn)
			if err != nil {
				if err == io.EOF {
					fmt.Fprintf(os.Stderr, "connection closed\n")
				} else {
					fmt.Fprintf(os.Stderr, "could not read the payload from the connection\n")
				}

				return
			}

			err = json.Unmarshal(payload, &result)
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not parse the response\n")
				continue
			}

			request, ok := cd.requests[result.RequestID]
			if !ok {
				fmt.Fprintf(os.Stderr, "result %d provided to not dispatched request\n", result.RequestID)
				continue
			}

			request <- result
			close(request)
		}
	}()
}

func prepareCommandPayload(commands []string, requestID int) ([]byte, error) {
	var payload []byte
	cmd := Command{
		Command:   commands,
		RequestID: requestID,
	}

	payload, err := json.Marshal(cmd)
	if err != nil {
		return payload, err
	}

	payload = append(payload, []byte("\n")...)

	return payload, nil
}

func readUntilNewline(conn net.Conn) ([]byte, error) {
	buf := make([]byte, bufSize)
	var result []byte

	for {
		nRead, err := conn.Read(buf)
		if err != nil {
			return result, err
		}

		result = append(result, buf[:nRead]...)

		newlineIdx := bytes.Index(buf, []byte("\n"))
		if newlineIdx != -1 {
			return result, nil
		}
	}
}
