// Package server implements the REST API server functionality of the Relay cli
// The main purpose is to provide an Labgrid compliant REST API
// https://github.com/labgrid-project/labgrid/blob/master/labgrid/driver/power/rest.py)
package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/nornforge/norn/pkg/norn"
	"github.com/nornforge/norn/version"
	"go.bug.st/serial"
)

// DefaultTimeout is how long the server waits for the response of the device
// when ServerConfig.Timeout is not set.
const DefaultTimeout = 2 * time.Second

// DefaultQueueTimeout is how long a request waits for the requests ahead of
// it when ServerConfig.QueueTimeout is not set.
const DefaultQueueTimeout = 5 * time.Second

// A ServerConfig holds all information required to serve the REST API
type ServerConfig struct {
	BindAddress string
	Port        uint16
	// Timeout is how long to wait for the response of the device to a
	// command. Zero means DefaultTimeout.
	Timeout       time.Duration
	maxChannel    int
	deviceVersion versionInfo
	SerialDevice  serial.Port
	// QueueTimeout is how long a request waits for the device while other
	// requests use it. Zero means DefaultQueueTimeout.
	QueueTimeout time.Duration
	// busy holds a token while a request uses the device
	busy     chan struct{}
	busyOnce sync.Once
}

// A serverInfo contains all information about the server. This is part of the JSON v2 API
type serverInfo struct {
	Version versionInfo `json:"version"`
}

// A deviceInfo contains all information about the device. This is part of the JSON v2 API
type deviceInfo struct {
	Version versionInfo `json:"version"`
}

// A versionInfo contains all information about a specific components version.
// This is part of the JSON v2 API
type versionInfo struct {
	Major      uint64 `json:"major"`
	Minor      uint64 `json:"minor"`
	Patch      uint64 `json:"patch"`
	PreRelease string `json:"prerelease"`
	MetaData   string `json:"metadata"`
	String     string `json:"string"`
}

// A channelInfo contains all information about the device output channels.
// This is part of the JSON v2 API
type channelInfo struct {
	Min uint `json:"min"`
	Max uint `json:"max"`
}

// A info struct contains all information about the relay service.
// This is part of the JSON v2 API
type info struct {
	Server   serverInfo  `json:"server"`
	Device   deviceInfo  `json:"device"`
	Channels channelInfo `json:"channels"`
}

const minChannel = 1 // the lowe bound of the output channels

// newMux creates a new http mux and registers the handler funcs
func newMux(serv *ServerConfig) http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("GET /api/v1/relay/{index}/value", serv.readRelayHandler)
	router.HandleFunc("PUT /api/v1/relay/{index}/value", serv.writeRelayHandler)

	// API version v2 aka JSON API
	router.HandleFunc("GET /api/v2/info", serv.info)

	return router
}

