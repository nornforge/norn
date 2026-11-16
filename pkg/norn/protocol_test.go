package norn_test

import (
	"bufio"
	"fmt"
	"strings"
	"testing"

	"github.com/nornforge/norn/pkg/norn"
)

func TestGetCommand(t *testing.T) {
	var channelToTest uint = 1
	inputData := fmt.Sprintf("E979#000000037#e2da8fab{\"channel\":%d,\"type\":1,\"status\":false}ba9f13b7", channelToTest)
	command := norn.Command{}
	reader := bufio.NewReader(strings.NewReader(inputData))
	err := command.Parse(reader)
	require.Nil(t, err)
	require.Equal(t, norn.Get, command.Type)
	require.Equal(t, channelToTest, command.Channel)

	channelToTest = 8
	inputData = fmt.Sprintf("E979#000000037#e2da8fab{\"channel\":%d,\"type\":1,\"status\":false}ffa9c131", channelToTest)
	reader = bufio.NewReader(strings.NewReader(inputData))
	err = command.Parse(reader)
	require.Nil(t, err)
	require.Equal(t, norn.Get, command.Type)
	require.Equal(t, channelToTest, command.Channel)
}

func TestSetCommand(t *testing.T) {
	var channelToTest uint = 1
	inputData := fmt.Sprintf("E979#000000037#e2da8fab{\"channel\":%d,\"type\":2,\"status\":false}5ab27756", channelToTest)
	command := norn.Command{}
	reader := bufio.NewReader(strings.NewReader(inputData))
	err := command.Parse(reader)
	require.Nil(t, err)
	require.Equal(t, norn.Set, command.Type)
	require.Equal(t, channelToTest, command.Channel)
	require.Equal(t, false, command.Status)

	channelToTest = 8
	inputData = fmt.Sprintf("E979#000000036#10b10ca8{\"channel\":%d,\"type\":2,\"status\":true}ef4039dc", channelToTest)
	reader = bufio.NewReader(strings.NewReader(inputData))
	err = command.Parse(reader)
	require.Nil(t, err)
	require.Equal(t, norn.Set, command.Type)
	require.Equal(t, channelToTest, command.Channel)
	require.Equal(t, true, command.Status)
}

func TestInvalidCommand(t *testing.T) {
	inputData := "#x:<1>"
	command := norn.Command{}
	reader := bufio.NewReader(strings.NewReader(inputData))
	err := command.Parse(reader)
	require.NotNil(t, err)
}

func TestMultipleCommand(t *testing.T) {
	inputData := "E979#000000037#e2da8fab{\"channel\":1,\"type\":1,\"status\":false}ba9f13b7"
	inputData += "E979#000000037#e2da8fab{\"channel\":2,\"type\":1,\"status\":false}cc6c9fa6"
	command := norn.Command{}
	reader := bufio.NewReader(strings.NewReader(inputData))
	err := command.Parse(reader)
	require.Nil(t, err)
	var channelToTest uint = 1
	require.Equal(t, norn.Get, command.Type)
	require.Equal(t, channelToTest, command.Channel)
	channelToTest = 2
	err = command.Parse(reader)
	require.Nil(t, err)
	require.Equal(t, norn.Get, command.Type)
	require.Equal(t, channelToTest, command.Channel)
}

func TestVersionCommand(t *testing.T) {
	inputData := "E979#000000037#e2da8fab{\"channel\":1,\"type\":3,\"status\":false}0556AB09"
	command := norn.Command{}
	reader := bufio.NewReader(strings.NewReader(inputData))
	err := command.Parse(reader)
	require.Nil(t, err)
	require.Equal(t, norn.Version, command.Type)
}

// TestResponse is a unit test for the Response.Parse method.
func TestResponse(t *testing.T) {
	err := fmt.Errorf("Test")
	inputData := norn.MarshalError(err)
	reader := bufio.NewReader(strings.NewReader(inputData))
	response := norn.Response{}
	err = response.Parse(reader)
	require.Nil(t, err)
	require.Equal(t, response.Message, "Test")
	require.Equal(t, response.Success, false)
}

// The command types are part of the wire format shared with the firmware,
// so their values must never change.
func TestCommandTypeValues(t *testing.T) {
	require.Equal(t, norn.CommandType(0), norn.Nil)
	require.Equal(t, norn.CommandType(1), norn.Get)
	require.Equal(t, norn.CommandType(2), norn.Set)
	require.Equal(t, norn.CommandType(3), norn.Version)
	require.Equal(t, norn.CommandType(4), norn.Bootloader)
	require.Equal(t, norn.CommandType(5), norn.MaxChannels)
	require.Equal(t, norn.CommandType(6), norn.Brightness)
	require.Equal(t, 255, norn.MaxBrightness)
}

func TestBrightnessCommandRoundTrip(t *testing.T) {
	for _, brightness := range []uint{0, 16, norn.MaxBrightness} {
		sent := norn.Command{Type: norn.Brightness, Channel: brightness}
		data := sent.Marshal()
		require.Contains(t, string(data), fmt.Sprintf(`{"channel":%d,"type":6,"status":false}`, brightness))

		received := norn.Command{}
		err := received.Parse(bufio.NewReader(strings.NewReader(string(data))))
		require.NoError(t, err)
		require.Equal(t, sent, received)
	}
}

