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

	"github.com/gorilla/mux"
	"github.com/nornforge/norn/pkg/norn"
	"github.com/nornforge/norn/version"
	"go.bug.st/serial"
)

// A ServerConfig holds all information required to serve the REST API
type ServerConfig struct {
	Host          string
	Port          uint16
	maxChannel    int
	deviceVersion versionInfo
	SerialDevice  serial.Port
	mutex         sync.Mutex
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
	Major uint `json:"major"`
	Minor uint `json:"minor"`
	Patch uint `json:"patch"`
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

// newMux creats a new http mux and registers the handler funcs
func newMux(serv *ServerConfig) http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/relay/{index}/value", serv.readRelayHandler).Methods("GET")
	router.HandleFunc("/api/v1/relay/{index}/value", serv.writeRelayHandler).Methods("PUT")

	// API version v2 aka JSON API
	router.HandleFunc("/api/v2/info", serv.info).Methods("GET")

	return router
}

// readRelayHandler reads the given output status from the device and returns the result
// to the caller
func (serv *ServerConfig) readRelayHandler(w http.ResponseWriter, r *http.Request) {
	var channel int = 0
	var err error = nil
	serv.mutex.Lock()
	defer serv.mutex.Unlock()
	vars := mux.Vars(r)
	index := vars["index"]
	if channel, err = serv.getRelayIndex(index); err != nil {
		replyTextContent(w, r, http.StatusBadRequest, err.Error())
		fmt.Println(err.Error())
		return
	}
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
	response := info{
		Server: serverInfo{
			Version: versionInfo{
				Major: version.Major,
				Minor: version.Minor,
				Patch: version.Patch,
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
	serv.mutex.Lock()
	defer serv.mutex.Unlock()
	vars := mux.Vars(r)
	index := vars["index"]

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

	fmt.Printf("The index: %d  and the status to set: <%t> \n", channel, status)

	command := norn.Command{
		Type:    norn.Set,
		Channel: uint(channel),
		Status:  status,
	}
	if _, err = serv.execute(command); err != nil {
		replyTextContent(w, r, http.StatusBadRequest, err.Error())
		fmt.Println(err.Error())
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
	serv.SerialDevice.Write(cmd.Marshal())
	reader := bufio.NewReader(serv.SerialDevice)
	if err := resp.Parse(reader); err != nil {
		return resp, fmt.Errorf("error while parsing the response from the hardware: %s", resp.Message)
	}

	if !resp.Success {
		return resp, fmt.Errorf("error while talking to the hardware >  %s", resp.Message)
	}
	return resp, nil
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
		Addr:         fmt.Sprintf("%s:%d", server.Host, server.Port),
		Handler:      newMux(server),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
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
