// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strings"

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

		// TODO: Run in parallel
		for _, nodeName := range nodesNames(targetClient) {
			unblockAddresses(nodeName, addressesToBlock)
		}
	},
}

func unblockAddresses(nodeName string, addresses []string) {
	dbglog.Printf("unblocking addresses in node %s", nodeName)

	// `ip route del`` is not idempotent, so we build a command with existing
	// blackholed addresses.

	blackholes, err := findBlackholes(nodeName)
	if err != nil {
		errlog.Fatalf("failed to find blackholes on node %s: %s", nodeName, err)
	}

	var sb strings.Builder
	for _, address := range addresses {
		if blackholes.Has(address) {
			sb.WriteString("ip route del blackhole " + address + "\n")
		}
	}

	if sb.Len() == 0 {
		dbglog.Printf("No address to unblock on node %s", nodeName)
		return
	}

	_, err = execScript(nodeName, sb.String())
	if err != nil {
		errlog.Fatalf("failed to unblock addresses on node %s: %s", nodeName, err.Error())
	}
}

func init() {
	rootCmd.AddCommand(unblockCmd)
}
