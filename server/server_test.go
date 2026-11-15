package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/nornforge/norn/pkg/norn"
	"github.com/stretchr/testify/require"
	"go.bug.st/serial"
)

// fakeDevice stands in for the serial port of a device running the firmware.
// Every command written to it is answered the way the firmware does.
type fakeDevice struct {
	serial.Port // the methods the server does not use stay unimplemented

	mutex    sync.Mutex
	channels []bool
	version  string
	output   bytes.Buffer
	commands []norn.Command
	// failNext makes the device answer the next command with this error.
	failNext string
	// garbleNext makes the device answer the next command with bytes that
	// are not a valid frame.
	garbleNext bool
	// failWrite makes the next write to the device fail with this error.
	failWrite error
}

func newFakeDevice(numChannels int, version string) *fakeDevice {
	return &fakeDevice{channels: make([]bool, numChannels), version: version}
}

func (d *fakeDevice) Write(p []byte) (int, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.failWrite != nil {
		err := d.failWrite
		d.failWrite = nil
		return 0, err
	}
	command := norn.Command{}
	if err := command.Parse(bufio.NewReader(bytes.NewReader(p))); err != nil {
		return 0, err
	}
	d.commands = append(d.commands, command)
	if d.garbleNext {
		d.garbleNext = false
		d.output.WriteString("panic: runtime error")
		return len(p), nil
	}
	response := d.handle(command)
	d.output.Write(response.Marshal())
	return len(p), nil
}

func (d *fakeDevice) Read(p []byte) (int, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.output.Read(p)
}

func (d *fakeDevice) handle(command norn.Command) norn.Response {
	if d.failNext != "" {
		message := d.failNext
		d.failNext = ""
		return norn.Response{Success: false, Message: message}
	}
	switch command.Type {
	case norn.Get, norn.Set:
		if command.Channel < 1 || command.Channel > uint(len(d.channels)) {
			return norn.Response{Success: false, Message: fmt.Sprintf("Invalid index provided: %d", command.Channel)}
		}
		if command.Type == norn.Set {
			d.channels[command.Channel-1] = command.Status
		}
		return norn.Response{Success: true, Channel: command.Channel, Status: d.channels[command.Channel-1]}
	case norn.Version:
		return norn.Response{Success: true, Message: d.version}
	case norn.MaxChannels:
		return norn.Response{Success: true, MaxChannel: len(d.channels)}
	}
	return norn.Response{Success: false, Message: "unknown command"}
}

func (d *fakeDevice) channel(index int) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.channels[index-1]
}

func (d *fakeDevice) numCommands() int {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return len(d.commands)
}

// lockedBuffer is a bytes.Buffer that is safe for concurrent use.
type lockedBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.String()
}

// startServer returns a test server in front of a fake device, set up the
// way Serve does it.
func startServer(t *testing.T, device *fakeDevice) *httptest.Server {
	t.Helper()
	config := &ServerConfig{SerialDevice: device}
	require.NoError(t, config.getMaxChannels())
	require.NoError(t, config.getDeviceVersion())
	server := httptest.NewUnstartedServer(newMux(config))
	// net/http reports misuse of the ResponseWriter, such as writing the
	// status twice, through this log. A handler must not cause any of it.
	serverLog := &lockedBuffer{}
	server.Config.ErrorLog = log.New(serverLog, "", 0)
	server.Start()
	t.Cleanup(func() {
		server.Close()
		require.Empty(t, serverLog.String(), "errors logged by the HTTP server")
	})
	return server
}

func request(t *testing.T, method, url, body string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(content)
}

func relayURL(server *httptest.Server, index string) string {
	return fmt.Sprintf("%s/api/v1/relay/%s/value", server.URL, index)
}

func TestRelayCanBeSwitchedAndRead(t *testing.T) {
	device := newFakeDevice(8, "v1.2.3")
	server := startServer(t, device)

	for index := 1; index <= 8; index++ {
		url := relayURL(server, fmt.Sprint(index))

		status, body := request(t, http.MethodGet, url, "")
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "0", body)

		status, _ = request(t, http.MethodPut, url, "1")
		require.Equal(t, http.StatusOK, status)
		require.True(t, device.channel(index), "channel %d on the device", index)

		status, body = request(t, http.MethodGet, url, "")
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "1", body)

		status, _ = request(t, http.MethodPut, url, "0")
		require.Equal(t, http.StatusOK, status)
		require.False(t, device.channel(index), "channel %d on the device", index)
	}
}

func TestSwitchingOneRelayLeavesTheOthersAlone(t *testing.T) {
	device := newFakeDevice(8, "v1.2.3")
	server := startServer(t, device)

	status, _ := request(t, http.MethodPut, relayURL(server, "3"), "1")
	require.Equal(t, http.StatusOK, status)

	for index := 1; index <= 8; index++ {
		require.Equal(t, index == 3, device.channel(index), "channel %d", index)
	}
}

func TestInvalidRelayIndexIsRejectedWithoutAskingTheDevice(t *testing.T) {
	device := newFakeDevice(8, "v1.2.3")
	server := startServer(t, device)
	before := device.numCommands()

	for _, index := range []string{"0", "9", "-1", "abc", "1.5"} {
		for _, method := range []string{http.MethodGet, http.MethodPut} {
			status, body := request(t, method, relayURL(server, index), "1")
			require.Equal(t, http.StatusBadRequest, status, "%s index %q", method, index)
			require.Contains(t, body, "invalid index", "%s index %q", method, index)
		}
	}
	require.Equal(t, before, device.numCommands(), "commands sent to the device")
}

