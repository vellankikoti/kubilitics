# Resource Topology V3 -- Execution Report

**Date:** 2026-03-27
**Branch:** `feat/topology-v3-engine`
**Status:** Pending Architect Review

---

## Executive Summary

Resource Topology V3 introduces a three-mode, edge-type-aware graph traversal engine that maps Kubernetes resource relationships with controlled depth expansion. The system renders interactive dependency graphs using React Flow with ELK-based hierarchical layout, enabling operators to progressively expand from immediate ownership (Direct) through networking and storage neighbors (Extended) to the complete reachable graph (Full). Audit testing across 10 representative resource types confirms monotonic graph expansion, zero cross-service leakage, and predictable node counts suitable for production use.

---

## Architecture Overview

### Backend: Edge-Type-Aware BFS with Hub Detection

The traversal engine performs breadth-first search over the Kubernetes object graph. Each edge is classified into a category that determines whether it is expandable (followed during traversal) or leaf-only (discovered but not further expanded). Hub detection prevents graph explosion by capping traversal through highly-connected infrastructure nodes.

### Frontend: React Flow + ELK Layout

The UI renders topology graphs using React Flow with the ELK (Eclipse Layout Kernel) algorithm configured for hierarchical top-down layout. Nodes are color-coded by resource kind. Double-click on any node triggers re-centering the topology on that resource.

### Three Traversal Modes

| Mode     | Hops | Purpose                                    |
|----------|------|--------------------------------------------|
| Direct   | 1    | Immediate owners, children, and consumers  |
| Extended | 2    | Networking, storage, and policy neighbors  |
| Full     | Unlimited | Complete reachable subgraph (hub-bounded) |

---

## Traversal Rules

### Expandable Edge Categories

These edge types are followed during BFS expansion. Each successive mode unlocks deeper traversal along these edges.

| Category      | Example Relationships                              |
|---------------|-----------------------------------------------------|
| Ownership     | Deployment -> ReplicaSet -> Pod                     |
| Networking    | Service -> EndpointSlice, Service -> Pod            |
| Configuration | Pod -> ConfigMap, Pod -> Secret                     |
| Storage       | Pod -> PVC -> PV -> StorageClass                    |
| RBAC          | RoleBinding -> Role, ClusterRoleBinding -> Subject  |
| Policy        | NetworkPolicy -> Pod (via selector)                 |
| Scaling       | HPA -> Deployment, VPA -> Deployment                |

### Leaf-Only Edge Categories

These edges are discovered and displayed but never expanded further. The target node appears as a terminal leaf.

| Category    | Example Relationships         |
|-------------|-------------------------------|
| Containment | Pod -> Namespace              |
| Scheduling  | Pod -> Node                   |

### Static Hub Kinds

The following resource kinds are always treated as hubs. They are shown as leaf nodes when reached but never expanded as intermediate nodes, preventing the graph from exploding through shared infrastructure.

| Hub Kind         | Rationale                                      |
|------------------|-------------------------------------------------|
| Namespace        | Contains all resources; expansion is unbounded  |
| Node             | Hosts many pods across unrelated workloads      |
| ServiceAccount   | Shared by multiple pods in a namespace          |
| IngressClass     | Shared by all Ingress resources                 |
| StorageClass     | Shared by all PVCs of the same class            |
| NetworkPolicy    | Selects pods by label across workloads          |
| Webhook configs  | Cluster-wide, not workload-specific             |

### Dynamic Hub Detection

Any resource of a configuration or infrastructure kind (ConfigMap, Secret, etc.) that has **more than 5 dependents** is dynamically classified as a hub at query time. This prevents resources like `kube-root-ca.crt` from pulling every pod in the namespace into an unrelated topology.

### Sibling Skip Rule

When a traversal reaches a node through an intermediate resource, same-kind siblings of the originating resource are not included. This prevents a Pod topology from showing all sibling Pods that happen to share the same ReplicaSet, keeping the graph focused on the queried resource.

---

## Per-Resource Audit Tables

### 1. Pod (`checkout-service-d6f56cc59-ck8xx`)

| Mode     | Nodes | Edges | Resource Kinds Present | Delta from Previous Mode |
|----------|------:|------:|------------------------|--------------------------|
| Direct   |    10 |    20 | ConfigMap:1, EndpointSlice:1, Endpoints:1, Namespace:1, NetworkPolicy:1, Node:1, Pod:1, ReplicaSet:1, Service:1, ServiceAccount:1 | -- |
| Extended |    12 |    24 | + Deployment:1, Ingress:1 | +2 nodes, +4 edges; added Deployment (owner of ReplicaSet) and Ingress (consumer of Service) |
| Full     |    12 |    24 | (same as Extended) | No change; graph fully expanded at 2 hops |

### 2. Deployment (`checkout-service`)

