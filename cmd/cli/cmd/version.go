/*
Copyright © 2023 Christian Ege <ch@ege.io>

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

	"github.com/nornforge/norn/pkg/norn"
	"github.com/nornforge/norn/version"
	"github.com/spf13/cobra"
)

// getDeviceVersion asks the device for the version of its firmware.
func getDeviceVersion() (string, error) {
	response, err := sendCommandOverSerial(norn.Command{Type: norn.Version})
	if err != nil {
		return "", fmt.Errorf("unable to get the version info from the device: %w", err)
	}
	return response.Message, nil
}

// ...

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:          "version",
	Short:        "Prints the program version",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		version := version.GetProgramVersion()
		fmt.Printf("Program Version : v%s\n", version.String())
		deviceVersion, err := getDeviceVersion()
		if err != nil {
			return err
		}
		fmt.Printf("Device Version  : %s\n", deviceVersion)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
