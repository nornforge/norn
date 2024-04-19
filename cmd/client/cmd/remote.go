package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// Remote represents a remote client for controlling a relay.
type Remote struct {
	url string
}

// NewRemote creates a new Remote client with the specified URL.
func NewRemote(url string) *Remote {
	return &Remote{
		url: url,
	}
}

// SendRequest sends a PUT request to the remote server with the specified payload.
// Returns an error if there was a problem sending the request or reading the response.
func (r *Remote) SendRequest(channel uint, payload []byte) error {
	var err error
	url := fmt.Sprintf("%s/api/v1/relay/%d/value", r.url, channel)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %s", err)
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", string(result))
	}
	return err
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
