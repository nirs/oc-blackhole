// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

func addBlackholeRoutes(context string, nodeName string, addresses []string) error {
	dbglog.Printf("blocking addresses in node %s", nodeName)

	var sb strings.Builder
	for _, address := range addresses {
		// `replace` is idempotent, no need to check for existing blackholes.
		sb.WriteString("ip route replace blackhole " + address + "\n")
	}

	_, err := execScript(context, nodeName, sb.String())
	if err != nil {
		return err
	}

	return nil
}

func deleteBlackholeRoutes(context string, nodeName string, addresses []string) error {
	dbglog.Printf("unblocking addresses in node %s", nodeName)

	// `ip route del`` is not idempotent, so we build a command with existing
	// blackholed addresses.

	blackholes, err := findBlackholeRoutes(context, nodeName)
	if err != nil {
		return err
	}

	var sb strings.Builder
	for _, address := range addresses {
		if blackholes.Has(address) {
			sb.WriteString("ip route del blackhole " + address + "\n")
		}
	}

	if sb.Len() == 0 {
		dbglog.Printf("No address to unblock on node %s", nodeName)
		return nil
	}

	_, err = execScript(context, nodeName, sb.String())
	if err != nil {
		return err
	}

	return nil
}

func findBlackholeRoutes(context string, nodeName string) (sets.Set[string], error) {
	dbglog.Printf("Looking up blackholes on node %s", nodeName)

	res := sets.New[string]()

	cmd := exec.Command(
		"oc",
		"debug",
		"node/"+nodeName,
		"--context",
		context,
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

func execScript(context string, nodeName string, script string) ([]byte, error) {
	cmd := exec.Command(
		"oc",
		"debug",
		"node/"+nodeName,
		"--context",
		context,
		"--",
		"sh",
		"-c",
		script,
	)

	dbglog.Printf("Running command on node %s: %s", nodeName, cmd.Args)

	return cmd.Output()
}
