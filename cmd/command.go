// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

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
