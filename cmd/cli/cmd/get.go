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
	"strconv"

	"github.com/nornforge/norn/pkg/norn"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:          "get",
	Short:        "Get the status of an given channel",
	Args:         cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		channel, err := strconv.Atoi(string(args[0]))
		if err != nil {
			return err
		}
		command := norn.Command{
			Type:    norn.Get,
			Channel: uint(channel),
		}
		response, err := sendCommandOverSerial(command)
		if err != nil {
			return err
		}
		if !response.Success {
			return fmt.Errorf(response.Message)
		}
		status := 0
		if response.Status {
			status = 1
		}
		fmt.Printf("%d\n", status)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
