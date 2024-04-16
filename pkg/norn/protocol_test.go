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
