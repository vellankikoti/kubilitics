# Resource-Focused Topology — Single Source of Truth

> **Status**: DEFINITIVE SPECIFICATION
> **Scope**: Resource-specific topology only (NOT cluster/namespace/workload views)
> **Authority**: This document overrides all other topology docs for resource-focused behavior

---

## 1. The Absolute Rule

When a user opens topology for a specific resource (Pod, Deployment, ReplicaSet, Service, etc.):

**Every single node in the graph MUST be connected to THAT resource — directly or through its dependency chain.**

- If a node is NOT part of this resource's dependency chain → it MUST NOT appear
- No sibling pods. No unrelated deployments. No cluster-wide noise. Zero exceptions.

---

## 2. Why This Matters

Resource-focused topology is the core USP of Kubilitics. The user selects ONE resource and expects to see ONLY that resource's world. The moment unrelated resources appear, the topology becomes a noisy cluster map — defeating its entire purpose.

---

## 3. The Three Modes

Resource-focused topology operates in three progressive modes. Each mode expands the graph, but **NEVER beyond the selected resource's dependency chain**.

### 3.1 DIRECT Mode (depth = 1)

Show ONLY the selected resource and its immediate neighbors (1 hop).

**Example — Pod selected:**

```
                    ┌─────────────┐
                    │  ReplicaSet │
                    └──────┬──────┘
                           │ owns
     ┌──────────┐    ┌─────┴─────┐    ┌──────────┐
     │ ConfigMap ├────┤  THIS POD ├────┤  Secret  │
     └──────────┘    └─────┬─────┘    └──────────┘
              mounts       │  runs on
                    ┌──────┴──────┐
                    │    Node     │
                    └─────────────┘
```

**Visible**: ReplicaSet, Service (if selects this pod), ConfigMap, Secret, PVC, ServiceAccount, Node
**NOT visible**: Other pods (even in same ReplicaSet), other deployments, Namespace node

### 3.2 EXTENDED Mode (depth = 2)

Expand from nodes already shown in DIRECT mode — one more hop outward.

**Example — Pod selected:**

```
     ┌────────────┐
     │ Deployment  │ ← expanded from ReplicaSet
     └──────┬─────┘
            │ owns
     ┌──────┴──────┐
     │  ReplicaSet │
     └──────┬──────┘
            │ owns
     ┌──────┴──────┐    ┌─────────┐    ┌───────────┐
     │  THIS POD   ├────┤ Service ├────┤ Endpoints │ ← expanded from Service
     └──────┬──────┘    └─────────┘    └───────────┘
            │
     ┌──────┴──────┐
     │    PVC      │
     └──────┬──────┘
            │ binds
     ┌──────┴──────┐
     │     PV      │ ← expanded from PVC
     └─────────────┘
```

**New at depth 2**: Deployment (via ReplicaSet), Endpoints (via Service), PV (via PVC), Ingress (via Service), RBAC chain (via ServiceAccount)

**CRITICAL**: Expansion MUST NOT introduce unrelated resources. See Section 5 for blocked traversals.

### 3.3 FULL Mode (depth = unlimited, BFS/DFS)

Show the complete connected subgraph of the selected resource — every reachable node through valid, non-blocked edges.

**Still filtered**: Even in FULL mode, hub-based expansion is blocked (Section 5). The graph contains ONLY resources that are part of this resource's dependency chain.

---

## 4. Valid Relationship Types (Edge Types)

These are the Kubernetes relationships that form edges in the topology graph:

