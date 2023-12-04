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
	k8sClient          *kubernetes.Clientset
}

type TargetCluster struct {
	Context   string
	NodeNames []string
	config    *api.Config
	k8sClient *kubernetes.Clientset
}

func NewBlockedCluster(config *api.Config, context string) (*BlockedCluster, error) {
	k8sClient, err := createK8sClient(config, context)
	if err != nil {
		return nil, err
	}

	cluster := &BlockedCluster{Context: context, config: config, k8sClient: k8sClient}
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
	nodes, err := c.k8sClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var res []string

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
	context, ok := c.config.Contexts[c.Context]
	if !ok {
		return nil, fmt.Errorf("could not find context %q", c.Context)
	}

	cluster, ok := c.config.Clusters[context.Cluster]
	if !ok {
		return nil, fmt.Errorf("could not find cluster %q", context.Cluster)
	}

	server, err := url.Parse(cluster.Server)
	if err != nil {
		return nil, fmt.Errorf("cannnot parse cluster %q server URL %q",
			c.Context, cluster.Server)
	}

	ips, err := net.LookupIP(server.Hostname())
	if err != nil {
		return nil, err
	}

	var res []string
	for _, ip := range ips {
		dbglog.Printf("found blocked cluster api server %s address %s",
			server.Hostname(), ip)
		res = append(res, ip.String())
	}

	return res, nil
}

func NewTargetCluster(config *api.Config, context string) (*TargetCluster, error) {
	k8sClient, err := createK8sClient(config, context)
	if err != nil {
		return nil, err
	}

	cluster := &TargetCluster{Context: context, config: config, k8sClient: k8sClient}
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

	nodes, err := c.k8sClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
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

func createK8sClient(config *api.Config, context string) (*kubernetes.Clientset, error) {
	rc, err := clientcmd.NewNonInteractiveClientConfig(*config, context, nil, nil).ClientConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(rc)
}
