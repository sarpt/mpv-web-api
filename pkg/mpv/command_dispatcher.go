package mpv

import (
	"bytes"
	"encoding/json"
	"net"
)

const (
	socketType = "unix"
	readBuffer = 512
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
}

// NewCommandDispatcher returns dispatcher connected to the socket
// Error returned when connection to the socket is not possible
func NewCommandDispatcher(socketPath string) (*CommandDispatcher, error) {
	conn, err := net.Dial(socketType, socketPath)
	if err != nil {
		return nil, err
	}

	return &CommandDispatcher{
		conn: conn,
	}, nil
}

// Dispatch sends a commmand to the mpv using socket in path provided during construction
// Returns result sent back by mpv
// TODO: implement async commands handlling
// TODO: implement requestId check
func (cd *CommandDispatcher) Dispatch(command []string) (Result, error) {
	var result Result

	cmd := Command{
		Command:   command,
		RequestID: cd.requestID,
	}

	payload, err := json.Marshal(cmd)
	if err != nil {
		return result, err
	}

	payload = append(payload, []byte("\n")...)

	written, err := cd.conn.Write(payload)
	if err != nil || len(payload) != written {
		return result, err
	}

	cd.requestID++

	response, err := readUntilNewline(cd.conn)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(response, &result)

	return result, err
}

// Close makes connection by ipc to the mpv closed
func (cd CommandDispatcher) Close() {
	cd.conn.Close()
}

func readUntilNewline(conn net.Conn) ([]byte, error) {
	buf := make([]byte, readBuffer)
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
