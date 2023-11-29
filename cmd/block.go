// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strings"

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

		// TODO: Run in parallel
		for _, nodeName := range nodesNames(targetClient) {
			blockAddresses(nodeName, addressesToBlock)
		}
	},
}

func blockAddresses(nodeName string, addresses []string) {
	dbglog.Printf("blocking addresses in node %s", nodeName)

	var sb strings.Builder
	for _, address := range addresses {
		// `replace` is idempotent, no need to check for existing blackholes.
		sb.WriteString("ip route replace blackhole " + address + "\n")
	}

	_, err := execScript(nodeName, sb.String())
	if err != nil {
		errlog.Fatalf("failed to block addresses on node %s: %s", nodeName, err.Error())
	}
}

func init() {
	rootCmd.AddCommand(blockCmd)
}
