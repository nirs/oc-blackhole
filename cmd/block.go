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
		validateContexts(blockedContext)

		config := loadConfig()
		blockedClient := clientForContext(config, blockedContext)
		targetClient := clientForContext(config, targetContext)

		addressesToBlock := nodesAddresses(blockedClient)
		dbglog.Printf("blocked nodes addresses: %v", addressesToBlock)

		for _, nodeName := range nodesNames(targetClient) {
			blockAddresses(nodeName, addressesToBlock)
		}
	},
}

func blockAddresses(nodeName string, addreses []string) {
	dbglog.Printf("blocking addresses in node %s", nodeName)
}

func init() {
	rootCmd.AddCommand(blockCmd)
}