| Relationship | Category | Example | Traversal Direction |
|---|---|---|---|
| `owns` / `owned_by` | ownership | Deployment → ReplicaSet → Pod | bidirectional |
| `selects` | networking | Service → Pod (via label selector) | bidirectional |
| `mounts` | config | Pod → ConfigMap, Pod → Secret, Pod → PVC | bidirectional |
| `configures` | config | ConfigMap/Secret → Pod (via envFrom) | bidirectional |
| `routes` | networking | Ingress → Service | bidirectional |
| `targets` | networking | Endpoints/EndpointSlice → Pod | bidirectional |
| `scheduling` | scheduling | Pod → Node | **forward only** (see Section 5) |
| `service_account` | rbac | Pod → ServiceAccount | bidirectional |
| `permits` / `grants` / `binds_to` | rbac | RoleBinding → Role, RoleBinding → ServiceAccount | bidirectional |
| `scales` | scaling | HPA → Deployment/StatefulSet | bidirectional |
| `stores` / `provisioned_by` | storage | PVC → PV, PV → StorageClass | bidirectional |
| `calls` | networking | Webhook → Service | bidirectional |
| `namespace` | containment | Resource → Namespace | **BLOCKED** (see Section 5) |

---

## 5. Blocked Traversals (Hub-Node Prevention)

This is the most critical section. Certain nodes act as "hubs" in Kubernetes — they connect to many unrelated resources. The BFS MUST NOT traverse through these hubs to prevent cross-resource leakage.

### 5.1 Blocked Edge Types in Resource-Focused BFS

| Edge Type | Why Blocked | What Happens |
|---|---|---|
| `namespace` (containment) | Namespace connects to ALL resources in it | Node→Namespace edge is shown but Namespace is a **terminal node** — BFS does NOT expand from it |
| `scheduling` (reverse direction) | Node connects to ALL pods scheduled on it | Pod→Node edge is shown but Node is a **terminal node** — BFS does NOT expand from it to other pods |

### 5.2 Terminal Nodes

A **terminal node** is included in the graph (it IS connected to the selected resource) but the BFS **does NOT traverse its other edges** to discover new nodes.

In resource-focused mode, these kinds are terminal nodes:
- **Namespace** — shown as context, never expanded
- **Node** — shown as scheduling info, never expanded to other pods

### 5.3 Implementation Rule

```
BEFORE expanding from a node during BFS:
  IF node.Kind == "Namespace" → SKIP (do not enqueue its neighbors)
  IF node.Kind == "Node" → SKIP (do not enqueue its neighbors)
  ELSE → expand normally
```

The terminal node ITSELF is included in the result. Only its outgoing expansion is blocked.

### 5.4 Why This Works

Without hub blocking:
```
THIS POD → Node-1 → Pod-X, Pod-Y, Pod-Z (UNRELATED — leaked in)
THIS POD → Namespace → Deploy-A, Deploy-B, Service-C (UNRELATED — leaked in)
```

With hub blocking:
```
THIS POD → Node-1 (terminal, shown but not expanded)
THIS POD → ReplicaSet → Deployment (valid chain, expanded)
THIS POD → Service → Ingress (valid chain, expanded)
```

---

## 6. Algorithm — Resource-Focused BFS

### 6.1 Pseudocode (Backend — `filter.go`)

```
function filterResource(response, focusResourceID, depth):
    // Terminal node kinds — included but never expanded
    TERMINAL_KINDS = {"Namespace", "Node"}

    // Build node lookup
    nodeMap = {node.ID: node for node in response.Nodes}

    // Build bidirectional adjacency (from edges)
    adjacency = buildAdjacency(response.Edges)

    // BFS from focus resource
    visited = {focusResourceID: 0}
    queue = [focusResourceID]

    while queue is not empty:
        current = queue.dequeue()
        currentDepth = visited[current]

        // Stop expanding if at max depth
        if currentDepth >= depth:
            continue

        // Stop expanding if current node is a terminal kind
        currentNode = nodeMap[current]
        if currentNode.Kind in TERMINAL_KINDS:
            continue

        // Expand neighbors
        for neighbor in adjacency[current]:
            if neighbor not in visited:
                visited[neighbor] = currentDepth + 1
                queue.enqueue(neighbor)

    // Build result from visited nodes
    return filterNodesAndEdges(response, visited)
```

### 6.2 Pseudocode (Frontend — `graphTraversal.ts`)

