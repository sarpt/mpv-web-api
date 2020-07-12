package mpv

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

const (
	socketType = "unix"
	bufSize    = 512

	resultSuccess = "success"
)

var (
	// ErrCommandFailedResponse informs about mpv returning something other than "success" in an error field of a response
	ErrCommandFailedResponse = errors.New("mpv response does not include success state")
)

// CommandPayload represents command payload sent to the mpv
type CommandPayload struct {
	Command   []interface{} `json:"command"`
	RequestID int           `json:"request_id"`
}

// Response is a result of executing mpv request command.
type Response struct {
	Data interface{} `json:"data"`
}

// ObserveResponse is a result of mpv emitting event with a property change
type ObserveResponse struct {
	Response
	Property string
}

// ResponsePayload holds data returned after mpv command execution through json IPC.
type ResponsePayload struct {
	Err       string      `json:"error"`
	RequestID int         `json:"request_id"`
	ID        int         `json:"id"`
	Event     string      `json:"event"`
	Name      string      `json:"name"`
	Data      interface{} `json:"data"`
}

// CommandDispatcher connects to the provided socket path and handles sending commands and handling results
type CommandDispatcher struct {
	conn                   net.Conn
	requests               map[int]chan ResponsePayload
	requestID              int
	requestIDLock          *sync.Mutex
	propertyObservers      map[string]propertyObserverGroup
	propertyObserverID     int
	propertyObserverIDLock *sync.Mutex
}

type propertyObserverGroup struct {
	responsePayloads chan ResponsePayload
	observers        map[int]propertyObserver
}

type propertyObserver struct {
	propertyChanges chan<- ObserveResponse
	done            chan bool
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
		conn:                   conn,
		requests:               make(map[int]chan ResponsePayload),
		requestID:              1,
		requestIDLock:          &sync.Mutex{},
		propertyObservers:      make(map[string]propertyObserverGroup),
		propertyObserverID:     1,
		propertyObserverIDLock: &sync.Mutex{},
	}

	cd.listenOnUnixSocket()
	return cd, nil
}

// ObserveProperty listen to observe property mpv events.
// Returned id is used as a key to listened observe property mpv events. Id should be used to unsubscribe.
// When error is encountered id is useless.
// The channel provided is never closed to enable aggregation from multiple observers.
// However calling unsubscribe will ensure that command dispatcher will stop trying to send on a specified channel.
func (cd *CommandDispatcher) ObserveProperty(propertyName string, out chan<- ObserveResponse) (int, error) {
	var responsePayloads chan ResponsePayload

	done := make(chan bool)
	propertyObserverID := cd.ReservePropertyObserverID()

	propertyObservers, ok := cd.propertyObservers[propertyName]
	if !ok {
		responsePayloads = make(chan ResponsePayload)
		outputs := make(map[int]propertyObserver)
		outputs[propertyObserverID] = propertyObserver{
			propertyChanges: out,
			done:            done,
		}
		newObserver := propertyObserverGroup{
			responsePayloads: responsePayloads,
			observers:        outputs,
		}
		cd.propertyObservers[propertyName] = newObserver
		command := NewObserveProperty(propertyObserverID, propertyName)
		_, err := cd.Request(command)
		if err != nil {
			return 0, err
		}
	} else {
		responsePayloads = propertyObservers.responsePayloads
		propertyObservers.observers[propertyObserverID] = propertyObserver{
			propertyChanges: out,
			done:            done,
		}
	}

	go func() {
		var payload ResponsePayload
		for {
			select {
			case payload = <-responsePayloads:
				out <- ObserveResponse{
					Property: propertyName,
					Response: Response{
						Data: payload.Data,
					},
				}
			case <-done:
				delete(propertyObservers.observers, propertyObserverID)
				return
			}
		}
	}()

	return propertyObserverID, nil
}

// UnobserveProperty instructs command dispatcher to stop sending updates about property on specified id.
func (cd *CommandDispatcher) UnobserveProperty(propertyName string, id int) error {

	propertyObservers, ok := cd.propertyObservers[propertyName]
	if !ok {
		return errors.New("could not find observer for a provided property name")
	}

	propertyObserver, ok := propertyObservers.observers[id]
	if !ok {
		return errors.New("could not find observer for a provided observer id")
	}

	propertyObserver.done <- true
	return nil
}

// Request is used to send simple request->response command that is completed after the first response from mpv comes.
func (cd *CommandDispatcher) Request(command Command) (Response, error) {
	var resPayload ResponsePayload
	var result Response

	requestResult := make(chan ResponsePayload)

	requestID := cd.ReserveRequestID()
	cd.requests[requestID] = requestResult
	defer delete(cd.requests, requestID)

	err := cd.Dispatch(command, requestID)
	if err != nil {
		return result, err
	}

	resPayload = <-requestResult
	if !IsResultSuccess(resPayload) {
		return result, ErrCommandFailedResponse
	}

	return Response{
		Data: resPayload.Data,
	}, nil
}

// ReserveRequestID takes a requestID from the available pool in a concurrent safe way.
func (cd *CommandDispatcher) ReserveRequestID() int {
	cd.requestIDLock.Lock()
	defer cd.requestIDLock.Unlock()

	requestID := cd.requestID
	cd.requestID++

	return requestID
}

// ReservePropertyObserverID takes a subscriptionID from the available pool in a concurrent safe way.
func (cd *CommandDispatcher) ReservePropertyObserverID() int {
	cd.propertyObserverIDLock.Lock()
	defer cd.propertyObserverIDLock.Unlock()

	propertyObserverID := cd.propertyObserverID
	cd.propertyObserverID++

	return propertyObserverID
}

// Dispatch sends a commmand with specified requestID to the mpv using socket in path provided during construction.
// Returns error if command was not correctly dispatched.
func (cd *CommandDispatcher) Dispatch(command Command, requestID int) error {
	payload, err := prepareCommandPayload(command, requestID)
	if err != nil {
		return err
	}

	written, err := cd.conn.Write(payload)
	if err != nil || len(payload) != written {
		return err
	}

	return nil
}

// Close makes connection by ipc to the mpv closed
func (cd CommandDispatcher) Close() {
	cd.conn.Close()
}

// IsResultSuccess return whether returned result specifies successful command execution
func IsResultSuccess(result ResponsePayload) bool {
	return result.Err == resultSuccess
}

func (cd CommandDispatcher) listenOnUnixSocket() {
	go func() {
		for {
			var result ResponsePayload

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

			if result.Event == propertyChangeEvent {
				propertyObserver, ok := cd.propertyObservers[result.Name]
				if !ok {
					fmt.Fprintf(os.Stderr, "observe property event provided to not observed property %s\n", result.Name)
					continue
				}

				propertyObserver.responsePayloads <- result
			} else {
				if result.RequestID == 0 {
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
		}
	}()
}

func prepareCommandPayload(command Command, requestID int) ([]byte, error) {
	var payload []byte
	cmd := CommandPayload{
		Command:   command.Get(),
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