// The device echoes the brightness in the channel field of its response.
func TestBrightnessResponseRoundTrip(t *testing.T) {
	sent := norn.Response{Success: true, Channel: 16}

	received := norn.Response{}
	err := received.Parse(bufio.NewReader(strings.NewReader(string(sent.Marshal()))))
	require.NoError(t, err)
	require.Equal(t, sent, received)
}

func parseResponses(t *testing.T, input string, count int) []norn.Response {
	t.Helper()
	reader := bufio.NewReader(strings.NewReader(input))
	responses := make([]norn.Response, count)
	for i := range responses {
		require.NoError(t, responses[i].Parse(reader), "frame %d", i)
	}
	return responses
}

// Bytes before a frame, for example a partial frame or output of the
// firmware that is not a frame, must not keep the parser out of sync.
func TestParserSkipsBytesBeforeTheMarker(t *testing.T) {
	first := norn.Response{Success: true, Channel: 1}
	second := norn.Response{Success: true, Channel: 2, Status: true}
	for _, garbage := range []string{
		"x",
		"panic: runtime error\r\n",
		"E97",     // the start of a marker
		"EE979",   // a marker that starts twice
		"E979E97", // a marker cut off twice
	} {
		input := garbage + string(first.Marshal()) + garbage + string(second.Marshal())
		responses := parseResponses(t, input, 2)
		require.Equal(t, first, responses[0], "garbage %q", garbage)
		require.Equal(t, second, responses[1], "garbage %q", garbage)
	}
}

// After a frame with a broken checksum, the parser reports the error and
// the next frame is parsed again.
func TestParserRecoversAfterACorruptFrame(t *testing.T) {
	sent := norn.Response{Success: true, Channel: 3}
	frame := sent.Marshal()
	corrupt := []byte(string(frame))
	corrupt[len(corrupt)-12] ^= 0x01 // a byte of the payload

	reader := bufio.NewReader(strings.NewReader(string(corrupt) + string(frame)))
	response := norn.Response{}
	err := response.Parse(reader)
	require.ErrorContains(t, err, "crc checksums do not match")

	received := norn.Response{}
	require.NoError(t, received.Parse(reader))
	require.Equal(t, sent, received)
}

// A frame that was cut off is reported, but must not take the frame that
// follows it down too.
func TestParserRecoversAfterAFrameThatWasCutOff(t *testing.T) {
	sent := norn.Response{Success: true, Channel: 4, Status: true}
	frame := string(sent.Marshal())
	for _, length := range []int{6, 15, 20, 30, len(frame) - 1} {
		reader := bufio.NewReader(strings.NewReader(frame[:length] + frame))
		response := norn.Response{}
		require.Error(t, response.Parse(reader), "length %d", length)

		received := norn.Response{}
		require.NoError(t, received.Parse(reader), "length %d", length)
		require.Equal(t, sent, received, "length %d", length)
	}
}

// Frames larger than the buffer of the reader take another path.
func TestParserHandlesFramesLargerThanTheBuffer(t *testing.T) {
	sent := norn.Response{Message: strings.Repeat("m", 100)}
	data := string(sent.Marshal())

	// 16 is the smallest buffer bufio allows, smaller than the header
	for _, size := range []int{16, 32} {
		reader := bufio.NewReaderSize(strings.NewReader("xx"+data+data), size)
		for i := 0; i < 2; i++ {
			received := norn.Response{}
			require.NoError(t, received.Parse(reader), "size %d", size)
			require.Equal(t, sent, received, "size %d", size)
		}

		corrupt := strings.Replace(data, "mmm", "mmn", 1)
		received := norn.Response{}
		err := received.Parse(bufio.NewReaderSize(strings.NewReader(corrupt), size))
		require.ErrorContains(t, err, "crc checksums do not match", "size %d", size)
	}
}

func TestParserRejectsAnInvalidLengthField(t *testing.T) {
	for _, input := range []string{
		"E979#00000037#e2da8fab",   // one digit too few
		"E979#0000000037#e2da8fab", // one digit too many
		"E979#000000037#e2da8fac",  // checksum of the length does not match
		"E979#00000003x#00000000",  // not a number
	} {
		response := norn.Response{}
		err := response.Parse(bufio.NewReader(strings.NewReader(input)))
		require.Error(t, err, input)
	}
}

func TestParserReportsATruncatedFrame(t *testing.T) {
	frame := (&norn.Response{Success: true}).Marshal()
	for _, length := range []int{0, 3, 10, 20, len(frame) - 1} {
		response := norn.Response{}
		err := response.Parse(bufio.NewReader(strings.NewReader(string(frame[:length]))))
		require.Error(t, err, "length %d", length)
	}
}

// The wire format is shared with the firmware and must not change.
func TestWireFormat(t *testing.T) {
	command := norn.Command{Channel: 1, Type: norn.Get}
	require.Equal(t,
		"E979#000000037#e2da8fab{\"channel\":1,\"type\":1,\"status\":false}ba9f13b7",
		string(command.Marshal()))

	response := norn.Response{Success: true, Message: "v0.1.0", Channel: 2, MaxChannel: 8, Status: true}
	data := string(response.Marshal())
	require.True(t, strings.HasPrefix(data, "E979#"), data)
	require.Contains(t, data, `{"isOk":true,"message":"v0.1.0","channel":2,"max_channel":8,"status":true}`)
}
