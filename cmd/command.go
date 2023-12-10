// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"os"

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
	Cluster  *BlockedCluster
	Targets  []*TargetCluster
	progress *Progress
}

func NewCommand(blockedContext string, targetContexts []string, kubeconfig string, showProgress bool) (*Command, error) {
	var err error

	out := io.Discard
	if showProgress {
		out = os.Stderr
	}

	// We start with inderminate progress, since we don't know yet the number of nodes.
	progress := NewProgress("setting up", 0, out)

	defer func() {
		if err != nil {
			progress.Clear()
		}
	}()

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

	command := &Command{Cluster: cluster, Targets: targets, progress: progress}
	return command, nil
}

func (c *Command) inspectClusters() error {
	errors := make(chan error)

	go func() {
		dbglog.Printf("Inspecting cluster %q ...", c.Cluster.Context)
		errors <- c.Cluster.Inspect()
	}()

	for i := range c.Targets {
		target := c.Targets[i]
		go func() {
			dbglog.Printf("Inspecting target %q ...", target.Context)
			errors <- target.Inspect()
		}()
	}

	for i := 0; i < len(c.Targets)+1; i += 1 {
		if err := <-errors; err != nil {
			return err
		}
	}

	return nil
}

func (c *Command) BlockCluster() error {
	defer c.progress.Clear()

	c.progress.SetDescription("inspecting clusters")

	if err := c.inspectClusters(); err != nil {
		return err
	}

	tasks := c.targetNodeCount()
	c.progress.SetTasks(uint(tasks))
	c.progress.SetDescription("modifying nodes")

	addresses := c.Cluster.AllAddresses()
	errors := make(chan error)

	for i := range c.Targets {
		target := c.Targets[i]
		dbglog.Printf("Blocking cluster %q in target %q ...", c.Cluster.Context, target.Context)

		for j := range target.NodeNames {
			nodeName := target.NodeNames[j]

			go func() {
				err := addBlackholeRoutes(target.Context, nodeName, addresses)
				if err == nil {
					dbglog.Printf("Cluster %q blocked in node %q", c.Cluster.Context, nodeName)
				}
				c.progress.Add(1)
				errors <- err
			}()
		}
	}

	return firstError(errors, tasks)
}

func (c *Command) UnblockCluster() error {
	defer c.progress.Clear()

	c.progress.SetDescription("inspecting clusters")

	if err := c.inspectClusters(); err != nil {
		return err
	}

	tasks := c.targetNodeCount()
	c.progress.SetTasks(uint(tasks))
	c.progress.SetDescription("modifying nodes")

	addresses := c.Cluster.AllAddresses()
	errors := make(chan error)

	for i := range c.Targets {
		target := c.Targets[i]
		dbglog.Printf("Unblocking cluster %q in target %q ...", c.Cluster.Context, target.Context)

		for j := range target.NodeNames {
			nodeName := target.NodeNames[j]

			go func() {
				err := deleteBlackholeRoutes(target.Context, nodeName, addresses)
				if err == nil {
					dbglog.Printf("Cluster %q unblocked in node %q", c.Cluster.Context, nodeName)
				}
				c.progress.Add(1)
				errors <- err
			}()
		}
	}

	return firstError(errors, tasks)
}

func firstError(errors <-chan error, count int) error {
	for i := 0; i < count; i += 1 {
		if err := <-errors; err != nil {
			return err
		}
	}

	return nil
}

type Result struct {
	Context string
	Node    string
	Routes  sets.Set[string]
	Err     error
}

func (c *Command) ClusterStatus() (map[string]*ClusterStatus, error) {
	defer c.progress.Clear()

	c.progress.SetDescription("inspecting clusters")

	if err := c.inspectClusters(); err != nil {
		return nil, err
	}

	tasks := c.targetNodeCount()
	c.progress.SetTasks(uint(tasks))
	c.progress.SetDescription("inspecting nodes")

	results := make(chan *Result)

	for i := range c.Targets {
		target := c.Targets[i]
		dbglog.Printf("Inspecting cluster %q status in target %q ...", c.Cluster.Context, target.Context)

		for j := range target.NodeNames {
			nodeName := target.NodeNames[j]

			go func() {
				dbglog.Printf("Inspecting node %q ...", nodeName)
				routes, err := findBlackholeRoutes(target.Context, nodeName)
				c.progress.Add(1)
				results <- &Result{Context: target.Context, Node: nodeName, Routes: routes, Err: err}
			}()
		}
	}

	return c.collectResults(results, tasks)
}

func (c *Command) collectResults(results <-chan *Result, count int) (map[string]*ClusterStatus, error) {
	res := map[string]*ClusterStatus{}

	for _, target := range c.Targets {
		res[target.Context] = &ClusterStatus{
			Valid: true,
			Nodes: map[string]BlackholeStatus{},
		}
	}

	addresses := c.Cluster.AllAddresses()

	for i := 0; i < count; i += 1 {
		result := <-results
		if result.Err != nil {
			return nil, result.Err
		}

		status := res[result.Context]
		var newStatus BlackholeStatus

		if result.Routes.HasAll(addresses...) {
			newStatus = StatusBlocked
		} else if result.Routes.HasAny(addresses...) {
			newStatus = StatusPartlyBlocked
			status.Valid = false
		} else {
			newStatus = StatusUnblocked
		}

		if lastStatus, ok := status.Nodes[result.Node]; ok {
			if lastStatus != newStatus {
				status.Valid = false
			}
		}

		status.Nodes[result.Node] = newStatus
	}

	return res, nil
}

func (c *Command) targetNodeCount() int {
	count := 0
	for _, target := range c.Targets {
		count += len(target.NodeNames)
	}
	return count
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
