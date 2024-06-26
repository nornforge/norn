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
	"github.com/nornforge/norn/pkg/norn"
	"github.com/spf13/cobra"
	"go.bug.st/serial"
)

// versionCmd represents the version command
var bootCmd = &cobra.Command{
	Use:          "boot",
	Short:        "Enters the bootloader mode, this is useful for firmware updates",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		command := norn.Command{Type: norn.Bootloader}
		portName, err := rootCmd.PersistentFlags().GetString("device")
		if err != nil {
			return err
		}
		mode := &serial.Mode{
			BaudRate: 115200,
			Parity:   serial.NoParity,
			DataBits: 8,
			StopBits: serial.OneStopBit,
		}
		port, err := serial.Open(portName, mode)
		if err != nil {
			return err
		}
		defer port.Close()
		port.Write(command.Marshal())

		return nil
	},
}

func init() {
	rootCmd.AddCommand(bootCmd)
}
