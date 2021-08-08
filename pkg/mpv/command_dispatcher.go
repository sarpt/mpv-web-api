package mpv

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	conn                       net.Conn
	connectionTimeout          time.Duration
	errLog                     *log.Logger
	listeningOnSocket          bool
	listeningOnSocketLock      *sync.RWMutex
	outLog                     *log.Logger
	propertyObservers          map[string]propertyObserver
	propertyObserversLock      *sync.RWMutex
	propertySubscriptionID     int
	propertySubscriptionIDLock *sync.Mutex
	requests                   map[int]chan ResponsePayload
	requestID                  int
	requestIDLock              *sync.Mutex
	socketPath                 string
}

type propertyObserver struct {
	responsePayloads chan ResponsePayload
	subscriptions    map[int]propertySubscriber
}

type propertySubscriber struct {
	propertyChanges chan<- ObservePropertyResponse
	done            chan bool
}

type commandDispatcherConfig struct {
	connectionTimeout time.Duration
	errWriter         io.Writer
	socketPath        string
	outWriter         io.Writer
}

// newCommandDispatcher returns dispatcher connected to the socket.
func newCommandDispatcher(cfg commandDispatcherConfig) *commandDispatcher {
	return &commandDispatcher{
		connectionTimeout:          cfg.connectionTimeout,
		errLog:                     log.New(cfg.errWriter, commandDispatcherLogPrefix, log.LstdFlags),
		listeningOnSocket:          false,
		listeningOnSocketLock:      &sync.RWMutex{},
		outLog:                     log.New(cfg.outWriter, commandDispatcherLogPrefix, log.LstdFlags),
		propertyObservers:          make(map[string]propertyObserver),
		propertyObserversLock:      &sync.RWMutex{},
		propertySubscriptionID:     1,
		propertySubscriptionIDLock: &sync.Mutex{},
		requests:                   make(map[int]chan ResponsePayload),
		requestID:                  1,
		requestIDLock:              &sync.Mutex{},
		socketPath:                 cfg.socketPath,
	}
}

// Close makes connection by ipc to the mpv closed.
func (cd commandDispatcher) Close() {
	cd.conn.Close()
}

// Connect attempts to connect to the unix socket through which dispatcher will communicate with MPV.
// When connection is already estabilished, ErrConnectionInProgress will be returned,
// as connection is an invalid operation while dispatcher is already connected.
func (cd *commandDispatcher) Connect() error {
	if cd.Connected() {
		return ErrConnectionInProgress
	}

	cd.outLog.Printf("trying to connect to mpv socket at '%s' with timeout: %f seconds\n", cd.socketPath, cd.connectionTimeout.Seconds())
	err := cd.connectToMpvSocket()
	if err != nil {
		cd.errLog.Printf("could not connect to socket due to error: %s\n", err)
		return err
	}
	cd.outLog.Printf("connected to socket at '%s'\n", cd.socketPath)

	return nil
}