// readRelayHandler reads the given output status from the device and returns the result
// to the caller
func (serv *ServerConfig) readRelayHandler(w http.ResponseWriter, r *http.Request) {
	var channel int = 0
	var err error = nil
	index := r.PathValue("index")
	if channel, err = serv.getRelayIndex(index); err != nil {
		replyTextContent(w, r, http.StatusBadRequest, err.Error())
		fmt.Println(err.Error())
		return
	}
	// Only once the request is known to be valid, it waits for the device
	if !serv.acquireDevice(w, r) {
		return
	}
	defer serv.releaseDevice()
	command := norn.Command{
		Type:    norn.Get,
		Channel: uint(channel),
	}

	response, err := serv.execute(command)
	if err != nil {
		replyTextContent(w, r, http.StatusBadRequest, err.Error())
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("The index: %d  and the status: <%t> \n", channel, response.Status)

	replyTextContent(w, r, http.StatusOK, fmt.Sprintf("%d", serv.responseStatusToInt(response)))
}

// info provides the information about the device and the service to the caller
func (serv *ServerConfig) info(w http.ResponseWriter, r *http.Request) {
	version := version.GetProgramVersion()
	response := info{
		Server: serverInfo{
			Version: versionInfo{
				Major:      version.Major(),
				Minor:      version.Minor(),
				Patch:      version.Patch(),
				MetaData:   version.Metadata(),
				PreRelease: version.Prerelease(),
				String:     version.String(),
			},
		},
		Device: deviceInfo{
			Version: versionInfo{
				Major: serv.deviceVersion.Major,
				Minor: serv.deviceVersion.Minor,
				Patch: serv.deviceVersion.Patch,
			},
		},
		Channels: channelInfo{
			Max: uint(serv.maxChannel),
			Min: minChannel,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	status := http.StatusOK
	data, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("Error occured while creating JSON response: %s\n", err.Error())
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)
	w.Write(data)
}

// responseStatusToInt converts the response from the device to an integer
func (serv *ServerConfig) responseStatusToInt(response norn.Response) int {
	status := 0
	if response.Status {
		status = 1
	}
	return status
}

// writeRelayHandler writes the given output status to the device
func (serv *ServerConfig) writeRelayHandler(w http.ResponseWriter, r *http.Request) {
	var channel int = 0
	var err error = nil
	status := false
	index := r.PathValue("index")

	if channel, err = serv.getRelayIndex(index); err != nil {
		replyTextContent(w, r, http.StatusBadRequest, err.Error())
		fmt.Println(err.Error())
		return
	}

	if status, err = serv.readStatus(r); err != nil {
		replyTextContent(w, r, http.StatusBadRequest, err.Error())
		fmt.Println(err.Error())
		return
	}

	// Only once the request is known to be valid, and its body has been read,
	// it waits for the device: net/http notices a client that went away only
	// after the body was read.
	if !serv.acquireDevice(w, r) {
		return
	}
	defer serv.releaseDevice()

	fmt.Printf("The index: %d  and the status to set: <%t> \n", channel, status)

	command := norn.Command{
		Type:    norn.Set,
		Channel: uint(channel),
		Status:  status,
	}
	if _, err = serv.execute(command); err != nil {
		replyTextContent(w, r, http.StatusBadRequest, err.Error())
		fmt.Println(err.Error())
		return
	}
	reply(w, r, http.StatusOK)
}

// getRelayIndex converts and checks the input string
//
// On success getRelayIndex returns err == nil and the integer index of the output
func (serv *ServerConfig) getRelayIndex(index string) (int, error) {
	channel, err := strconv.Atoi(string(index))
	if err != nil || ((channel < minChannel) || (channel > serv.maxChannel)) {
		return 0, fmt.Errorf("invalid index provided: %s [%d,%d]",
			index,
			minChannel,
			serv.maxChannel)
	}
	return channel, nil
}

// readStatus parses the http request for the PIN status to be set
//
// On success err == nil and the status to be set
func (serv *ServerConfig) readStatus(r *http.Request) (bool, error) {
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		return false, err
	}
	status := true
	intStatus, err := strconv.Atoi(string(bytes))
	if err != nil {
		return false, err
	}
	if intStatus != 0 && intStatus != 1 {
		return false, fmt.Errorf("value out of range %d [0,1]", intStatus)
	}
	if intStatus == 0 {
		status = false
	}
	return status, nil
}

// execute will send and execute a command to the device
//
// A successful execution will return err == nil and the response
func (serv *ServerConfig) execute(cmd norn.Command) (norn.Response, error) {
	resp := norn.Response{
		Success: false,
	}
	// A response that arrived after an earlier command timed out would
	// otherwise be taken as the response to this command.
	if err := serv.SerialDevice.ResetInputBuffer(); err != nil {
		return resp, fmt.Errorf("error while clearing the input of the hardware: %w", err)
	}
	if _, err := serv.SerialDevice.Write(cmd.Marshal()); err != nil {
		return resp, fmt.Errorf("error while sending the command to the hardware: %w", err)
	}
	timeout := serv.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	reader := bufio.NewReader(newDeadlineReader(serv.SerialDevice, timeout))
	if err := resp.Parse(reader); err != nil {
		return resp, fmt.Errorf("error while parsing the response from the hardware: %w", err)
	}

	if !resp.Success {
		return resp, fmt.Errorf("error while talking to the hardware >  %s", resp.Message)
	}
	return resp, nil
}

// deadlineReader reads from a serial port until a deadline. All reads
// together have to finish before it, so a device that does not answer, or
// stops in the middle of a response, does not block the server, and with it
// all other requests, forever.
type deadlineReader struct {
	port     serial.Port
	timeout  time.Duration
	deadline time.Time
}

// newDeadlineReader returns a reader whose deadline is the timeout from now.
func newDeadlineReader(port serial.Port, timeout time.Duration) *deadlineReader {
	return &deadlineReader{port: port, timeout: timeout, deadline: time.Now().Add(timeout)}
}

func (r *deadlineReader) Read(p []byte) (int, error) {
	for {
		remaining := time.Until(r.deadline)
		if remaining <= 0 {
			return 0, fmt.Errorf("no response from the device within %s", r.timeout)
		}
		if err := r.port.SetReadTimeout(remaining); err != nil {
			return 0, err
		}
		// On a timeout the port returns no data and no error.
		if n, err := r.port.Read(p); n > 0 || err != nil {
			return n, err
		}
	}
}

// acquireDevice waits until no other request uses the device and reserves
// it for this request. It gives up when the client goes away, so the command
// of a request nobody waits for any more is not executed, and after the queue
// timeout, replying 503. Without that limit, requests piled up behind a
// device that does not answer would only run after their clients, or the
// write timeout of the server, gave up.
//
// When acquireDevice returns true, the caller must call releaseDevice.
func (serv *ServerConfig) acquireDevice(w http.ResponseWriter, r *http.Request) bool {
	serv.busyOnce.Do(func() { serv.busy = make(chan struct{}, 1) })
	wait := serv.queueTimeout()
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case serv.busy <- struct{}{}:
	case <-r.Context().Done():
		fmt.Printf("%s %s: the client went away while waiting for the device\n", r.Method, r.URL.Path)
		return false
	case <-timer.C:
		message := fmt.Sprintf("the device is busy: other requests used it for more than %s", wait)
		replyTextContent(w, r, http.StatusServiceUnavailable, message)
		fmt.Println(message)
		return false
	}
	// Both may have been ready at the same time
	if r.Context().Err() != nil {
		serv.releaseDevice()
		fmt.Printf("%s %s: the client went away while waiting for the device\n", r.Method, r.URL.Path)
		return false
	}
	return true
}

