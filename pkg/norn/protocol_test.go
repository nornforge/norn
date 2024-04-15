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
	inputData := fmt.Sprintf("000000037:{\"channel\":%d,\"type\":1,\"status\":false}", channelToTest)
	command := norn.Command{}
	reader := bufio.NewReader(strings.NewReader(inputData))
	err := command.Parse(reader)
	require.Nil(t, err)
	require.Equal(t, norn.Get, command.Type)
	require.Equal(t, channelToTest, command.Channel)

	channelToTest = 8
	inputData = fmt.Sprintf("000000037:{\"channel\":%d,\"type\":1,\"status\":false}", channelToTest)
	reader = bufio.NewReader(strings.NewReader(inputData))
	err = command.Parse(reader)
	require.Nil(t, err)
	require.Equal(t, norn.Get, command.Type)
	require.Equal(t, channelToTest, command.Channel)
}

func TestSetCommand(t *testing.T) {
	var channelToTest uint = 1
	inputData := fmt.Sprintf("000000037:{\"channel\":%d,\"type\":2,\"status\":false}", channelToTest)
	command := norn.Command{}
	reader := bufio.NewReader(strings.NewReader(inputData))
	err := command.Parse(reader)
	require.Nil(t, err)
	require.Equal(t, norn.Set, command.Type)
	require.Equal(t, channelToTest, command.Channel)
	require.Equal(t, false, command.Status)

	channelToTest = 8
	inputData = fmt.Sprintf("000000036:{\"channel\":%d,\"type\":2,\"status\":true}", channelToTest)
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
	inputData := "000000037:{\"channel\":1,\"type\":1,\"status\":false}#BA9F13B7"
	inputData += "000000037:{\"channel\":2,\"type\":1,\"status\":false}#CC6C9FA6"
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
	inputData := "000000037:{\"channel\":1,\"type\":3,\"status\":false}#0556AB09\r\n"
	command := norn.Command{}
	reader := bufio.NewReader(strings.NewReader(inputData))
	err := command.Parse(reader)
	require.Nil(t, err)
	require.Equal(t, norn.Version, command.Type)
}
