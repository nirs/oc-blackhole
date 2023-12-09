// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"
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
	Targets []*TargetCluster
}

func NewCommand(blockedContext string, targetContexts []string, kubeconfig string) (*Command, error) {
	var err error

	err = validateContexts(blockedContext, targetContexts)
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

	var targets []*TargetCluster
	for _, target := range targetContexts {
		target, err := NewTargetCluster(config, target)
		if err != nil {
			return nil, err
		}

		targets = append(targets, target)
	}

	command := &Command{Cluster: cluster, Targets: targets}
	return command, nil
}

func (c *Command) InspectClusters() error {
	var err error

	// TODO: Run in parallel

	dbglog.Printf("Inspecting cluster %q ...", c.Cluster.Context)
	err = c.Cluster.Inspect()
	if err != nil {
		return err
	}

	for _, target := range c.Targets {
		dbglog.Printf("Inspecting target %q ...", target.Context)
		err = target.Inspect()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Command) BlockCluster() error {
	// TODO: Run in parallel

	addresses := c.Cluster.AllAddresses()

	for _, target := range c.Targets {
		dbglog.Printf("Blocking cluster %q in target %q ...", c.Cluster.Context, target.Context)
		for _, nodeName := range target.NodeNames {
			err := addBlackholeRoutes(target.Context, nodeName, addresses)
			if err != nil {
				return err
			}
			dbglog.Printf("Cluster %q blocked in node %q", c.Cluster.Context, nodeName)
		}
	}

	return nil
}

func (c *Command) UnblockCluster() error {
	// TODO: Run in parallel

	addresses := c.Cluster.AllAddresses()

	for _, target := range c.Targets {
		dbglog.Printf("Unblocking cluster %q in target %q ...", c.Cluster.Context, target.Context)
		for _, nodeName := range target.NodeNames {
			err := deleteBlackholeRoutes(target.Context, nodeName, addresses)
			if err != nil {
				return err
			}
			dbglog.Printf("Cluster %q unblocked in node %q", c.Cluster.Context, nodeName)
		}
	}

	return nil
}

func (c *Command) ClusterStatus() (map[string]*ClusterStatus, error) {
	// TODO: Run in parallel

	addresses := c.Cluster.AllAddresses()
	res := map[string]*ClusterStatus{}

	for _, target := range c.Targets {
		dbglog.Printf("Inspecting cluster %q status in target %q ...", c.Cluster.Context, target.Context)
		status := &ClusterStatus{
			Valid: true,
			Nodes: map[string]BlackholeStatus{},
		}
		var lastStatus BlackholeStatus

		for _, nodeName := range target.NodeNames {
			dbglog.Printf("Inspecting node %q ...", nodeName)
			routes, err := findBlackholeRoutes(target.Context, nodeName)
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

		res[target.Context] = status
	}

	return res, nil
}

func validateContexts(blockedContext string, targeContexts []string) error {
	targets := sets.New(targeContexts...)

	if len(targets) != len(targetContexts) {
		return fmt.Errorf("duplicate contexts: %v", targetContexts)
	}

	if targets.Has(blockedContext) {
		return fmt.Errorf("blocked cluster %q in target clusters %q",
			blockedContext, targetContexts)
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
