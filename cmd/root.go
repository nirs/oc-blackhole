// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

var targetContext string
var kubeconfig string
var verbose bool

var example = `  # Make cluster 'foo' unreachable from cluster 'bar':
  oc blackhole block foo --context bar

  # Make cluster 'foo' reachable again from cluster 'bar'
  oc backhole unblock foo --context bar
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
	rootCmd.PersistentFlags().StringVar(&targetContext, "context", "", "The name of the kubeconfig context to use")
	rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "", defaultKubeconfig(), "kubeconfig file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "be more verbose")
}
