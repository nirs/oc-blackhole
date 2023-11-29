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
		validateContexts(blockedContext)

		config := loadConfig()
		blockedClient := clientForContext(config, blockedContext)
		targetClient := clientForContext(config, targetContext)

		addressesToBlock := nodesAddresses(blockedClient)
		dbglog.Printf("blocked nodes addresses: %v", addressesToBlock)

		for _, nodeName := range nodesNames(targetClient) {
			unblockAddresses(nodeName, addressesToBlock)
		}
	},
}

func unblockAddresses(nodeName string, addreses []string) {
	dbglog.Printf("unblocking addresses in node %s", nodeName)
}

func init() {
	rootCmd.AddCommand(unblockCmd)
}
