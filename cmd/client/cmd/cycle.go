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
	"time"

	"github.com/spf13/cobra"
)

const defaultDelay uint8 = 3

// cycleCmd represents the cycle command
var cycleCmd = &cobra.Command{
	Use:          "cycle",
	Short:        "Cycle a channel",
	Long:         `This will turn off the channel wait the defined 'delay' time and turn it on again`,
	SilenceUsage: true,
	Args:         cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		remote, err := newRemoteFromFlags()
		if err != nil {
			return err
		}

		channel, err := strconv.ParseUint(string(args[0]), 10, 32)
		if err != nil {
			return err
		}

		delay, err := cmd.Flags().GetUint8("delay")
		if err != nil {
			return err
		}

		return cycle(remote, uint(channel), time.Duration(delay)*time.Second)
	},
}

// cycle turns the channel off, waits for the delay and turns it on again.
func cycle(remote *Remote, channel uint, delay time.Duration) error {
	if err := remote.Off(channel); err != nil {
		return fmt.Errorf("unable to turn off the channel %d on %s: %w", channel, remote.url, err)
	}
	time.Sleep(delay)
	if err := remote.On(channel); err != nil {
		return fmt.Errorf("unable to turn on the channel %d on %s: %w", channel, remote.url, err)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(cycleCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// cycleCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	cycleCmd.Flags().Uint8("delay", defaultDelay, "The time in seconds between the off and on state")
}
