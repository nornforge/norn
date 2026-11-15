package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// fakeServer answers the relay routes the way the norn server routes them,
// with an http.ServeMux, and records the requests that reach them.
type fakeServer struct {
	mutex    sync.Mutex
	requests []string
	status   int
	reply    string
	delay    time.Duration
}

func (f *fakeServer) start(t *testing.T) *httptest.Server {
	t.Helper()
	handler := func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mutex.Lock()
		f.requests = append(f.requests, r.Method+" "+r.PathValue("index")+" "+string(body))
		status, reply, delay := f.status, f.reply, f.delay
		f.mutex.Unlock()
		time.Sleep(delay)
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}
	router := http.NewServeMux()
	router.HandleFunc("GET /api/v1/relay/{index}/value", handler)
	router.HandleFunc("PUT /api/v1/relay/{index}/value", handler)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server
}

func (f *fakeServer) recorded() []string {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return append([]string(nil), f.requests...)
}

func TestOnAndOffSendAPut(t *testing.T) {
	fake := &fakeServer{}
	remote := NewRemote(fake.start(t).URL, time.Second)

	require.NoError(t, remote.On(3))
	require.NoError(t, remote.Off(8))
	require.Equal(t, []string{"PUT 3 1", "PUT 8 0"}, fake.recorded())
}

// With a trailing slash, the request went to "//api/...". The router
// redirected it, and Go followed with a GET: the relay stayed as it was,
// and the client reported success.
func TestATrailingSlashInTheURLStillSwitchesTheRelay(t *testing.T) {
	fake := &fakeServer{}
	remote := NewRemote(fake.start(t).URL+"/", time.Second)

	require.NoError(t, remote.On(1))
	require.Equal(t, []string{"PUT 1 1"}, fake.recorded())
}

func TestARedirectIsReportedInsteadOfFollowed(t *testing.T) {
	fake := &fakeServer{}
	target := fake.start(t)
	redirect := httptest.NewServer(http.RedirectHandler(target.URL+"/api/v1/relay/1/value", http.StatusMovedPermanently))
	t.Cleanup(redirect.Close)

	err := NewRemote(redirect.URL, time.Second).On(1)
	require.ErrorContains(t, err, "301")
	require.Empty(t, fake.recorded())
}

func TestTheMessageOfTheServerIsReported(t *testing.T) {
	fake := &fakeServer{status: http.StatusBadRequest, reply: "invalid index provided: 9 [1,8]"}
	err := NewRemote(fake.start(t).URL, time.Second).On(9)
	require.EqualError(t, err, "invalid index provided: 9 [1,8]")
}

func TestAnErrorWithoutMessageReportsTheStatus(t *testing.T) {
	fake := &fakeServer{status: http.StatusInternalServerError}
	err := NewRemote(fake.start(t).URL, time.Second).On(1)
	require.EqualError(t, err, "500 Internal Server Error")
}

func TestAServerThatDoesNotAnswerTimesOut(t *testing.T) {
	fake := &fakeServer{delay: time.Second}
	remote := NewRemote(fake.start(t).URL, 100*time.Millisecond)

	start := time.Now()
	err := remote.On(1)
	require.ErrorContains(t, err, "error sending request")
	require.Less(t, time.Since(start), 900*time.Millisecond)
}

func TestAServerThatIsNotRunningIsReported(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()

	err := NewRemote(url, time.Second).On(1)
	require.ErrorContains(t, err, "error sending request")
}

func TestCycleTurnsTheChannelOffAndOnAgain(t *testing.T) {
	fake := &fakeServer{}
	remote := NewRemote(fake.start(t).URL, time.Second)

	require.NoError(t, cycle(remote, 4, 10*time.Millisecond))
	require.Equal(t, []string{"PUT 4 0", "PUT 4 1"}, fake.recorded())
}

// The reason a cycle failed must reach the user, not only the URL.
func TestCycleReportsTheErrorOfTheServer(t *testing.T) {
	fake := &fakeServer{status: http.StatusBadRequest, reply: "relay stuck"}
	remote := NewRemote(fake.start(t).URL, time.Second)

	err := cycle(remote, 4, 0)
	require.ErrorContains(t, err, "unable to turn off the channel 4")
	require.ErrorContains(t, err, "relay stuck")
	require.Equal(t, []string{"PUT 4 0"}, fake.recorded(), "no attempt to turn it on")
}

func TestCommandsTakeExactlyOneArgument(t *testing.T) {
	for _, command := range []*cobra.Command{onCommand, offCommand, cycleCmd} {
		require.Error(t, command.Args(command, []string{}), command.Use)
		require.NoError(t, command.Args(command, []string{"1"}), command.Use)
		require.Error(t, command.Args(command, []string{"1", "2"}), command.Use)
	}
}