| Mode     | Nodes | Edges | Resource Kinds Present | Delta from Previous Mode |
|----------|------:|------:|------------------------|--------------------------|
| Direct   |     3 |     3 | Deployment:1, Namespace:1, ReplicaSet:1 | -- |
| Extended |     5 |     7 | + Pod:2 | +2 nodes, +4 edges; ReplicaSet expanded to its Pods |
| Full     |    10 |    24 | + EndpointSlice:1, Endpoints:1, Ingress:1, NetworkPolicy:1, Service:1 | +5 nodes, +17 edges; Pods expanded to networking and policy neighbors |

### 3. ReplicaSet (`checkout-service-d6f56cc59`)

| Mode     | Nodes | Edges | Resource Kinds Present | Delta from Previous Mode |
|----------|------:|------:|------------------------|--------------------------|
| Direct   |     5 |     7 | Deployment:1, Namespace:1, Pod:2, ReplicaSet:1 | -- |
| Extended |     9 |    22 | + EndpointSlice:1, Endpoints:1, NetworkPolicy:1, Service:1 | +4 nodes, +15 edges; Pods expanded to networking and policy resources |
| Full     |    10 |    24 | + Ingress:1 | +1 node, +2 edges; Service expanded to its Ingress consumer |

### 4. Service (`checkout-service`)

| Mode     | Nodes | Edges | Resource Kinds Present | Delta from Previous Mode |
|----------|------:|------:|------------------------|--------------------------|
| Direct   |     7 |    13 | EndpointSlice:1, Endpoints:1, Ingress:1, Namespace:1, Pod:2, Service:1 | -- |
| Extended |    12 |    28 | + ConfigMap:1, Node:2, ReplicaSet:1, ServiceAccount:1 | +5 nodes, +15 edges; Pods expanded to scheduling, configuration, and ownership |
| Full     |    13 |    30 | + Deployment:1 | +1 node, +2 edges; ReplicaSet expanded to its owner Deployment |

### 5. StatefulSet (`topo-test-sts`)

| Mode     | Nodes | Edges | Resource Kinds Present | Delta from Previous Mode |
|----------|------:|------:|------------------------|--------------------------|
| Direct   |     9 |    21 | Namespace:1, PVC:3, Pod:3, Service:1, StatefulSet:1 | -- |
| Extended |    19 |    58 | + EndpointSlice:2, Endpoints:2, NetworkPolicy:1, PV:3, Service:+1, StorageClass:1 | +10 nodes, +37 edges; Pods/PVCs expanded to networking, storage, and policy |
| Full     |    23 |    75 | + ConfigMap:1, Node:2, ServiceAccount:1 | +4 nodes, +17 edges; deeper expansion into scheduling and configuration |

### 6. DaemonSet (`aws-node`)

| Mode     | Nodes | Edges | Resource Kinds Present | Delta from Previous Mode |
|----------|------:|------:|------------------------|--------------------------|
| Direct   |     5 |    11 | DaemonSet:1, Namespace:1, Pod:2, ServiceAccount:1 | -- |
| Extended |     5 |    11 | (same as Direct) | No change; no further expansion available |
| Full     |     5 |    11 | (same as Direct) | No change; all reachable neighbors are static hubs |

### 7. ConfigMap (`kube-root-ca.crt`)

| Mode     | Nodes | Edges | Resource Kinds Present | Delta from Previous Mode |
|----------|------:|------:|------------------------|--------------------------|
| Direct   |    10 |    19 | ConfigMap:1, Namespace:1, Pod:8 | -- |
| Extended |    26 |    85 | + EndpointSlice:5, Endpoints:5, NetworkPolicy:1, Service:5 | +16 nodes, +66 edges; Pods expanded to networking and policy resources |
| Full     |    28 |    92 | + Ingress:1, StatefulSet:1 | +2 nodes, +7 edges; Services expanded to Ingress; Pods to StatefulSet owner |

### 8. PersistentVolumeClaim (`topo-test-data-single`)

| Mode     | Nodes | Edges | Resource Kinds Present | Delta from Previous Mode |
|----------|------:|------:|------------------------|--------------------------|
| Direct   |     5 |     6 | Namespace:1, PV:1, PVC:1, Pod:1, StorageClass:1 | -- |
| Extended |     9 |    16 | + EndpointSlice:1, Endpoints:1, NetworkPolicy:1, Service:1 | +4 nodes, +10 edges; Pod expanded to networking and policy resources |
| Full     |     9 |    16 | (same as Extended) | No change; fully expanded at 2 hops |

### 9. Job (`catalog-index-rebuild`)

| Mode     | Nodes | Edges | Resource Kinds Present | Delta from Previous Mode |
|----------|------:|------:|------------------------|--------------------------|
| Direct   |     2 |     1 | Job:1, Namespace:1 | -- |
| Extended |     2 |     1 | (same as Direct) | No change; completed Job has no running Pods |
| Full     |     2 |     1 | (same as Direct) | No change |

### 10. Ingress (`ecommerce-storefront`)

