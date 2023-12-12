<!--
SPDX-FileCopyrightText: The oc-blackhole authors
SPDX-License-Identifier: Apache-2.0
-->

# oc-blackhole

This is an oc plugin simulating cluster failures by making a cluster
unreachable from other clusters.

[![asciicast](https://asciinema.org/a/626178.svg)](https://asciinema.org/a/626178)

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
oc blackhole block cluster1 --contexts hub,cluster2
```

Now we can test how the system handles the unreachable cluster.

When done, we can make the cluster reachable again:

```sh
oc blackhole unblock cluster1 --contexts hub,cluster2
```

To inspect the status of the cluster:

```sh
$ oc blackhole show cluster1 --contexts hub,cluster2
status:
  cluster: cluster1
  targets:
    - name: hub
      valid: true
      nodes:
        - name: perf1-d8zsg-master-0
          status: unblocked
        - name: perf1-d8zsg-master-1
          status: unblocked
        - name: perf1-d8zsg-master-2
          status: unblocked
        - name: perf1-d8zsg-ocs-0-6nzrx
          status: unblocked
        - name: perf1-d8zsg-ocs-0-kd5xq
          status: unblocked
        - name: perf1-d8zsg-ocs-0-wl7cn
          status: unblocked
    - name: cluster2
      valid: true
      nodes:
        - name: perf3-xxkb8-master-0
          status: unblocked
        - name: perf3-xxkb8-master-1
          status: unblocked
        - name: perf3-xxkb8-master-2
          status: unblocked
        - name: perf3-xxkb8-ocs-0-2gvdz
          status: unblocked
        - name: perf3-xxkb8-ocs-0-hwb29
          status: unblocked
        - name: perf3-xxkb8-ocs-0-nj2z5
          status: unblocked
```

## How a blackholed cluster looks like

Accessing the API server from the target host will fail:

```sh
$ oc cluster-info --context cluster2
Kubernetes control plane is running at https://api.perf2.example.com:6443

$ oc debug node/perf3-lhps4-acm-0-5jdjh -- curl --silent --show-error \
    https://api.perf2.example.com:6443 2>/dev/null
curl: (7) Couldn't connect to server
```

Accessing routes on the blocked cluster will fail:

```sh
$ oc get route s3 -n openshift-storage --context cluster2
NAME   HOST/PORT                                     PATH   SERVICES   PORT       TERMINATION       WILDCARD
s3     s3-openshift-storage.apps.perf2.example.com          s3         s3-https   reencrypt/Allow   None

$ oc debug node/perf3-lhps4-acm-0-5jdjh -- curl curl --silent --show-error \
    https://s3-openshift-storage.apps.perf2.example.com/odrbucket-b1b922184baf/ 2>/dev/null
curl: (6) Could not resolve host: curl
curl: (7) Couldn't connect to server
```

When using OCM, it will report the cluster availability as `Unknown` after few minutes:

```sh
$ oc get managedclusters --context hub
NAME            HUB ACCEPTED   MANAGED CLUSTER URLS                 JOINED   AVAILABLE   AGE
cluster1        true           https://api.perf1.example.com:6443   True     Unknown     8d
cluster2        true           https://api.perf2.example.com:6443   True     True        8d
local-cluster   true           https://api.perf3.example.com:6443   True     True        8d
```

When using Rook Ceph pool configured for RBD mirroring, the `dr health` command
will fail:

```sh
$ oc rook-ceph -n openshift-storage --context cluster2 dr health
Info: fetching the cephblockpools with mirroring enabled
Info: found "ocs-storagecluster-cephblockpool" cephblockpool with mirroring enabled
Info: running ceph status from peer cluster
Error: command terminated with exit code 1
timed out
Warning: failed to get ceph status from peer cluster, please check for network issues between the clusters
```

Ceph status will report `WARNING` health:

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

When we blackhole a cluster, we get the cluster node addresses, api
server address, and route addresses, and add a `blackhole` ip route
entry for every address on every node of the target cluster.

```sh
$ oc get nodes --context hub
NAME                      STATUS   ROLES                  AGE   VERSION
perf3-lhps4-acm-0-5jdjh   Ready    worker                 9d    v1.27.6+f67aeb3
perf3-lhps4-acm-0-l8f9f   Ready    worker                 9d    v1.27.6+f67aeb3
perf3-lhps4-acm-0-vzrq2   Ready    worker                 9d    v1.27.6+f67aeb3
perf3-lhps4-master-0      Ready    control-plane,master   9d    v1.27.6+f67aeb3
perf3-lhps4-master-1      Ready    control-plane,master   9d    v1.27.6+f67aeb3
perf3-lhps4-master-2      Ready    control-plane,master   9d    v1.27.6+f67aeb3

$ oc debug node/perf3-lhps4-acm-0-vzrq2 -- ip route show type blackhole
blackhole 10.70.56.101
blackhole 10.70.56.149
blackhole 10.70.56.168
blackhole 10.70.56.176
blackhole 10.70.56.187
blackhole 10.70.56.212

$ oc debug node/perf3-lhps4-acm-0-vzrq2 -- ping 10.70.56.168
connect: Invalid argument
```

Unlocking the cluster delete the `blackhole` route entries.
