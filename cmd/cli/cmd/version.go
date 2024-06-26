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
	"bufio"
	"fmt"

	"github.com/nornforge/norn/pkg/norn"
	"github.com/nornforge/norn/version"
	"github.com/spf13/cobra"
	"go.bug.st/serial"
)

func getDeviceVersion(portName string) (string, error) {
	command := norn.Command{Type: norn.Version}
	mode := &serial.Mode{
		BaudRate: 115200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	port, err := serial.Open(portName, mode)
	if err != nil {
		return "", fmt.Errorf(
			"unable to open the serial port: %s",
			portName,
		)
	}
	defer port.Close()
	port.Write(command.Marshal())
	reader := bufio.NewReader(port)
	response := norn.Response{}
	err = response.Parse(reader)
	if err != nil {
		return "", err
	}

	if !response.Success {
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

		portName, err := rootCmd.PersistentFlags().GetString("device")
		if err != nil {
			return err
		}

		version := version.GetProgramVersion()
		fmt.Printf("Program Version : v%s\n", version.String())
		deviceVersion, err := getDeviceVersion(portName)
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
