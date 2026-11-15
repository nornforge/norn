package cmd

import (
	"fmt"
	"io"
	"time"
)

// defaultTimeout is how long the CLI waits for the response of the device.
const defaultTimeout = 2 * time.Second

// timeoutReader fails a Read that does not return before the deadline. The
// serial port has no deadline of its own, so without it a device that does
// not answer would block the CLI forever.
//
// When the deadline passes, the pending Read of the underlying reader is still
// blocked. If that reader is an io.Closer, it is closed to unblock the Read;
// the timeoutReader cannot be used afterwards.
type timeoutReader struct {
	reader   io.Reader
	timeout  time.Duration
	deadline time.Time
}

// newTimeoutReader returns a reader on which all reads together have to
// finish within the timeout, counted from now.
func newTimeoutReader(reader io.Reader, timeout time.Duration) *timeoutReader {
	return &timeoutReader{reader: reader, timeout: timeout, deadline: time.Now().Add(timeout)}
}

func (t *timeoutReader) Read(p []byte) (int, error) {
	type result struct {
		n   int
		err error
	}
	// The read may still be running after the timeout, so it gets a buffer
	// of its own instead of the one of the caller.
	buffer := make([]byte, len(p))
	done := make(chan result, 1)
	go func() {
		n, err := t.reader.Read(buffer)
		done <- result{n, err}
	}()

	timer := time.NewTimer(time.Until(t.deadline))
	defer timer.Stop()
	select {
	case r := <-done:
		copy(p, buffer[:r.n])
		return r.n, r.err
	case <-timer.C:
		if closer, ok := t.reader.(io.Closer); ok {
			closer.Close()
		}
		return 0, fmt.Errorf("no response from the device within %s: it may be busy, or its firmware may not know this command", t.timeout)
	}
}
