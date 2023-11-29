// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func validateContexts(blockedContext string) {
	if blockedContext == targetContext {
		errlog.Fatalf("blocked cluster must be different from target cluster")
	}
}

func loadConfig() *api.Config {
	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		errlog.Fatal(err)
	}
	return config
}

func clientForContext(kubeconfig *api.Config, context string) *kubernetes.Clientset {
	config := clientcmd.NewNonInteractiveClientConfig(*kubeconfig, context, nil, nil)
	clientConfig, err := config.ClientConfig()
	if err != nil {
		errlog.Fatal(err)
	}
	client, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		errlog.Fatal(err)
	}
	return client
}

func nodesAddresses(client *kubernetes.Clientset) []string {
	var res []string

	nodes, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		errlog.Fatal(err)
	}

	for _, node := range nodes.Items {
		address := externalIP(&node)
		dbglog.Printf("found blocked cluster node %s address %s", node.Name, address)
		res = append(res, address)
	}

	if len(res) == 0 {
		errlog.Fatal("could not find any blocked cluster node addresses")
	}

	return res
}

func nodesNames(client *kubernetes.Clientset) []string {
	var res []string

	nodes, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		errlog.Fatal(err)
	}

	for _, node := range nodes.Items {
		res = append(res, node.Name)
	}

	if len(res) == 0 {
		errlog.Fatal("could not find any target cluster node")
	}

	return res
}

func externalIP(node *apiv1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == "ExternalIP" {
			return addr.Address
		}
	}
	errlog.Fatalf("couuld not find external IP address for node %s", node.Name)
	return ""
}

func findBlackholes(nodeName string) (sets.Set[string], error) {
	dbglog.Printf("Looking up blackholes on node %s", nodeName)

	res := sets.New[string]()

	cmd := exec.Command(
		"oc",
		"debug",
		"node/"+nodeName,
		"--context",
		targetContext,
		"--",
		"ip",
		"route",
	)

	dbglog.Printf("Running command on node %s: %s", nodeName, cmd.Args)

	out, err := cmd.Output()
	if err != nil {
		return res, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))

	for scanner.Scan() {
		route := strings.SplitN(scanner.Text(), " ", 2)
		if len(route) == 2 && route[0] == "blackhole" {
			res.Insert(strings.Trim(route[1], " "))
		}
	}
	if err = scanner.Err(); err != nil {
		return res, err
	}

	return res, nil
}

func execScript(nodeName string, script string) ([]byte, error) {
	cmd := exec.Command(
		"oc",
		"debug",
		"node/"+nodeName,
		"--context",
		targetContext,
		"--",
		"sh",
		"-c",
		script,
	)

	dbglog.Printf("Running command on node %s: %s", nodeName, cmd.Args)

	return cmd.Output()
}
