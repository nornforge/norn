package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// startQueueServer starts a server like startServer, with the given queue
// timeout. The device timeout is testTimeout.
func startQueueServer(t *testing.T, device *fakeDevice, queueTimeout time.Duration) *httptest.Server {
	t.Helper()
	config := &ServerConfig{SerialDevice: device, Timeout: testTimeout, QueueTimeout: queueTimeout}
	require.NoError(t, config.getMaxChannels())
	server := httptest.NewServer(newMux(config))
	t.Cleanup(server.Close)
	return server
}

// occupyDevice sends a request the device does not answer, so it uses the
// device for testTimeout. It returns once the device got the command, and
// the returned channel is closed when the request is done.
func occupyDevice(t *testing.T, server *httptest.Server, device *fakeDevice) <-chan struct{} {
	t.Helper()
	device.mutex.Lock()
	device.silentNext = true
	device.mutex.Unlock()
	before := device.numCommands()

	done := make(chan struct{})
	go func() {
		defer close(done)
		resp, err := http.Get(relayURL(server, "1"))
		if err == nil {
			resp.Body.Close()
		}
	}()
	require.Eventually(t, func() bool { return device.numCommands() == before+1 },
		time.Second, time.Millisecond, "the device never got the first command")
	return done
}

func TestARequestWaitsForTheOneAheadOfIt(t *testing.T) {
	device := newFakeDevice(8, "v1.2.3")
	server := startQueueServer(t, device, 5*time.Second)
	done := occupyDevice(t, server, device)

	status, body := request(t, http.MethodPut, relayURL(server, "2"), "1")
	require.Equal(t, http.StatusOK, status, body)
	require.True(t, device.channel(2))
	<-done
}

func TestARequestGivesUpWhenTheDeviceStaysBusy(t *testing.T) {
	device := newFakeDevice(8, "v1.2.3")
	server := startQueueServer(t, device, 50*time.Millisecond)
	done := occupyDevice(t, server, device)
	before := device.numCommands()

	start := time.Now()
	status, body := request(t, http.MethodPut, relayURL(server, "2"), "1")
	require.Equal(t, http.StatusServiceUnavailable, status)
	require.Contains(t, body, "the device is busy")
	require.Less(t, time.Since(start), testTimeout, "it waited for the device instead of the queue timeout")
	<-done

	require.Equal(t, before, device.numCommands(), "the command must not reach the device later")
	require.False(t, device.channel(2))
}

// A request whose client went away while it waited must not switch the relay
// afterwards.
func TestARequestWhoseClientWentAwayIsNotExecuted(t *testing.T) {
	device := newFakeDevice(8, "v1.2.3")
	server := startQueueServer(t, device, 5*time.Second)
	done := occupyDevice(t, server, device)
	before := device.numCommands()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, relayURL(server, "2"), strings.NewReader("1"))
	require.NoError(t, err)
	_, err = http.DefaultClient.Do(req)
	require.Error(t, err)

	<-done
	// Give a wrongly queued request the chance to run
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, before, device.numCommands())
	require.False(t, device.channel(2))
}

func TestTheWriteTimeoutCoversQueueAndDevice(t *testing.T) {
	config := &ServerConfig{Timeout: 3 * time.Second, QueueTimeout: 4 * time.Second}
	require.Greater(t, config.queueTimeout()+config.timeout(), 6*time.Second)
	require.Equal(t, DefaultQueueTimeout, (&ServerConfig{}).queueTimeout())
	require.Equal(t, DefaultTimeout, (&ServerConfig{}).timeout())
}
