# SPDX-FileCopyrightText: The oc-blackhole authors
# SPDX-License-Identifier: Apache-2.0

"""
oc-blackhole demo.

Installing requirements:

    python3 -m venv ~/.venv/oc-blackhole
    source ~/.venv/oc-blackhole/bin/activate
    pip install --upgrade pip nohands

Run using:

    python3 demo.py

"""

from nohands import *

run("clear")
msg("### OC BLACKHOLE PLUGIN DEMO ###", color=YELLOW)
msg()
msg("We have 3 OpenShift clusters: 'perf1', 'perf2', and 'perf3'.")
msg("Let's make cluster 'perf2' unreachable from other clusters:")
msg()
run("oc", "blackhole", "block", "perf2", "--progress", "--contexts", "perf1,perf3")
msg()
msg("What happened? we can show the cluster status:")
msg()
run("oc", "blackhole", "show", "perf2", "--progress", "--contexts", "perf1,perf3")
msg()
msg("Let's test how the system handles the unreachable cluster.")
msg("We can use the rook-ceph oc plugin to check dr health:")
msg()
run("oc", "rook-ceph", "-n", "openshift-storage", "--context", "perf3", "dr", "health")
msg()
msg("Ceph is not happy!")
msg("Let's make cluster 'perf2' reachable again:")
msg()
run("oc", "blackhole", "unblock", "perf2", "--progress", "--contexts", "perf1,perf3")
msg()
msg("Now we can test how the system handles the recovered cluster.")
msg()
msg("Created with https://pypi.org/project/nohands/", color=GREY)
