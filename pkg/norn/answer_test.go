package norn_test

import (
	"bufio"
	"strings"
	"testing"

	"github.com/nornforge/norn/pkg/norn"
	"github.com/stretchr/testify/require"
)

// The responses the firmware sends, from v0.0.1 on, to each command type.
func TestAnswersMatchesTheResponsesOfTheFirmware(t *testing.T) {
	get3 := norn.Command{Type: norn.Get, Channel: 3}
	set3 := norn.Command{Type: norn.Set, Channel: 3, Status: true}
	brightness0 := norn.Command{Type: norn.Brightness, Channel: 0}
	version := norn.Command{Type: norn.Version}
	maxChannels := norn.Command{Type: norn.MaxChannels}

	channel3 := norn.Response{Success: true, Channel: 3, Status: true}
	channel4 := norn.Response{Success: true, Channel: 4}
	brightness := norn.Response{Success: true, Channel: 0}
	versionInfo := norn.Response{Success: true, Message: "v0.1.0"}
	channelCount := norn.Response{Success: true, MaxChannel: 8}
	failure := norn.Response{Message: "Invalid index provided: 9"}

	for _, tc := range []struct {
		command  norn.Command
		response norn.Response
		answers  bool
	}{
		{get3, channel3, true},
		{get3, channel4, false},
		{get3, versionInfo, false},
		{get3, channelCount, false},
		{set3, channel3, true},
		{set3, norn.Response{Success: true, Channel: 3, Status: false}, true}, // status is not compared
		{set3, channel4, false},
		{brightness0, brightness, true},
		{brightness0, versionInfo, false},
		{brightness0, channelCount, false},
		{brightness0, channel3, false},
		{version, versionInfo, true},
		{version, brightness, false},
		{version, channelCount, false},
		{version, channel3, false},
		{maxChannels, channelCount, true},
		{maxChannels, versionInfo, false},
		{maxChannels, channel3, false},
		{get3, failure, true},
		{version, failure, true},
	} {
		require.Equal(t, tc.answers, tc.response.Answers(tc.command), "%+v answers %+v", tc.response, tc.command)
	}
}

func TestParseAnswerSkipsResponsesToEarlierCommands(t *testing.T) {
	stale := []norn.Response{
		{Success: true, Channel: 2, Status: true},
		{Success: true, Message: "v0.1.0"},
		{Success: true, MaxChannel: 8},
	}
	answer := norn.Response{Success: true, Channel: 5}
	var input strings.Builder
	for _, r := range stale {
		input.Write(r.Marshal())
	}
	input.Write(answer.Marshal())

	received := norn.Response{}
	err := received.ParseAnswer(bufio.NewReader(strings.NewReader(input.String())), norn.Command{Type: norn.Get, Channel: 5})
	require.NoError(t, err)
	require.Equal(t, answer, received)
}

func TestParseAnswerReportsAParseError(t *testing.T) {
	stale := norn.Response{Success: true, Channel: 2}
	received := norn.Response{}
	err := received.ParseAnswer(bufio.NewReader(strings.NewReader(string(stale.Marshal()))), norn.Command{Type: norn.Get, Channel: 5})
	require.Error(t, err) // EOF after skipping the stale response
}
