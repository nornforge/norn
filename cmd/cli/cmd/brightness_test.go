package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBrightness(t *testing.T) {
	for arg, want := range map[string]uint{"0": 0, "1": 1, "16": 16, "255": 255, "007": 7} {
		got, err := parseBrightness(arg)
		require.NoError(t, err, "argument %q", arg)
		require.Equal(t, want, got, "argument %q", arg)
	}
}

func TestParseBrightnessRejectsInvalidValues(t *testing.T) {
	for _, arg := range []string{"256", "1000", "-1", "", "abc", "1.5", "0x10", " 5", "99999999999999999999999"} {
		_, err := parseBrightness(arg)
		require.Error(t, err, "argument %q", arg)
		require.Contains(t, err.Error(), "[0,255]", "argument %q", arg)
	}
}

func TestBrightnessCommandTakesExactlyOneArgument(t *testing.T) {
	require.Error(t, brightnessCmd.Args(brightnessCmd, []string{}))
	require.NoError(t, brightnessCmd.Args(brightnessCmd, []string{"16"}))
	require.Error(t, brightnessCmd.Args(brightnessCmd, []string{"16", "32"}))
}