// Connected informs whether CommandDispatcher is ready to make requests and observe properties.
func (cd *commandDispatcher) Connected() bool {
	cd.listeningOnSocketLock.RLock()
	defer cd.listeningOnSocketLock.RUnlock()

	return cd.listeningOnSocket
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

// Serve instructs command dispatcher to serve communication handling with mpv through the socket -
// this involves dispatching requests and property observing.
// During the process property observers already registered on command dispatcher are observed.
// It's necessary since either command dispatcher could be reconnected (due to MPV instance closing etc.), thus losing all observers,
// or subscriptions occured before connection was made, resulting in no request being sent since there was no MPV instance to receive those requests.
// Property observing errors are non fatal to serving of CommandDispatcher, as such no errors interecepting is done on "observerProperties".
func (cd *commandDispatcher) Serve() error {
	go cd.observeProperties()
	cd.outLog.Printf("listening on unix socket at '%s'\n", cd.socketPath)

	return cd.listenOnUnixSocket()
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

func (cd *commandDispatcher) connectToMpvSocket() error {
	conn, err := waitForSocketConnection(cd.socketPath, cd.connectionTimeout)
	if err != nil {
		return err
	}

	cd.conn = conn

	return nil
}

func (cd *commandDispatcher) distributeResponse(payload []byte) error {
	result, err := getResponsePayload(payload)
	if err != nil {
		return fmt.Errorf("could not parse payload '%s' as ResponsePayload: %w", payload, err)
	}

	if result.Event == propertyChangeEvent {
		propertyObserver, ok := cd.propertyObserver(result.Name)
		if !ok {
			return fmt.Errorf("observe property event provided to not observed property %s", result.Name)
		}

		propertyObserver.responsePayloads <- result
	} else {
		if result.RequestID == 0 {
			return fmt.Errorf("result provided without RequestID")
		}

		request, ok := cd.requests[result.RequestID]
		if !ok {
			return fmt.Errorf("result %d provided to not dispatched request", result.RequestID)
		}

		request <- result
		close(request)
	}

	return nil
}

func (cd *commandDispatcher) listenOnUnixSocket() error {
	cd.setListeningOnSocket(true)
	defer cd.setListeningOnSocket(false)

	payloads := make(chan []byte)

	// TODO: goroutine below can be replaced with synchronous code (and put inside 'for' below) however,
	// it would be necessary to keep the state between calls to "readNewlineSeparatedJSONs",
	// as the unfinished JSON chunk read in buffer should be used in the next read
	// - probably could be solved by writing a struct holding the buffer between calls.
	go func() {
		err := readNewlineSeparatedJSONs(cd.conn, payloads)
		if err == io.EOF {
			cd.outLog.Println("connection closed")
		} else {
			cd.errLog.Println("could not read the payload from the connection")
		}

		close(payloads)
	}()

	for {
		payload, more := <-payloads
		if !more {
			break
		}

		if len(payload) == 0 {
			continue
		}

		err := cd.distributeResponse(payload)
		if err != nil {
			cd.errLog.Printf("could not distribute response: %s\n", err)
		}
	}

	return nil
}

func (cd *commandDispatcher) observeProperties() {
	cd.propertyObserversLock.RLock()
	defer cd.propertyObserversLock.RUnlock()

	for propertyName := range cd.propertyObservers {
		err := cd.observeProperty(propertyName)
		if err != nil {
			cd.errLog.Printf("could not observe property '%s' due to error: %s", propertyName, err)
		}
	}
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

func (cd *commandDispatcher) setListeningOnSocket(listening bool) {
	cd.listeningOnSocketLock.Lock()
	defer cd.listeningOnSocketLock.Unlock()

	cd.listeningOnSocket = listening
}

// IsResultSuccess return whether returned result specifies successful command execution.
func IsResultSuccess(result ResponsePayload) bool {
	return result.Err == resultSuccess
}

func waitForSocketConnection(socketPath string, timeout time.Duration) (net.Conn, error) {
	var conn net.Conn
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	connection := make(chan net.Conn)
	go dialSocket(socketType, socketPath, connection)

	select {
	case conn = <-connection:
		return conn, nil
	case <-ctx.Done():
		return conn, ctx.Err()
	}
}

func dialSocket(socketType string, socketPath string, done chan<- net.Conn) {
	for {
		conn, err := net.Dial(socketType, socketPath)
		if err == nil {
			done <- conn
		}

		// mpv takes a moment (up to a few seconds) to start listening on the socket, repeat until connection successful.
		time.Sleep(1 * time.Second)
	}
}

func getResponsePayload(payload []byte) (ResponsePayload, error) {
	var result ResponsePayload
	err := json.Unmarshal(payload, &result)
	if err != nil {
		return result, fmt.Errorf("could not parse the response JSON as ResponsePayload: %w", err)
	}

	formatNodeConverter, ok := FormatNodeConverters[result.Name]
	if !ok {
		return result, err
	}

	convertedData, err := formatNodeConverter(result.Data)
	if err != nil {
		return result, fmt.Errorf("could not parse the format node data for the response: %w", err)
	}
	result.Data = convertedData

	return result, err
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
