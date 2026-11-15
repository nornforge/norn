package cmd

import (
	"bufio"
	"io"
	"testing"
	"time"

	"github.com/nornforge/norn/pkg/norn"
	"github.com/stretchr/testify/require"
)

func TestTimeoutReaderPassesDataThrough(t *testing.T) {
	sent := norn.Response{Success: true, Channel: 3, Status: true}
	pipeReader, pipeWriter := io.Pipe()
	go func() {
		pipeWriter.Write(sent.Marshal())
	}()

	received := norn.Response{}
	err := received.Parse(bufio.NewReader(newTimeoutReader(pipeReader, 5*time.Second)))
	require.NoError(t, err)
	require.Equal(t, sent, received)
}

func TestTimeoutReaderGivesUpOnASilentDevice(t *testing.T) {
	pipeReader, pipeWriter := io.Pipe() // nothing is ever written
	defer pipeWriter.Close()

	start := time.Now()
	response := norn.Response{}
	err := response.Parse(bufio.NewReader(newTimeoutReader(pipeReader, 100*time.Millisecond)))

	require.Error(t, err)
	require.Contains(t, err.Error(), "no response from the device within 100ms")
	require.Less(t, time.Since(start), 2*time.Second)
}

func TestTimeoutReaderGivesUpOnAResponseThatStopsHalfway(t *testing.T) {
	frame := (&norn.Response{Success: true}).Marshal()
	pipeReader, pipeWriter := io.Pipe()
	defer pipeWriter.Close()
	go func() {
		pipeWriter.Write(frame[:len(frame)/2])
	}()

	start := time.Now()
	response := norn.Response{}
	err := response.Parse(bufio.NewReader(newTimeoutReader(pipeReader, 100*time.Millisecond)))

	require.Error(t, err)
	require.Contains(t, err.Error(), "no response from the device")
	require.Less(t, time.Since(start), 2*time.Second)
}

// The timeout covers the whole response, not each read on its own: a device
// that keeps sending single bytes must not keep the CLI waiting forever.
func TestTimeoutReaderLimitsTheTotalTime(t *testing.T) {
	pipeReader, pipeWriter := io.Pipe()
	defer pipeWriter.Close()
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-time.After(20 * time.Millisecond):
				pipeWriter.Write([]byte("x"))
			}
		}
	}()

	reader := newTimeoutReader(pipeReader, 150*time.Millisecond)
	start := time.Now()
	_, err := io.ReadAll(reader)

	require.Error(t, err)
	require.Less(t, time.Since(start), 2*time.Second)
}

// blockingReader blocks in Read until it is closed, like a serial port
// without data.
type blockingReader struct {
	closed   chan struct{}
	returned chan struct{}
}

func (b *blockingReader) Read(p []byte) (int, error) {
	defer close(b.returned)
	<-b.closed
	return 0, io.ErrClosedPipe
}

func (b *blockingReader) Close() error {
	close(b.closed)
	return nil
}

func TestTimeoutReaderClosesTheReaderToUnblockThePendingRead(t *testing.T) {
	blocking := &blockingReader{closed: make(chan struct{}), returned: make(chan struct{})}

	_, err := newTimeoutReader(blocking, 50*time.Millisecond).Read(make([]byte, 8))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no response from the device")

	select {
	case <-blocking.returned:
	case <-time.After(2 * time.Second):
		t.Fatal("the pending Read was left blocked")
	}
}

func TestTimeoutReaderDoesNotCloseAReaderThatAnswers(t *testing.T) {
	pipeReader, pipeWriter := io.Pipe()
	go func() {
		pipeWriter.Write([]byte("abc"))
		pipeWriter.Write([]byte("def"))
	}()

	reader := newTimeoutReader(pipeReader, 5*time.Second)
	buffer := make([]byte, 3)
	_, err := io.ReadFull(reader, buffer)
	require.NoError(t, err)
	// A closed pipe would fail the second read
	_, err = io.ReadFull(reader, buffer)
	require.NoError(t, err)
	require.Equal(t, "def", string(buffer))
}