```typescript
function getResourceFocusedComponent(
  model: GraphModel,
  startNodeId: string,
  maxDepth: number
): { nodes: TopologyNode[]; edges: TopologyEdge[] } {
    const TERMINAL_KINDS = new Set(["Namespace", "Node"]);
    const visited = new Map<string, number>();  // nodeId → depth
    const queue = [{ id: startNodeId, depth: 0 }];

    while (queue.length > 0) {
        const { id, depth } = queue.shift()!;
        if (visited.has(id)) continue;
        visited.set(id, depth);

        if (depth >= maxDepth) continue;

        // Terminal nodes: include but don't expand
        const node = model.getNode(id);
        if (node && TERMINAL_KINDS.has(node.kind)) continue;

        // Expand both directions
        for (const parent of model.getParents(id)) {
            if (!visited.has(parent.id))
                queue.push({ id: parent.id, depth: depth + 1 });
        }
        for (const child of model.getChildren(id)) {
            if (!visited.has(child.id))
                queue.push({ id: child.id, depth: depth + 1 });
        }
    }

    // Filter nodes and edges
    const nodes = [...visited.keys()]
        .map(id => model.getNode(id))
        .filter(Boolean);
    const edges = model.edges.filter(
        e => visited.has(e.source) && visited.has(e.target)
    );
    return { nodes, edges };
}
```

---

## 7. Mode-to-Depth Mapping

| User Mode | Backend `depth` Parameter | Behavior |
|---|---|---|
| **Direct** | `depth=1` | Focus resource + immediate neighbors only |
| **Extended** | `depth=2` | Focus + neighbors + their neighbors |
| **Full** | `depth=0` (means unlimited, cap at 10) | Complete reachable subgraph via non-blocked edges |

### API Mapping

```
GET /api/v1/clusters/{id}/topology/v2/resource/{kind}/{ns}/{name}?depth=1  → Direct
GET /api/v1/clusters/{id}/topology/v2/resource/{kind}/{ns}/{name}?depth=2  → Extended
GET /api/v1/clusters/{id}/topology/v2/resource/{kind}/{ns}/{name}?depth=0  → Full
```

---

## 8. Resource-Specific Examples

### 8.1 Pod Topology

**Direct (depth=1):**
- ReplicaSet (owner)
- Service (if selects this pod)
- ConfigMap(s) (mounted)
- Secret(s) (mounted)
- PVC(s) (mounted)
- ServiceAccount
- Node (terminal)

**Extended (depth=2):**
- Everything from Direct, plus:
- Deployment (owner of ReplicaSet)
- Endpoints (from Service)
- Ingress (from Service)
- PV (from PVC)
- RoleBinding (from ServiceAccount)

**Full:**
- Everything reachable: Deployment → ReplicaSet → Pod → Service → Ingress → ...
- StorageClass (from PV), ClusterRole (from RoleBinding), etc.

### 8.2 Deployment Topology

**Direct (depth=1):**
- ReplicaSet(s) owned by this Deployment
- HPA (if targets this Deployment)

**Extended (depth=2):**
- Everything from Direct, plus:
- Pods (owned by the ReplicaSets)
- Service (if selects the Pods)

**Full:**
- Complete chain: Deployment → RS → Pods → Services → Ingress → ...
- Pod-level connections: ConfigMaps, Secrets, PVCs, Nodes (terminal)

### 8.3 Service Topology

**Direct (depth=1):**
- Pods (selected by this Service)
- Endpoints/EndpointSlice
- Ingress (routing to this Service)

**Extended (depth=2):**
- Everything from Direct, plus:
- ReplicaSets (owners of the Pods)
- ConfigMaps/Secrets (mounted by the Pods)

---

## 9. What This Is NOT

| This Spec | NOT This |
|---|---|
| Resource-focused topology (`ViewModeResource`) | Cluster topology (`ViewModeCluster`) |
| Filtered, scoped, noise-free | Cluster-wide graph with everything |
| Selected resource is always the center | Flat list of all resources |
| Hub nodes are terminal (Namespace, Node) | All nodes expanded equally |
| Progressive depth (1 → 2 → full) | Single depth for all modes |

The cluster/namespace/workload views have their own filtering logic and are NOT subject to this spec. They are intentionally broader.

