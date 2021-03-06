package mpv

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

const (
	socketType = "unix"
	bufSize    = 512

	resultSuccess = "success"

	propertyChangeEvent = "property-change"
)

var (
	// ErrCommandFailedResponse informs about mpv returning something other than "success" in an error field of a response.
	ErrCommandFailedResponse = errors.New("mpv response does not include success state")

	// ErrConnectionInProgress informs about failure of operation due to connection of command dispatcher being in progress.
	ErrConnectionInProgress = errors.New("command dispatcher is connected to mpv socket")

	// ErrNoPropertyObserver informs about failure of finding observer for a specified property name (most likely property is not observed).
	ErrNoPropertyObserver = errors.New("could not find observer for a provided property name")

	// ErrNoPropertySubscription informs about failure of finding observer for a specified subscription id.
	ErrNoPropertySubscription = errors.New("could not find subscription for a provided subscription id")

	newline = []byte("\n")

	commandDispatcherLogPrefix = "mpv.CommandDispatcher#"
)

// commandPayload represents command payload sent to the mpv
type commandPayload struct {
	Command   []interface{} `json:"command"`
	RequestID int           `json:"request_id"`
}

// Response is a result of executing mpv request command.
type Response struct {
	Data interface{} `json:"data"`
}

// ObservePropertyResponse is a result of mpv emitting event with a property change
type ObservePropertyResponse struct {
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

// commandDispatcher connects to the provided socket path and handles sending commands and handling results
type commandDispatcher struct {
	socketPath                 string
	listeningOnSocket          bool
	linteningOnSocketLock      *sync.RWMutex
	conn                       net.Conn
	requests                   map[int]chan ResponsePayload
	requestID                  int
	requestIDLock              *sync.Mutex
	propertyObservers          map[string]propertyObserver
	propertyObserversLock      *sync.RWMutex
	propertySubscriptionID     int
	propertySubscriptionIDLock *sync.Mutex
	errLog                     *log.Logger
}

type propertyObserver struct {
	responsePayloads chan ResponsePayload
	subscriptions    map[int]propertySubscriber
}

type propertySubscriber struct {
	propertyChanges chan<- ObservePropertyResponse
	done            chan bool
}

// newCommandDispatcher returns dispatcher connected to the socket.
// Error is returned when connection to the socket fails.
func newCommandDispatcher(socketPath string, errWriter io.Writer) *commandDispatcher {
	return &commandDispatcher{
		socketPath:                 socketPath,
		listeningOnSocket:          false,
		linteningOnSocketLock:      &sync.RWMutex{},
		requests:                   make(map[int]chan ResponsePayload),
		requestID:                  1,
		requestIDLock:              &sync.Mutex{},
		propertyObservers:          make(map[string]propertyObserver),
		propertyObserversLock:      &sync.RWMutex{},
		propertySubscriptionID:     1,
		propertySubscriptionIDLock: &sync.Mutex{},
		errLog:                     log.New(errWriter, commandDispatcherLogPrefix, log.LstdFlags),
	}
}

func (cd *commandDispatcher) connectToMpvSocket() error {
	conn, err := waitForSocketConnection(cd.socketPath)
	if err != nil {
		return err
	}

	cd.conn = conn
	cd.listenOnUnixSocket()

	return nil
}

func (cd *commandDispatcher) observeProperties() error {
	cd.propertyObserversLock.RLock()
	defer cd.propertyObserversLock.RUnlock()

	for propertyName := range cd.propertyObservers {
		err := cd.observeProperty(propertyName)
		if err != nil {
			return err
		}
	}

	return nil
}

// addPropertyObserver creates a new observer for a specific property.
// The request to observer property will not be made if the connection is not estabilished since it will fail,
// but the observer is added to propertyObservers map which will be used during connection to start observing properties on a new connection.
func (cd *commandDispatcher) addPropertyObserver(propertyName string) (propertyObserver, error) {
	newObserver := propertyObserver{
		responsePayloads: make(chan ResponsePayload),
		subscriptions:    make(map[int]propertySubscriber),
	}

	cd.propertyObserversLock.Lock()
	cd.propertyObservers[propertyName] = newObserver
	cd.propertyObserversLock.Unlock()

	// Do not try to send a request when dispatcher is not connected to the MPV instance through the socket.
	if !cd.Connected() {
		return newObserver, nil
	}

	err := cd.observeProperty(propertyName)
	return newObserver, err
}

func (cd commandDispatcher) observeProperty(propertyName string) error {
	requestID := cd.reserveRequestID()
	cmd := command{
		name:     observePropertyCommand,
		elements: []interface{}{requestID, propertyName},
	}
	_, err := cd.Request(cmd)

	return err
}

// Connect attempts to connect to the unix socket through which dispatcher will communicate with MPV.
// When connection is already estabilished, ErrConnectionInProgress will be returned as connection is an invalid operation while dispatcher is already connected.
// During the process the property observers already registered on command dispatcher are observed.
// It's necessary since either there command dispatcher was reconnected (due to MPV instance closing or other), thus losing all observers,
// or subscriptions occured before connection was made, resulting in no request being sent since there was no MPV instance to receive those requests.
func (cd *commandDispatcher) Connect() error {
	if cd.Connected() {
		return ErrConnectionInProgress
	}

	err := cd.connectToMpvSocket()
	if err != nil {
		return err
	}

	return cd.observeProperties()
}

// Connected notifies whether CommandDispatcher is ready to make requests and observe properties.
func (cd *commandDispatcher) Connected() bool {
	cd.linteningOnSocketLock.RLock()
	defer cd.linteningOnSocketLock.RUnlock()

	return cd.listeningOnSocket
}

// SubscribeToProperty listens to property mpv events.
// Returned id is used as a key to listened property mpv events. Id should be used when unsubscribing. When error is encountered id is useless.
// The channel provided is never closed to enable aggregation from multiple observers.
// However calling unsubscribe will ensure that command dispatcher will stop trying to send on a specified channel.
func (cd *commandDispatcher) SubscribeToProperty(propertyName string, out chan<- ObservePropertyResponse) (int, error) {
	var propertyObserver propertyObserver

	done := make(chan bool)
	propertySubscriptionID := cd.reservePropertySubscriptionID()

	propertyObserver, ok := cd.propertyObserver(propertyName)
	if !ok {
		newObserver, err := cd.addPropertyObserver(propertyName)
		if err != nil {
			return 0, err
		}

		propertyObserver = newObserver
	}

	propertyObserver.subscriptions[propertySubscriptionID] = propertySubscriber{
		propertyChanges: out,
		done:            done,
	}
	responsePayloads := propertyObserver.responsePayloads

	go func() {
		var payload ResponsePayload
		for {
			select {
			case payload = <-responsePayloads:
				out <- ObservePropertyResponse{
					Property: propertyName,
					Response: Response{
						Data: payload.Data,
					},
				}
			case <-done:
				delete(propertyObserver.subscriptions, propertySubscriptionID)
				return
			}
		}
	}()

	return propertySubscriptionID, nil
}

// UnobserveProperty instructs command dispatcher to stop sending updates about property on specified id.
func (cd *commandDispatcher) UnobserveProperty(propertyName string, id int) error {
	propertyObserver, ok := cd.propertyObserver(propertyName)
	if !ok {
		return ErrNoPropertyObserver
	}

	propertySubscription, ok := propertyObserver.subscriptions[id]
	if !ok {
		return ErrNoPropertySubscription
	}

	propertySubscription.done <- true
	return nil
}

// Request is used to send simple Request->response command that is completed after the first response from mpv comes.
func (cd *commandDispatcher) Request(cmd command) (Response, error) {
	var resPayload ResponsePayload
	var result Response

	requestResult := make(chan ResponsePayload)

	requestID := cd.reserveRequestID()
	cd.requests[requestID] = requestResult
	defer delete(cd.requests, requestID)

	err := cd.Dispatch(cmd, requestID)
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

// Dispatch sends a commmand with specified requestID to the mpv using socket.
// Returns error if command was not correctly dispatched.
func (cd *commandDispatcher) Dispatch(cmd command, requestID int) error {
	payload, err := prepareCommandPayload(cmd, requestID)
	if err != nil {
		return err
	}

	written, err := cd.conn.Write(payload)
	if err != nil || len(payload) != written {
		return err
	}

	return nil
}

// Close makes connection by ipc to the mpv closed.
func (cd commandDispatcher) Close() {
	cd.conn.Close()
}

func (cd commandDispatcher) propertyObserver(propertyName string) (propertyObserver, bool) {
	cd.propertyObserversLock.RLock()
	defer cd.propertyObserversLock.RUnlock()

	propertyObserver, ok := cd.propertyObservers[propertyName]
	return propertyObserver, ok
}

func (cd *commandDispatcher) reserveRequestID() int {
	cd.requestIDLock.Lock()
	defer cd.requestIDLock.Unlock()

	requestID := cd.requestID
	cd.requestID++

	return requestID
}

func (cd *commandDispatcher) reservePropertySubscriptionID() int {
	cd.propertySubscriptionIDLock.Lock()
	defer cd.propertySubscriptionIDLock.Unlock()

	propertyObserverID := cd.propertySubscriptionID
	cd.propertySubscriptionID++

	return propertyObserverID
}

func (cd commandDispatcher) listenOnUnixSocket() {
	payloads := make(chan []byte)

	go func() {
		err := readNewlineSeparatedJSONs(cd.conn, payloads)
		if err == io.EOF {
			cd.errLog.Println("connection closed")
		} else {
			cd.errLog.Println("could not read the payload from the connection")
		}

		close(payloads)
	}()

	go func() {
		for {
			payload, more := <-payloads
			if !more {
				break
			}

			var result ResponsePayload
			if len(payload) == 0 {
				continue
			}

			err := json.Unmarshal(payload, &result)
			if err != nil {
				cd.errLog.Printf("could not parse the response: %s\npayload: %s\n", err, payload)
				continue
			}

			if result.Event == propertyChangeEvent {
				propertyObserver, ok := cd.propertyObserver(result.Name)
				if !ok {
					cd.errLog.Printf("observe property event provided to not observed property %s\n", result.Name)
					continue
				}

				propertyObserver.responsePayloads <- result
			} else {
				if result.RequestID == 0 {
					continue
				}

				request, ok := cd.requests[result.RequestID]
				if !ok {
					cd.errLog.Printf("result %d provided to not dispatched request\n", result.RequestID)
					continue
				}

				request <- result
				close(request)
			}
		}

		cd.linteningOnSocketLock.Lock()
		cd.listeningOnSocket = false
		cd.linteningOnSocketLock.Unlock()
	}()

	cd.linteningOnSocketLock.Lock()
	cd.listeningOnSocket = true
	cd.linteningOnSocketLock.Unlock()
}

// IsResultSuccess return whether returned result specifies successful command execution
func IsResultSuccess(result ResponsePayload) bool {
	return result.Err == resultSuccess
}

func waitForSocketConnection(socketPath string) (net.Conn, error) {
	var conn net.Conn
	var err error

	for {
		conn, err = net.Dial(socketType, socketPath)
		if err == nil {
			break
		}

		time.Sleep(1 * time.Second) // mpv takes a longer moment to start listening on the socket, repeat until connection succesful; TODO: add timeout
	}

	return conn, nil // error will be returned on timeout; TODO: add timeout
}

func prepareCommandPayload(cmd command, requestID int) ([]byte, error) {
	var payload []byte
	cmdPayload := commandPayload{
		Command:   cmd.JSONIPCFormat(),
		RequestID: requestID,
	}

	payload, err := json.Marshal(cmdPayload)
	if err != nil {
		return payload, err
	}

	payload = append(payload, newline...)

	return payload, nil
}

func readNewlineSeparatedJSONs(conn net.Conn, out chan<- []byte) error {
	buf := make([]byte, bufSize)
	var acc []byte

	for {
		nRead, err := conn.Read(buf)
		if err != nil {
			return err
		}

		acc = append(acc, buf[:nRead]...)

		newlineIdx := bytes.Index(buf, newline)
		if newlineIdx == -1 {
			continue
		}

		chunks := bytes.Split(acc, newline)
		acc = []byte{}
		for _, chunk := range chunks {
			chunkValid := json.Valid(chunk)
			if chunkValid {
				out <- chunk
			} else {
				acc = append(acc, chunk...)
			}
		}
	}
}