// releaseDevice frees the device for the next request.
func (serv *ServerConfig) releaseDevice() {
	<-serv.busy
}

func (serv *ServerConfig) queueTimeout() time.Duration {
	if serv.QueueTimeout <= 0 {
		return DefaultQueueTimeout
	}
	return serv.QueueTimeout
}

func (serv *ServerConfig) timeout() time.Duration {
	if serv.Timeout <= 0 {
		return DefaultTimeout
	}
	return serv.Timeout
}

// reply creates a http response
// This is a internal helper function
func reply(w http.ResponseWriter, r *http.Request, status int) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
}

// replyTextContent creates a text http response
// This is a internal helper function
func replyTextContent(w http.ResponseWriter, r *http.Request, status int, content string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	w.Write([]byte(content))
}

// Serve waits for incoming http requests and handles them
// This is a blocking function. In case of an error it returns
func Serve(server *ServerConfig) error {
	if err := server.getMaxChannels(); err != nil {
		return err
	}
	if err := server.getDeviceVersion(); err != nil {
		return err
	}
	s := &http.Server{
		Addr:        fmt.Sprintf("%s:%d", server.BindAddress, server.Port),
		Handler:     newMux(server),
		ReadTimeout: 10 * time.Second,
		// A request may wait in the queue and then for the device; the
		// response must still get through after that.
		WriteTimeout: server.queueTimeout() + server.timeout() + 5*time.Second,
	}
	return s.ListenAndServe()
}

// getDeviceVersion retrieves the version from the device
//
// On success getDeviceVersion return err == nil
func (serv *ServerConfig) getDeviceVersion() error {
	command := norn.Command{
		Type: norn.Version,
	}
	response, err := serv.execute(command)
	if err != nil {
		return err
	}
	fmt.Sscanf(response.Message,
		"v%d.%d.%d",
		&serv.deviceVersion.Major,
		&serv.deviceVersion.Minor,
		&serv.deviceVersion.Patch,
	)
	return nil
}

// getMaxChannels retrieves the maximum of available Channels from the device
//
// On success getMaxChannels returns err == nil
func (serv *ServerConfig) getMaxChannels() error {
	command := norn.Command{
		Type: norn.MaxChannels,
	}
	response, err := serv.execute(command)
	if err != nil {
		return err
	}
	serv.maxChannel = response.MaxChannel
	return nil
}