---

## 10. Validation Checklist

Before any node is added to the resource-focused topology result, validate:

- [ ] Is this node reachable from the focus resource through valid edges? (BFS path exists)
- [ ] Was the traversal path blocked by a terminal node? (If so, the node behind it is NOT reachable)
- [ ] Is the node within the depth limit for the current mode?
- [ ] Is the edge type a valid Kubernetes relationship? (Not a synthetic/computed edge)

If ANY check fails → the node MUST NOT be included.

---

## 11. Current Code Locations

### Backend (Go)

| File | Purpose | What to Change |
|---|---|---|
| `internal/topology/v2/filter.go` | BFS traversal + filtering | Add terminal-node logic to `filterResource()` at line 103-120 |
| `internal/topology/v2/service.go` | ViewMode constants, Options | Add depth=0 handling for Full mode |
| `internal/topology/v2/edge.go` | Edge/relationship types | Reference only (no changes needed) |
| `internal/topology/v2/relationships/namespace.go` | Namespace containment edges | Reference only — edges still created, just not traversed |
| `internal/topology/v2/relationships/scheduling.go` | Pod→Node scheduling edges | Reference only — edges still created, just not traversed |
| `internal/topology/v2/handler/topology_handler.go` | HTTP handler, query parsing | May need depth=0 → unlimited mapping |

### Frontend (TypeScript)

| File | Purpose | What to Change |
|---|---|---|
| `src/topology-engine/core/graphTraversal.ts` | BFS traversal | Add terminal-node logic to `getConnectedComponent()` |
| `src/topology-engine/core/graphModel.ts` | Graph adjacency model | Reference only |
| `src/topology-engine/hooks/useResourceTopology.ts` | React Query hook | May need mode→depth mapping |

---

## 12. The Bug — Root Cause Analysis

### Current Code (`filter.go:103-120`)

```go
for len(queue) > 0 {
    current := queue[0]
    queue = queue[1:]
    currentDepth := visited[current]
    if currentDepth >= depth {
        continue
    }
    for _, neighbor := range adjacency[current] {
        if _, ok := visited[neighbor]; !ok {
            visited[neighbor] = currentDepth + 1
            queue = append(queue, neighbor)
        }
    }
}
```

### Problem

The BFS expands ALL nodes equally. When it reaches a Namespace or Node, it traverses their edges to discover all resources in that namespace/on that node — **leaking unrelated resources into the graph**.

### Concrete Leak Path

```
Pod-A (focus, depth=0)
  → Namespace/default (depth=1)    ← valid edge, should be shown
    → Pod-B (depth=2)              ← LEAKED: unrelated pod in same namespace
    → Deploy-X (depth=2)           ← LEAKED: unrelated deployment
    → Service-Y (depth=2)          ← LEAKED: unrelated service
  → Node/worker-1 (depth=1)       ← valid edge, should be shown
    → Pod-C (depth=2)              ← LEAKED: unrelated pod on same node
    → Pod-D (depth=2)              ← LEAKED: unrelated pod on same node
```

### Fix

Add two lines before the neighbor expansion loop:

```go
// Terminal nodes: include in result but do not expand
currentNode := nodeMap[current]
if currentNode.Kind == "Namespace" || currentNode.Kind == "Node" {
    continue
}
```

This ensures Namespace and Node appear in the graph (they ARE connected to the focus resource) but do NOT act as bridges to unrelated resources.

---

## 13. Summary

| Principle | Rule |
|---|---|
| Center | Selected resource is ALWAYS the center of the graph |
| Scope | Every node must be in this resource's dependency chain |
| Terminal nodes | Namespace and Node are shown but never expanded |
| Direct mode | depth=1, immediate neighbors only |
| Extended mode | depth=2, progressive expansion |
| Full mode | depth=unlimited, complete chain (still filtered) |
| Validation | Before adding any node: "Is this connected to MY resource?" |
| Difference | Resource topology != cluster topology. They are separate views. |

---

*This document is the single source of truth for resource-focused topology in Kubilitics. All implementation MUST conform to this spec.*
