// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

var blockCmd = &cobra.Command{
	Use:   "block cluster [flags]",
	Short: "Make a cluster unreachable from target cluster",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		blockedContext := args[0]

		c, err := NewCommand(blockedContext, targetContexts, kubeconfig, showProgress)
		if err != nil {
			errlog.Fatal(err)
		}

		err = c.BlockCluster()
		if err != nil {
			errlog.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(blockCmd)
}
