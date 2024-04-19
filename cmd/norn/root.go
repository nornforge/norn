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
	"os"

	"github.com/spf13/cobra"
)

// getEnv retrieves the value of the environment variable specified by the key.
// If the environment variable is not found, it returns the fallback value.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// serialDevice represents the serial port used by the NORN application.
var serialDevice = getEnv("NORN_SERIAL_PORT", "/dev/ttyACM0")

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "norn",
	Short: "A CLI to control the relays connected through a Microcontroller running the Norn firmware",
	Long: `A CLI to control the relays connected through a Microcontroller running the Norn firmware

The norn CLI can be used to test the connection to the device and can also act 


Note:
  In a set-up with multiple serial devices it might come handy to predefine the 
  environment variable NORN_SERIAL_PORT, so the serial port does not need to be 
  provided to every call of the cli.

Environment Variables:
  NORN_SERIAL_PORT: Define the default serial port to be used

`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringP("device", "d", serialDevice, "The serial device used for communication")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
}
