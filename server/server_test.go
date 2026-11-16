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
	"time"

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
	// silentNext makes the device not answer the next command; its response
	// is kept in late until deliverLate is called.
	silentNext bool
	late       []byte
	// strayNext is sent right before the response to the next command, like
	// a response to an earlier command that arrives late.
	strayNext []byte
	// cutNext makes the device send only the first half of its next response.
	cutNext     bool
	readTimeout time.Duration
	resets      int
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
	handled := d.handle(command)
	response := handled.Marshal()
	d.output.Write(d.strayNext)
	d.strayNext = nil
	switch {
	case d.silentNext:
		d.silentNext = false
		d.late = response
	case d.cutNext:
		d.cutNext = false
		d.output.Write(response[:len(response)/2])
	default:
		d.output.Write(response)
	}
	return len(p), nil
}

// Read behaves like a serial port with a read timeout: without data, it
// returns no data and no error once the timeout has passed.
func (d *fakeDevice) Read(p []byte) (int, error) {
	d.mutex.Lock()
	if d.output.Len() > 0 {
		defer d.mutex.Unlock()
		return d.output.Read(p)
	}
	wait := min(d.readTimeout, 5*time.Millisecond)
	d.mutex.Unlock()
	time.Sleep(wait)
	return 0, nil
}

func (d *fakeDevice) SetReadTimeout(timeout time.Duration) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.readTimeout = timeout
	return nil
}

func (d *fakeDevice) ResetInputBuffer() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.output.Reset()
	d.resets++
	return nil
}

// deliverLate sends the response the device held back for silentNext.
func (d *fakeDevice) deliverLate() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.output.Write(d.late)
	d.late = nil
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

// testTimeout is how long the server waits for the fake device in tests.
const testTimeout = 200 * time.Millisecond

// startServer returns a test server in front of a fake device, set up the
// way Serve does it.
func startServer(t *testing.T, device *fakeDevice) *httptest.Server {
	t.Helper()
	config := &ServerConfig{SerialDevice: device, Timeout: testTimeout}
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
		// The reason must reach the caller instead of an empty message. The
		// parser skips bytes that are not a frame, so it runs out of data.
		require.Contains(t, body, "error while parsing the response", method)
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
	require.Equal(t, versionInfo{Major: 1, Minor: 2, Patch: 3, String: "1.2.3"}, got.Device.Version)
	require.Equal(t, uint(1), got.Channels.Min)
	require.Equal(t, uint(4), got.Channels.Max)
}

// A firmware built from a branch reports a pre-release, such as
// v0.1.0-resync. It must not be cut off after the patch number.
func TestInfoWithADevicePreRelease(t *testing.T) {
	server := startServer(t, newFakeDevice(8, "v0.1.0-resync+build.7"))

	status, body := request(t, http.MethodGet, server.URL+"/api/v2/info", "")
	require.Equal(t, http.StatusOK, status)

	var got info
	require.NoError(t, json.Unmarshal([]byte(body), &got))
	require.Equal(t, versionInfo{
		Major:      0,
		Minor:      1,
		Patch:      0,
		PreRelease: "resync",
		MetaData:   "build.7",
		String:     "0.1.0-resync+build.7",
	}, got.Device.Version)
}

func TestInfoWithADeviceVersionThatIsNotASemanticVersion(t *testing.T) {
	server := startServer(t, newFakeDevice(8, "development build"))

	status, body := request(t, http.MethodGet, server.URL+"/api/v2/info", "")
	require.Equal(t, http.StatusOK, status)

	var got info
	require.NoError(t, json.Unmarshal([]byte(body), &got))
	require.Equal(t, versionInfo{String: "development build"}, got.Device.Version)
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

func TestASilentDeviceTimesOut(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		device := newFakeDevice(8, "v1.2.3")
		server := startServer(t, device)

		device.mutex.Lock()
		device.silentNext = true
		device.mutex.Unlock()

		start := time.Now()
		status, body := request(t, method, relayURL(server, "1"), "1")
		require.Equal(t, http.StatusBadRequest, status, method)
		require.Contains(t, body, "no response from the device within 200ms", method)
		require.Less(t, time.Since(start), 2*time.Second, method)

		// The server is not stuck: the next request works again
		status, _ = request(t, method, relayURL(server, "1"), "1")
		require.Equal(t, http.StatusOK, status, method)
	}
}

func TestAResponseThatStopsHalfwayTimesOut(t *testing.T) {
	device := newFakeDevice(8, "v1.2.3")
	server := startServer(t, device)

	device.mutex.Lock()
	device.cutNext = true
	device.mutex.Unlock()

	status, body := request(t, http.MethodGet, relayURL(server, "1"), "")
	require.Equal(t, http.StatusBadRequest, status)
	require.Contains(t, body, "no response from the device within", body)

	status, body = request(t, http.MethodGet, relayURL(server, "1"), "")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "0", body)
}

// A response that arrives after its request timed out must not be taken as
// the response to the next request, or the server stays one response behind.
func TestALateResponseIsNotTakenForTheNextOne(t *testing.T) {
	device := newFakeDevice(8, "v1.2.3")
	server := startServer(t, device)

	status, _ := request(t, http.MethodPut, relayURL(server, "2"), "1")
	require.Equal(t, http.StatusOK, status)

	device.mutex.Lock()
	device.silentNext = true
	device.mutex.Unlock()
	status, _ = request(t, http.MethodGet, relayURL(server, "2"), "")
	require.Equal(t, http.StatusBadRequest, status)

	device.deliverLate() // "relay 2 is on" arrives too late

	for i := 0; i < 3; i++ {
		status, body := request(t, http.MethodGet, relayURL(server, "1"), "")
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "0", body, "request %d", i)
	}
}

func TestStartupFailsWhenTheDeviceDoesNotAnswer(t *testing.T) {
	device := newFakeDevice(8, "v1.2.3")
	device.silentNext = true
	config := &ServerConfig{SerialDevice: device, Timeout: testTimeout}

	start := time.Now()
	err := config.getMaxChannels()
	require.ErrorContains(t, err, "no response from the device")
	require.Less(t, time.Since(start), 2*time.Second)
}

func TestTheDefaultTimeoutIsUsedWhenNoneIsSet(t *testing.T) {
	device := newFakeDevice(8, "v1.2.3")
	config := &ServerConfig{SerialDevice: device}
	require.NoError(t, config.getMaxChannels())

	device.silentNext = true
	start := time.Now()
	err := config.getDeviceVersion()
	require.ErrorContains(t, err, fmt.Sprintf("within %s", DefaultTimeout))
	require.GreaterOrEqual(t, time.Since(start), DefaultTimeout)
}

// A response to an earlier command that arrives after the input was cleared,
// right before the real response, is skipped.
func TestAResponseToAnEarlierCommandIsSkipped(t *testing.T) {
	device := newFakeDevice(8, "v1.2.3")
	server := startServer(t, device)
	stray := norn.Response{Success: true, Channel: 2, Status: true}

	device.mutex.Lock()
	device.strayNext = stray.Marshal()
	device.mutex.Unlock()
	status, body := request(t, http.MethodGet, relayURL(server, "1"), "")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "0", body)

	device.mutex.Lock()
	device.strayNext = stray.Marshal()
	device.mutex.Unlock()
	status, body = request(t, http.MethodGet, server.URL+"/api/v2/info", "")
	require.Equal(t, http.StatusOK, status, body)
}
