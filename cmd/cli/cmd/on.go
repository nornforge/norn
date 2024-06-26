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
	"strconv"

	"github.com/nornforge/norn/pkg/norn"
	"github.com/spf13/cobra"
)

// onCommand represents the activate command
var onCommand = &cobra.Command{
	Use:          "on",
	Short:        "Turn a channel on",
	Args:         cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		channel, err := strconv.Atoi(string(args[0]))
		if err != nil {
			return err
		}
		command := norn.Command{
			Type:    norn.Set,
			Channel: uint(channel),
			Status:  true,
		}
		_, err = sendCommandOverSerial(command)
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(onCommand)
}
