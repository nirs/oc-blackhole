// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
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

	// `ip route replace` and `ip route del` handle both ipv4 and ipv6 routes,
	// but `ip route show` return only ipv4 routes.

	script := `
	ip -4 route show type blackhole
	ip -6 route show type blackhole
	`
	out, err := execScript(context, nodeName, script)
	if err != nil {
		return nil, err
	}

	res := sets.New[string]()

	for _, line := range strings.Split(string(out), "\n") {
		// We want the second field
		// - ipv4: "blackhole 172.217.22.14 "
		// - ipv6: "blackhole 2a00:1450:4028:809::200e dev lo metric 1024 pref medium "
		fields := strings.Fields(line)
		res.Insert(fields[1])
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
