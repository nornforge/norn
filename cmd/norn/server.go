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

	"github.com/nornforge/norn/server"
	"github.com/spf13/cobra"
	"go.bug.st/serial"
)

func serverCommand(cmd *cobra.Command, args []string) error {
	conf := server.ServerConfig{
		Host: "localhost",
		Port: 8080,
	}

	var err error = nil

	conf.Host, err = cmd.Flags().GetString("host")
	if err != nil {
		return err
	}

	conf.Port, err = cmd.Flags().GetUint16("port")
	if err != nil {
		return err
	}

	serialPortName, err := rootCmd.PersistentFlags().GetString("device")
	if err != nil {
		return err
	}
	mode := &serial.Mode{
		BaudRate: 115200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	serialPort, err := serial.Open(serialPortName, mode)
	if err != nil {
		fmt.Printf("Error: while opening serial device: %s", serialPortName)
		return err
	}
	conf.SerialDevice = serialPort
	defer serialPort.Close()
	return server.Serve(&conf)
}

// serverCmd represents a REST server to serve the relays
var serverCmd = &cobra.Command{
	Use:          "server",
	Short:        "Serves the Relays over a REST API",
	SilenceUsage: true,
	RunE:         serverCommand,
}

func init() {
	serverCmd.Flags().Uint16("port", 8080, "The port the relays service is listening on")
	serverCmd.Flags().String("bind", "127.0.0.1", "The address the relays service is binding on")
	rootCmd.AddCommand(serverCmd)
}
