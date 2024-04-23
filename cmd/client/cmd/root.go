/*
Copyright © 2024 Christian Ege <ch@ege.io>

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
var nornServerURL = getEnv("NORN_URL", "http://localhost:8080")

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "norn-client",
	Short: "A CLI to control the relays connected to a Norn Server",
	Long: `A CLI to control the relays attached to a Norn server
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

	rootCmd.PersistentFlags().String("url", nornServerURL, "The URL of the norn server")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
}
