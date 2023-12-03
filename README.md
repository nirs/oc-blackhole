<!--
SPDX-FileCopyrightText: The oc-blackhole authors
SPDX-License-Identifier: Apache-2.0
-->

# oc-blackhole

This is an oc plugin simulating cluster failures by making a cluster
unreachable from other clusters.

## Quick start

Assuming a system with 3 clusters:

```sh
$ oc config get-contexts
CURRENT   NAME       CLUSTER                      AUTHINFO                                NAMESPACE
          cluster1   api-perf1-example-com:6443   kube:admin/api-perf1-example-com:6443   default
          cluster2   api-perf2-example-com:6443   kube:admin/api-perf2-example-com:6443   default
*         hub        api-perf3-example-com:6443   kube:admin/api-perf3-example-com:6443   default
```

Making `cluster1` unreachable on clusters `hub` and `cluster2`:

```sh
oc blackhole block cluster1 --context hub
oc blackhole block cluster1 --context cluster2
```

Now can test how the system handles the unreachable cluster.

When done, we can make the cluster available again:

```sh
oc blackhole unblock cluster1 --context hub
oc blackhole unblock cluster1 --context cluster2
```

## How a blackholed cluster looks like

When using OCM, it will report the cluster availability as `Unknown`:

```sh
$ oc get managedclusters --context hub
NAME            HUB ACCEPTED   MANAGED CLUSTER URLS                 JOINED   AVAILABLE   AGE
cluster1        true           https://api.perf1.example.com:6443   True     Unknown     8d
cluster2        true           https://api.perf2.example.com:6443   True     True        8d
local-cluster   true           https://api.perf3.example.com:6443   True     True        8d
```

Workloads trying to access the cluster will fail:

```sh
$ oc rook-ceph -n openshift-storage --context cluster2 dr health
Info: fetching the cephblockpools with mirroring enabled
Info: found "ocs-storagecluster-cephblockpool" cephblockpool with mirroring enabled
Info: running ceph status from peer cluster
Error: command terminated with exit code 1
timed out
Warning: failed to get ceph status from peer cluster, please check for network issues between the clusters
```

Or report warning status:

```sh
$ oc rook-ceph -n openshift-storage --context cluster2 \
    rbd mirror pool status -p ocs-storagecluster-cephblockpool
health: WARNING
daemon health: OK
image health: WARNING
images: 1 total
    1 unknown
```

## What's going on under the hood

When we blackhole a cluster, we get the cluster node addresses, and add
a `blackhole` ip route entry for every address on every node of the
target cluster.

```sh
$ oc get nodes --context hub
NAME                      STATUS   ROLES                  AGE   VERSION
perf3-lhps4-acm-0-5jdjh   Ready    worker                 9d    v1.27.6+f67aeb3
perf3-lhps4-acm-0-l8f9f   Ready    worker                 9d    v1.27.6+f67aeb3
perf3-lhps4-acm-0-vzrq2   Ready    worker                 9d    v1.27.6+f67aeb3
perf3-lhps4-master-0      Ready    control-plane,master   9d    v1.27.6+f67aeb3
perf3-lhps4-master-1      Ready    control-plane,master   9d    v1.27.6+f67aeb3
perf3-lhps4-master-2      Ready    control-plane,master   9d    v1.27.6+f67aeb3

$ oc debug node/perf3-lhps4-acm-0-vzrq2 -- sh -c 'ip route | grep ^blackhole'
Starting pod/perf3-lhps4-acm-0-vzrq2-debug ...
To use host binaries, run `chroot /host`
blackhole 10.70.56.101
blackhole 10.70.56.149
blackhole 10.70.56.168
blackhole 10.70.56.176
blackhole 10.70.56.187
blackhole 10.70.56.212

Removing debug pod ...

$ oc debug node/perf3-lhps4-acm-0-vzrq2 -- ping 10.70.56.168
Starting pod/perf3-lhps4-acm-0-vzrq2-debug ...
To use host binaries, run `chroot /host`
connect: Invalid argument

Removing debug pod ...
error: non-zero exit code from debug container
```

Unlocking the cluster delete the `blackhole` route entries.
