// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"net"
	"net/url"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type BlockedCluster struct {
	Context            string
	NodeAddresses      []string
	APIServerAddresses []string
	config             *api.Config
	client             *kubernetes.Clientset
}

type TargetCluster struct {
	Context   string
	NodeNames []string
	config    *api.Config
	client    *kubernetes.Clientset
}

func NewBlockedCluster(config *api.Config, context string) (*BlockedCluster, error) {
	client, err := clientForContext(config, context)
	if err != nil {
		return nil, err
	}
	cluster := &BlockedCluster{Context: context, config: config, client: client}
	return cluster, nil
}

func (c *BlockedCluster) Inspect() error {
	var err error

	c.NodeAddresses, err = c.findNodesAddresses()
	if err != nil {
		return err
	}

	c.APIServerAddresses, err = c.findAPIServerAddress()
	if err != nil {
		return err
	}

	return nil
}

// AllAddresses return sorted list of uniqe cluster address that must be blocked
// on the target cluster.
func (c *BlockedCluster) AllAddresses() []string {
	res := sets.New(c.NodeAddresses...)
	res.Insert(c.APIServerAddresses...)
	return sets.List(res)
}

func (c *BlockedCluster) findNodesAddresses() ([]string, error) {
	var res []string

	nodes, err := c.client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, node := range nodes.Items {
		address, err := externalIP(&node)
		if err != nil {
			return nil, err
		}

		dbglog.Printf("found blocked cluster node %s address %s", node.Name, address)
		res = append(res, address)
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("could not find any blocked cluster node addresses")
	}

	return res, nil
}

func externalIP(node *apiv1.Node) (string, error) {
	for _, addr := range node.Status.Addresses {
		if addr.Type == "ExternalIP" {
			return addr.Address, nil
		}
	}
	return "", fmt.Errorf("couuld not find external IP address for node %s", node.Name)
}

func (c *BlockedCluster) findAPIServerAddress() ([]string, error) {
	cluster, ok := c.config.Clusters[c.Context]
	if !ok {
		return nil, fmt.Errorf("no cluster for context %q", c.Context)
	}

	server, err := url.Parse(cluster.Server)
	if err != nil {
		return nil, fmt.Errorf("cannnot parse cluster %q server URL %q", c.Context, cluster.Server)
	}

	ips, err := net.LookupIP(server.Host)
	if err != nil {
		return nil, err
	}

	var res []string
	for _, ip := range ips {
		res = append(res, ip.String())
	}

	return res, nil
}

func NewTargetCluster(config *api.Config, context string) (*TargetCluster, error) {
	client, err := clientForContext(config, context)
	if err != nil {
		return nil, err
	}
	cluster := &TargetCluster{Context: context, config: config, client: client}
	return cluster, nil
}

func (c *TargetCluster) Inspect() error {
	var err error

	c.NodeNames, err = c.findNodeNames()
	if err != nil {
		return err
	}

	return nil
}

func (c *TargetCluster) findNodeNames() ([]string, error) {
	var res []string

	nodes, err := c.client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, node := range nodes.Items {
		res = append(res, node.Name)
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("could not find any target cluster node")
	}

	return res, nil
}

func clientForContext(kubeconfig *api.Config, context string) (*kubernetes.Clientset, error) {
	config := clientcmd.NewNonInteractiveClientConfig(*kubeconfig, context, nil, nil)
	clientConfig, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}
	return client, nil
}
