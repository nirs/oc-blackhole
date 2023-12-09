// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

var unblockCmd = &cobra.Command{
	Use:   "unblock cluster [flags]",
	Short: "Make cluster reachable again from target cluster",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		blockedContext := args[0]

		c, err := NewCommand(blockedContext, targetContexts, kubeconfig)
		if err != nil {
			errlog.Fatal(err)
		}

		err = c.InspectClusters()
		if err != nil {
			errlog.Fatal(err)
		}

		err = c.UnblockCluster()
		if err != nil {
			errlog.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(unblockCmd)
}
