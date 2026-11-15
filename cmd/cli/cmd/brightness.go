/*
Copyright © 2026 Christian Ege <ch@ege.io>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"strconv"

	"github.com/nornforge/norn/pkg/norn"
	"github.com/spf13/cobra"
)

// parseBrightness converts the command line argument to a brightness value.
func parseBrightness(arg string) (uint, error) {
	value, err := strconv.ParseUint(arg, 10, 0)
	if err != nil || value > norn.MaxBrightness {
		return 0, fmt.Errorf("invalid brightness %q: expected a number in range [0,%d]", arg, norn.MaxBrightness)
	}
	return uint(value), nil
}

// brightnessCmd represents the brightness command
var brightnessCmd = &cobra.Command{
	Use:   "brightness <0-255>",
	Short: "Set the brightness of the status LED",
	Long: `Set the brightness of the status LED of the device.

The value ranges from 0, which turns the LED off, to 255 for full brightness.
The device does not store the setting: after a restart it uses its default again.`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		brightness, err := parseBrightness(args[0])
		if err != nil {
			return err
		}
		command := norn.Command{
			Type:    norn.Brightness,
			Channel: brightness,
		}
		_, err = sendCommandOverSerial(command)
		return err
	},
}

func init() {
	rootCmd.AddCommand(brightnessCmd)
}
