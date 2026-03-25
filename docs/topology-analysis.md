# Kubilitics Topology — Critical Analysis

**Date**: 2026-03-25
**Status**: Audit complete, gaps identified

## What We Have

### Architecture
- **Frontend**: React Flow (@xyflow/react) with semantic zoom (4 node types based on zoom level)
- **Backend**: Go topology v2 API with pluggable matcher registry pattern
- **Layout**: ELK (Eclipse Layout Kernel) with smart hybrid strategy — layered for <300 nodes, category grid for larger graphs
- **Caching**: In-memory with 30s TTL

### Resource Coverage (34 types)
Pods, Deployments, StatefulSets, DaemonSets, ReplicaSets, Jobs, CronJobs, Services, Endpoints, EndpointSlices, Ingresses, IngressClasses, ConfigMaps, Secrets, PVCs, PVs, StorageClasses, Nodes, Namespaces, ServiceAccounts, Roles, RoleBindings, ClusterRoles, ClusterRoleBindings, HPAs, PDBs, NetworkPolicies, MutatingWebhooks, ValidatingWebhooks

### Relationship Matchers (12)
| Matcher | Relationships | Method |
|---------|--------------|--------|
| OwnerRef | Deployment→RS→Pod, Job→Pod | ownerReferences |
| Selector | Service→Pod, PDB→Pod, NP→Pod | spec.selector |
| VolumeMount | Pod→PVC, Pod→ConfigMap, Pod→Secret | spec.volumes |
| EnvRef | Pod→ConfigMap, Pod→Secret | env/envFrom |
| Ingress | Ingress→Service, Ingress→IngressClass, Ingress→Secret(TLS) | spec.rules |
| Endpoint | Endpoints/EndpointSlices→Pods | addresses |
| RBAC | RoleBinding→Role, SA→RoleBinding | subjects/roleRef |
| Scheduling | Pod→Node | spec.nodeName |
| Scaling | HPA→Deployment/StatefulSet | scaleTargetRef |
| Storage | PVC→PV, PVC→StorageClass, PV→StorageClass | spec bindings |
| Webhook | Webhook→Service | clientConfig.service |
| Namespace | Resource→Namespace | metadata.namespace |

### Interactions
- Zoom (semantic — node types change at thresholds)
- Pan, drag nodes
- Click → detail panel (metadata, labels, metrics, cost, connected nodes)
- Keyboard shortcuts (+/-, F fit, P presentation, S search, arrows)
- Namespace/kind filtering
- Full-text search with highlighting
- Collapse/expand groups
- Export: PNG, SVG, PDF, JSON, draw.io

### View Modes
- Namespace view (grouped by namespace)
- Cluster view (all resources flat)
- RBAC view (ServiceAccount→RoleBinding→Role chains)
- Resource view (single-resource centric, used in detail page tabs)

## What's Missing — Critical Gaps

### P0 — Scale & Real-Time

1. **No scale limits or aggregation** — All pods rendered individually. A namespace with 200 pods creates 200 nodes. No "50 pods" summary node. Large clusters (500+ pods) will choke both layout and rendering.

2. **WebSocket handler incomplete** — Frontend subscribes to `/ws/topology/{clusterId}/v2` with auto-reconnect. Backend route exists but **no server-side push implementation found**. The graph doesn't update in real-time when pods scale, crash, or deploy.

3. **Selector matching is O(n²)** — For each Service, loops all Pods checking labels. 100 services × 500 pods = 50,000 comparisons. No label indexing.

### P1 — Missing Relationships

4. **Pod→ServiceAccount** — Pod uses `serviceAccountName` but no edge is drawn. Critical for RBAC view completeness.

5. **Network Policy rules** — Only shows NP→Pod selector edges. Missing: actual ingress/egress rules, allowed/blocked pod-to-pod connections, network flow direction.

6. **Cross-namespace** — Service selectors are namespace-scoped only. ExternalName services, cross-namespace Ingress→Service not visualized.

7. **CRD relationships** — Custom resources not included in the graph at all.

### P2 — Usability

8. **No pod grouping** — 10 identical nginx pods show as 10 separate nodes instead of a single "nginx (10 replicas)" summary. ReplicaSet pods, StatefulSet ordinals not grouped.

9. **No kind filtering in toolbar** — Can filter by namespace but can't hide all ConfigMaps or all Secrets to reduce noise.

10. **No lasso multi-select** — Can't select multiple nodes for bulk operations.

11. **No edge hiding** — All relationship types shown simultaneously. Can't hide volume mounts to focus on networking.

12. **No dependency/impact analysis** — Can't answer "what happens if I delete this ConfigMap?" or "show me the critical path from Ingress to Pod."

### P3 — Quality

13. **No distributed cache** — In-memory only, no cache sync for multi-instance backends.

14. **No server-side rendering** — PNG/SVG exports are viewport-dependent (client-side canvas capture). No consistent export at arbitrary resolutions.

15. **Legacy D3 code** — `D3TopologyCanvas.tsx` (1,404 lines) is dead code from a previous implementation. Should be deleted.

16. **No performance benchmarks** — Layout algorithm strategy is smart (hybrid) but untested at scale. No documented limits.

## Strengths

- Pluggable matcher architecture — adding a new relationship type is trivial (implement interface, register)
- Semantic zoom is elegant — nodes simplify at low zoom, detail at high zoom
- 34 resource types with 12 matcher types is comprehensive for v1
- ELK layout produces clean hierarchical graphs
- Keyboard shortcuts for power users
- Multiple export formats including draw.io
- Caching prevents redundant API calls
- Multi-view modes serve different use cases

## Recommendations

### Immediate (v0.5.0)
- Add Pod→ServiceAccount edges (1 new matcher)
- Add pod aggregation/summary nodes for ReplicaSets with >3 pods
- Delete legacy D3TopologyCanvas.tsx (1,404 lines dead code)
- Document scale limits (recommend max 500 nodes without aggregation)

### Next Release (v0.6.0)
- Implement server-side WebSocket push for real-time graph updates
- Add kind filter to toolbar (show/hide resource types)
- Add edge type filter (show/hide relationship categories)
- Add label indexing to selector matcher (eliminate O(n²))
- Implement NetworkPolicy rule visualization (ingress/egress details)

### Future
- Pod grouping within controllers (summary nodes)
- Dependency impact analysis ("what breaks if I delete X?")
- Lasso multi-select
- Distributed cache (Redis) for HA
- CRD relationship discovery
- Performance benchmarks with 1000+ node graphs