func TestInvalidRelayValueIsRejected(t *testing.T) {
	device := newFakeDevice(8, "v1.2.3")
	server := startServer(t, device)
	before := device.numCommands()

	for _, value := range []string{"2", "-1", "on", "", "1 "} {
		status, _ := request(t, http.MethodPut, relayURL(server, "1"), value)
		require.Equal(t, http.StatusBadRequest, status, "value %q", value)
	}
	require.Equal(t, before, device.numCommands(), "commands sent to the device")
	require.False(t, device.channel(1))
}

func TestAnErrorFromTheDeviceIsReported(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		device := newFakeDevice(8, "v1.2.3")
		server := startServer(t, device)

		device.mutex.Lock()
		device.failNext = "relay stuck"
		device.mutex.Unlock()

		status, body := request(t, method, relayURL(server, "1"), "1")
		require.Equal(t, http.StatusBadRequest, status, method)
		require.Contains(t, body, "relay stuck", method)

		// The next request works again
		status, _ = request(t, method, relayURL(server, "1"), "1")
		require.Equal(t, http.StatusOK, status, method)
	}
}

func TestAResponseThatIsNotAValidFrameIsReported(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		device := newFakeDevice(8, "v1.2.3")
		server := startServer(t, device)

		device.mutex.Lock()
		device.garbleNext = true
		device.mutex.Unlock()

		status, body := request(t, method, relayURL(server, "1"), "1")
		require.Equal(t, http.StatusBadRequest, status, method)
		// The reason must reach the caller instead of an empty message
		require.Contains(t, body, "invalid marker", method)
	}
}

func TestAFailedWriteToTheDeviceIsReported(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		device := newFakeDevice(8, "v1.2.3")
		server := startServer(t, device)
		before := device.numCommands()

		device.mutex.Lock()
		device.failWrite = errors.New("port has been unplugged")
		device.mutex.Unlock()

		status, body := request(t, method, relayURL(server, "1"), "1")
		require.Equal(t, http.StatusBadRequest, status, method)
		// The I/O error must reach the caller, not a parse error that follows from it
		require.Contains(t, body, "error while sending the command", method)
		require.Contains(t, body, "port has been unplugged", method)
		require.NotContains(t, body, "parsing", method)
		require.Equal(t, before, device.numCommands(), "commands that reached the device")
		require.False(t, device.channel(1), method)
	}
}

func TestUnsupportedMethodsAreRejected(t *testing.T) {
	server := startServer(t, newFakeDevice(8, "v1.2.3"))

	for _, method := range []string{http.MethodPost, http.MethodDelete} {
		status, _ := request(t, method, relayURL(server, "1"), "1")
		require.Equal(t, http.StatusMethodNotAllowed, status, method)
	}
}

func TestInfo(t *testing.T) {
	server := startServer(t, newFakeDevice(4, "v1.2.3"))

	status, body := request(t, http.MethodGet, server.URL+"/api/v2/info", "")
	require.Equal(t, http.StatusOK, status)

	var got info
	require.NoError(t, json.Unmarshal([]byte(body), &got))
	require.Equal(t, uint64(1), got.Device.Version.Major)
	require.Equal(t, uint64(2), got.Device.Version.Minor)
	require.Equal(t, uint64(3), got.Device.Version.Patch)
	require.Equal(t, uint(1), got.Channels.Min)
	require.Equal(t, uint(4), got.Channels.Max)
}

func TestInfoWithADeviceVersionThatIsNotASemanticVersion(t *testing.T) {
	server := startServer(t, newFakeDevice(8, "development build"))

	status, body := request(t, http.MethodGet, server.URL+"/api/v2/info", "")
	require.Equal(t, http.StatusOK, status)

	var got info
	require.NoError(t, json.Unmarshal([]byte(body), &got))
	require.Equal(t, uint64(0), got.Device.Version.Major)
	require.Equal(t, uint(8), got.Channels.Max)
}

func TestConcurrentRequestsAreSerialized(t *testing.T) {
	device := newFakeDevice(8, "v1.2.3")
	server := startServer(t, device)

	var wg sync.WaitGroup
	failures := make(chan string, 8*20)
	for index := 1; index <= 8; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			url := relayURL(server, fmt.Sprint(index))
			for i := 0; i < 20; i++ {
				want := fmt.Sprint(i % 2)
				if status, _ := request(t, http.MethodPut, url, want); status != http.StatusOK {
					failures <- fmt.Sprintf("put %d: status %d", index, status)
					return
				}
				if status, body := request(t, http.MethodGet, url, ""); status != http.StatusOK || body != want {
					failures <- fmt.Sprintf("get %d: status %d body %q, want %q", index, status, body, want)
					return
				}
			}
		}(index)
	}
	wg.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
}

// A URL with a double slash, as a client produces from a base URL with a
// trailing slash, is redirected with 307: the client repeats the PUT with
// its body instead of turning it into a GET that switches nothing.
func TestADoubleSlashIsRedirectedWithoutLosingThePut(t *testing.T) {
	device := newFakeDevice(8, "v1.2.3")
	server := startServer(t, device)

	status, _ := request(t, http.MethodPut, server.URL+"//api/v1/relay/2/value", "1")
	require.Equal(t, http.StatusOK, status)
	require.True(t, device.channel(2))
}

func TestUnknownPathsAreNotFound(t *testing.T) {
	server := startServer(t, newFakeDevice(8, "v1.2.3"))

	for _, path := range []string{"/", "/api/v1/relay/1", "/api/v1/relay/1/value/", "/api/v2/info/"} {
		status, _ := request(t, http.MethodGet, server.URL+path, "")
		require.Equal(t, http.StatusNotFound, status, path)
	}
}