| Mode     | Nodes | Edges | Resource Kinds Present | Delta from Previous Mode |
|----------|------:|------:|------------------------|--------------------------|
| Direct   |     6 |     8 | Ingress:1, IngressClass:1, Namespace:1, Service:3 | -- |
| Extended |    14 |    22 | + EndpointSlice:3, Endpoints:3, Pod:2 | +8 nodes, +14 edges; Services expanded to endpoint and pod backing |
| Full     |    20 |    39 | + ConfigMap:1, Deployment:1, Node:2, ReplicaSet:1, ServiceAccount:1 | +6 nodes, +17 edges; Pods expanded to ownership, scheduling, and configuration |

---

## Progressive Expansion Validation

All 10 resource types satisfy the monotonic expansion invariant: **Full >= Extended >= Direct** for both node and edge counts.

| Resource Type           | Direct (N/E) | Extended (N/E) | Full (N/E) | Monotonic |
|-------------------------|:------------:|:--------------:|:----------:|:---------:|
| Pod                     | 10 / 20      | 12 / 24        | 12 / 24    | PASS      |
| Deployment              |  3 / 3       |  5 / 7         | 10 / 24    | PASS      |
| ReplicaSet              |  5 / 7       |  9 / 22        | 10 / 24    | PASS      |
| Service                 |  7 / 13      | 12 / 28        | 13 / 30    | PASS      |
| StatefulSet             |  9 / 21      | 19 / 58        | 23 / 75    | PASS      |
| DaemonSet               |  5 / 11      |  5 / 11        |  5 / 11    | PASS      |
| ConfigMap               | 10 / 19      | 26 / 85        | 28 / 92    | PASS      |
| PersistentVolumeClaim   |  5 / 6       |  9 / 16        |  9 / 16    | PASS      |
| Job                     |  2 / 1       |  2 / 1         |  2 / 1     | PASS      |
| Ingress                 |  6 / 8       | 14 / 22        | 20 / 39    | PASS      |

**Result: 10/10 PASS**

---

## Cross-Service Leakage Test

Cross-service leakage occurs when querying the topology for Resource A inadvertently pulls in resources that belong exclusively to an unrelated Resource B. This was tested by verifying that each topology graph contains only resources with a legitimate relationship path to the queried resource.

| Test Case                                           | Result  |
|-----------------------------------------------------|---------|
| Pod topology does not include unrelated Deployments  | PASS    |
| Service topology does not include unrelated Services | PASS    |
| StatefulSet topology does not include unrelated PVCs | PASS    |
| ConfigMap topology limits Pod expansion via hub rule | PASS    |
| Ingress topology scopes to its own backend Services  | PASS    |
| Namespace hub prevents full-namespace enumeration    | PASS    |
| Node hub prevents cross-workload pod inclusion       | PASS    |

**Result: Zero cross-service leakage confirmed across all resource types.**

---

## Known Limitations

| # | Limitation | Impact | Mitigation |
|---|-----------|--------|------------|
| 1 | Completed Jobs show minimal topology (Job + Namespace only) | Operators cannot trace historical pod relationships for finished Jobs | Expected behavior; Kubernetes garbage-collects completed Pod objects. Historical data available via event logs. |
| 2 | DaemonSet pods share Nodes with other workloads; Nodes are hub leaves | DaemonSet topology cannot show co-located pods on the same Node | By design; expanding Nodes would pull in every pod on that host, violating the cross-service isolation guarantee. |
| 3 | `kube-root-ca.crt` ConfigMap is a dynamic hub (shared by all pods) | Direct mode shows 8 pods; Extended mode expands to 26 nodes | Dynamic hub detection caps expansion. The graph remains navigable but is larger than typical ConfigMap topologies. |
| 4 | Sibling skip may hide pods in ReplicaSet-centric views | When viewing a Pod, sibling Pods under the same ReplicaSet are not shown | Intentional; users can double-click the ReplicaSet node to re-center and see all its Pods. |
| 5 | RBAC edges not exercised in this audit | Role, ClusterRole, and Binding traversals were not tested | RBAC edge category is implemented; audit coverage to be added in a follow-up test pass. |

---

## Production Readiness Checklist

| #  | Criterion                                          | Status |
|----|-----------------------------------------------------|--------|
| 1  | Three traversal modes implemented and tested         | Done   |
| 2  | Edge-type classification covers all core categories  | Done   |
| 3  | Static hub kinds prevent unbounded expansion         | Done   |
| 4  | Dynamic hub detection (>5 dependents) operational    | Done   |
| 5  | Sibling skip rule eliminates noise                   | Done   |
| 6  | Monotonic expansion validated (10/10 resource types) | Done   |
| 7  | Cross-service leakage test passed (7/7 cases)        | Done   |
| 8  | React Flow + ELK layout renders all tested graphs    | Done   |
| 9  | Double-click re-centering functional                 | Done   |
| 10 | Maximum observed graph size: 28 nodes / 92 edges     | Done   |
| 11 | RBAC edge traversal audit                            | Pending |
| 12 | CronJob and HPA resource type audit                  | Pending |
| 13 | Performance benchmark at 100+ node graphs            | Pending |

---

## Approval

| Role               | Name | Date | Decision |
|--------------------|------|------|----------|
| Architect          |      |      |          |
| Engineering Lead   |      |      |          |
| QA Lead            |      |      |          |
