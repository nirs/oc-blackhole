// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type BlackholeStatus string

const (
	// None of cluster addresses are blocked in target cluster.
	StatusUnblocked = BlackholeStatus("unblocked")

	// All cluster addresses are blocked in target cluster.
	StatusBlocked = BlackholeStatus("blocked")

	// Some of cluster addresses are blocked in target cluster.
	StatusPartlyBlocked = BlackholeStatus("partly-blocked")
)

type ClusterStatus struct {
	Valid bool
	Nodes map[string]BlackholeStatus
}

type Command struct {
	Cluster *BlockedCluster
	Target  *TargetCluster
}

func NewCommand(blockedContext string, targetContext string, kubeconfig string) (*Command, error) {
	var err error

	err = validateContexts(blockedContext, targetContext)
	if err != nil {
		return nil, err
	}

	config, err := loadConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	cluster, err := NewBlockedCluster(config, blockedContext)
	if err != nil {
		return nil, err
	}

	target, err := NewTargetCluster(config, targetContext)
	if err != nil {
		return nil, err
	}

	command := &Command{Cluster: cluster, Target: target}
	return command, nil
}

func (c *Command) InspectClusters() error {
	var err error

	// TODO: Run in parallel

	err = c.Cluster.Inspect()
	if err != nil {
		return err
	}

	err = c.Target.Inspect()
	if err != nil {
		return err
	}

	return nil
}

func (c *Command) BlockCluster() error {
	// TODO: Run in parallel

	addresses := c.Cluster.AllAddresses()

	for _, nodeName := range c.Target.NodeNames {
		err := addBlackholeRoutes(c.Target.Context, nodeName, addresses)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Command) UnblockCluster() error {
	// TODO: Run in parallel

	addresses := c.Cluster.AllAddresses()

	for _, nodeName := range c.Target.NodeNames {
		err := deleteBlackholeRoutes(c.Target.Context, nodeName, addresses)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Command) ClusterStatus() (*ClusterStatus, error) {
	// TODO: Run in parallel

	addresses := c.Cluster.AllAddresses()
	status := &ClusterStatus{
		Valid: true,
		Nodes: map[string]BlackholeStatus{},
	}
	var lastStatus BlackholeStatus

	for _, nodeName := range c.Target.NodeNames {
		routes, err := findBlackholeRoutes(c.Target.Context, nodeName)
		if err != nil {
			return nil, err
		}

		if routes.HasAll(addresses...) {
			status.Nodes[nodeName] = StatusBlocked
		} else if routes.HasAny(addresses...) {
			status.Nodes[nodeName] = StatusPartlyBlocked
			status.Valid = false
		} else {
			status.Nodes[nodeName] = StatusUnblocked
		}

		if lastStatus != "" && lastStatus != status.Nodes[nodeName] {
			status.Valid = false
		}

		lastStatus = status.Nodes[nodeName]
	}

	return status, nil
}

func validateContexts(blockedContext string, targeContext string) error {
	if blockedContext == targetContext {
		return fmt.Errorf("blocked cluster must be different from target cluster")
	}
	return nil
}

func loadConfig(kubeconfig string) (*api.Config, error) {
	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}
