// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

var targetContexts []string
var kubeconfig string
var verbose bool

var example = `  # Make cluster 'foo' unreachable from clusters 'bar' and 'baz':
  oc blackhole block foo --contexts bar,baz

  # Make cluster 'foo' reachable again from clusters 'bar' and 'baz':
  oc backhole unblock foo --contexts bar,baz
`
var rootCmd = &cobra.Command{
	Use:     "oc-blackhole",
	Short:   "Make a cluster unreachable from another cluster",
	Example: example,
	Annotations: map[string]string{
		cobra.CommandDisplayNameAnnotation: "oc blackhole",
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func defaultKubeconfig() string {
	env := os.Getenv("KUBECONFIG")
	if env != "" {
		return env
	}
	return clientcmd.RecommendedHomeFile
}

func init() {
	rootCmd.PersistentFlags().StringSliceVar(&targetContexts, "contexts", []string{},
		"the kubeconfig contexts of the target clusters")
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", defaultKubeconfig(),
		"the kubeconfig file to use")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"be more verbose")
}
