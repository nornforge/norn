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
	"strconv"

	"github.com/spf13/cobra"
)

// onCommand represents the activate command
var onCommand = &cobra.Command{
	Use:          "on",
	Short:        "Turn a channel on",
	Args:         cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {

		remoteURL, err := rootCmd.PersistentFlags().GetString("url")
		if err != nil {
			return err
		}

		channel, err := strconv.ParseUint(string(args[0]), 10, 32)
		if err != nil {
			return err
		}

		remote := NewRemote(remoteURL)

		return remote.On(uint(channel))
	},
}

func init() {
	rootCmd.AddCommand(onCommand)
}
