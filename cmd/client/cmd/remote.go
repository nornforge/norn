package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// defaultTimeout is how long the client waits for the server by default. The
// server itself waits at most a few seconds for the device.
const defaultTimeout = 10 * time.Second

// Remote represents a remote client for controlling a relay.
type Remote struct {
	url    string
	client *http.Client
}

// NewRemote creates a new Remote client for the server at the given URL. A
// request that takes longer than the timeout fails.
func NewRemote(url string, timeout time.Duration) *Remote {
	return &Remote{
		// "http://host:8080/" would lead to "//api/...", which the server
		// answers with a redirect
		url: strings.TrimRight(url, "/"),
		client: &http.Client{
			Timeout: timeout,
			// Go follows a redirect of a PUT with a GET without body. The GET
			// succeeds, so the relay would silently stay as it is.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// newRemoteFromFlags creates a Remote from the --url and --timeout flags.
func newRemoteFromFlags() (*Remote, error) {
	remoteURL, err := rootCmd.PersistentFlags().GetString("url")
	if err != nil {
		return nil, err
	}
	timeout, err := rootCmd.PersistentFlags().GetDuration("timeout")
	if err != nil {
		return nil, err
	}
	return NewRemote(remoteURL, timeout), nil
}

// SendRequest sends a PUT request to the remote server with the specified payload.
// Returns an error if there was a problem sending the request or reading the response.
func (r *Remote) SendRequest(channel uint, payload []byte) error {
	url := fmt.Sprintf("%s/api/v1/relay/%d/value", r.url, channel)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		message := strings.TrimSpace(string(result))
		if message == "" {
			message = resp.Status
		}
		return fmt.Errorf("%s", message)
	}
	return nil
}

// On turns on the relay channel specified by the given channel number.
// It sends a PUT request to the remote server to update the relay value.
// Returns an error if there was a problem sending the request or reading the response.
func (r *Remote) On(channel uint) error {
	payload := []byte("1")
	return r.SendRequest(channel, payload)
}

// Off turns off the relay channel specified by the given channel number.
// It sends a PUT request to the remote server to update the relay value.
// Returns an error if there was a problem sending the request or reading the response.
func (r *Remote) Off(channel uint) error {
	payload := []byte("0")
	return r.SendRequest(channel, payload)
}
