// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show cluster",
	Short: "Show if cluster is in blackhole",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		blockedContext := args[0]

		c, err := NewCommand(blockedContext, targetContext, kubeconfig)
		if err != nil {
			errlog.Fatal(err)
		}

		err = c.InspectClusters()
		if err != nil {
			errlog.Fatal(err)
		}

		status, err := c.ClusterStatus()
		if err != nil {
			errlog.Fatal(err)
		}

		fmt.Printf("status:\n")
		fmt.Printf("  cluster: %q\n", blockedContext)
		fmt.Printf("  target: %q\n", targetContext)
		fmt.Printf("  valid: %v\n", status.Valid)
		fmt.Printf("  nodes:\n")
		for _, name := range sortedKeys(status.Nodes) {
			fmt.Printf("    - name: %q\n", name)
			fmt.Printf("      status: %q\n", status.Nodes[name])
		}
	},
}

func sortedKeys(m map[string]BlackholeStatus) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func init() {
	rootCmd.AddCommand(showCmd)
}
