package norn_test

import (
	"bufio"
	"fmt"
	"strings"
	"testing"

	"github.com/nornforge/norn/pkg/norn"
	"github.com/stretchr/testify/require"
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
