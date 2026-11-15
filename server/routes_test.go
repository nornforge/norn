package server

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

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
