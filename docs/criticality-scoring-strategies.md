# Criticality Scoring Strategies for Kubilitics

> Research document: 5 production-grade strategies for calculating Critical Scores on Kubernetes topology graphs.

---

## Strategy 1: Graph Centrality Model (Weighted PageRank)

### Core Idea

Adapt Google's PageRank to a Kubernetes dependency graph. Nodes that are depended upon by many other important nodes receive higher scores. This captures transitive importance — a ConfigMap referenced by a critical Deployment inherits criticality from that Deployment.

In SRE terms: "If many important things depend on you, you are important."

### Algorithm

```
CR(v) = (1 - d) + d * Σ [ w(u,v) * CR(u) / OutDeg(u) ]
         for all u → v

where:
  d = 0.85 (damping factor)
  w(u,v) = relationship weight * edge confidence
  OutDeg(u) = weighted out-degree of node u
```

**Relationship weights** (from existing `RELATIONSHIP_CONFIDENCE` in `blastRadiusCompute.ts`):

| Relationship | Weight |
|---|---|
| owns | 1.0 |
| selects | 0.9 |
| scheduled_on | 0.9 |
| routes | 0.85 |
| runs | 0.85 |
| exposes | 0.8 |
| backed_by | 0.75 |
| references | 0.7 |
| mounts | 0.6 |
| configures | 0.6 |
| contains, stores, permits, limits, manages | 0.5 |

**Iteration**: Run 20 iterations (converges for graphs up to 10K+ nodes). Normalize final scores to 0–100.

**Edge direction**: Reverse edges before running PageRank. In Kubernetes, edges go parent→child (Deployment→ReplicaSet→Pod). We want importance to flow _upward_ — a Pod's importance should boost its ReplicaSet and Deployment. Reversing edges makes PageRank accumulate score at high-dependency nodes.

### Data Requirements

- Graph edges with `relationshipType` and `metadata.confidence` ✅ (already available)
- Node list ✅
- No traffic data needed

### Pros & Cons

| Pros | Cons |
|---|---|
| Captures transitive importance automatically | Ownership edges dominate (Namespace→everything gets inflated) |
| Battle-tested algorithm (Google scale) | Doesn't account for health status or runtime behavior |
| O(V + E) per iteration, 20 iterations = fast | "Contains" edges from Namespace need dampening or exclusion |
| Incrementally updatable (warm-start from previous scores) | Cold start: all nodes equal until convergence |

### Real-World Usefulness

During an incident, an SRE opens the topology view sorted by PageRank score. The top-10 nodes are the services whose failure would cascade the most. If `api-gateway` Service has PageRank 95 but `debug-sidecar` Pod has PageRank 3, the SRE knows where to focus.

### Recommendation Scores

| Metric | Score |
|---|---|
| Accuracy | 7/10 — captures structural importance well, misses runtime signals |
| Performance | 9/10 — 20 iterations over 10K edges < 50ms |
| Practicality | 8/10 — works with zero external data, just the graph |

---

## Strategy 2: Blast Radius Dominance Model

### Core Idea

Score each node by the size and severity of its blast radius. A node that, when failed, takes down 40% of the cluster is more critical than one that affects 2 pods.

This directly leverages the existing `computeBlastRadius()` BFS engine in `blastRadiusCompute.ts`. Instead of computing blast radius on-demand for a selected node, we pre-compute it for every node and use the result as the criticality score.

In SRE terms: "How bad is it if this thing dies?"

### Algorithm

```
BlastScore(v) = α * AffectedRatio(v) + β * AvgSeverity(v) + γ * SPOFBonus(v)

where:
  AffectedRatio = |affected_nodes| / |total_nodes| * 100
  AvgSeverity = mean severity across affected nodes (existing propagation formula)
  SPOFBonus = 25 if node is a single point of failure, else 0
  α = 0.4, β = 0.4, γ = 0.2
```

**Optimization for scale**: Don't BFS every node. Only compute for "eligible" nodes:
- Services, Deployments, StatefulSets, DaemonSets, Ingresses, Nodes, PersistentVolumes
- Skip Pods (their parent's score covers them), ConfigMaps with 0 dependents, Namespaces
- This reduces candidates from 1000+ to ~200-300 in a typical cluster

**Incremental update**: Cache blast radius per node. On graph change, only recompute for nodes within 2 hops of the changed edge.

### Data Requirements

- Full graph with edges and relationship types ✅
- Edge confidence scores ✅
- Node kinds for type-based criticality multipliers ✅
- Existing `computeBlastRadius()` function ✅

### Pros & Cons

| Pros | Cons |
|---|---|
| Directly answers "what breaks if this fails?" | O(V * (V+E)) worst case — expensive for full recompute |
| Leverages existing blast radius code | Overestimates for nodes high in ownership hierarchy |
| SPOF detection built-in | Doesn't distinguish between "50 pods go down" and "1 critical ingress goes down" (without weighting) |
| Most intuitive score for SREs | Needs smart caching to be real-time |

### Real-World Usefulness

An SRE is doing a deployment rollout and wants to know: "If I mess up this rollout, what's the worst case?" They check the BlastScore of the Deployment. Score of 85 means "this affects most of the cluster." Score of 12 means "only a few pods, safe to proceed."

Also powers automated deployment gates: block rollouts to services with BlastScore > 70 without extra approval.

### Recommendation Scores

| Metric | Score |
|---|---|
| Accuracy | 9/10 — directly measures what SREs care about |
| Performance | 5/10 — needs optimization (eligible-node filtering + caching) |
| Practicality | 9/10 — uses existing code, immediately actionable |

---

## Strategy 3: Traffic-Weighted Betweenness Model

### Core Idea

Score nodes by how much traffic flows _through_ them. A service that sits on the critical path between Ingress and backend databases is more critical than a leaf service, even if both have the same number of edges.

Uses betweenness centrality weighted by traffic inference scores from `TrafficOverlay.ts`.

In SRE terms: "How many communication paths go through this node?"

### Algorithm

```
TrafficBetweenness(v) = Σ [ σ_st(v) / σ_st * TrafficWeight(s,t) ]
                        for all pairs (s,t) where s ≠ v ≠ t

where:
  σ_st = total shortest paths from s to t
  σ_st(v) = shortest paths from s to t that pass through v
  TrafficWeight(s,t) = edge traffic score (from TrafficOverlay heuristics)
```

**Practical simplification** (for scale): Don't compute all-pairs. Instead:
1. Identify "entry points" (Ingress, Services with `type: LoadBalancer`)
2. Identify "backends" (StatefulSets, PersistentVolumes, external services)
3. Compute betweenness only on paths from entry points → backends
4. This reduces from O(V³) to O(E * P) where P = entry×backend pairs (~50-100)

**Traffic weighting from existing heuristics**:
- `routes` edges: weight 0.8
- `exposes` edges: weight 0.7
- `selects` edges: weight 0.6
- `owns` edges: weight 0.4 (structural, not traffic)

Normalize final scores to 0–100.

### Data Requirements

- Graph edges ✅
- Relationship types (for traffic weight inference) ✅
- Node kinds (to identify entry/backend nodes) ✅
- Optional: actual traffic metrics from Prometheus (enhances accuracy but not required)

### Pros & Cons

| Pros | Cons |
|---|---|
| Finds "chokepoint" services that aren't obvious from edge count | All-pairs betweenness is O(V³) — must use sampling |
| Traffic weighting makes it runtime-aware | Without real traffic data, heuristic weights are approximate |
| Identifies services on critical communication paths | Doesn't capture nodes that are important but not on paths (e.g., standalone CronJobs) |
| Works well with microservices architectures | Less useful for batch/offline workloads |

### Real-World Usefulness

Netflix scenario: You have 200 microservices. Service `auth-proxy` sits between every Ingress and every backend. Its betweenness score is 95. If it goes down, _all_ user-facing traffic stops. Meanwhile `recommendation-engine` has betweenness 15 — it's a leaf service. During an incident, the SRE immediately knows `auth-proxy` is the critical path.

### Recommendation Scores

| Metric | Score |
|---|---|
| Accuracy | 8/10 — excellent for traffic-path criticality |
| Performance | 6/10 — needs entry/backend sampling optimization |
| Practicality | 7/10 — great for service mesh clusters, decent with heuristics |

---

## Strategy 4: Structural Risk Score (Resource-Aware Composite)

### Core Idea

Not all Kubernetes resources are created equal. A StatefulSet with 1 replica running a database is categorically more critical than a DaemonSet running a log collector, regardless of graph position. This strategy combines resource-type risk, operational health signals, and redundancy analysis into a single score.

In SRE terms: "How risky is this resource based on what it IS and how it's configured?"

### Algorithm

```
RiskScore(v) = TypeWeight(v) * RedundancyPenalty(v) * HealthMultiplier(v) * ConnectivityFactor(v)

Normalized to 0-100.
```

**TypeWeight** (base criticality by resource kind):

| Kind | Weight | Rationale |
|---|---|---|
| Ingress | 30 | Entry point for all external traffic |
| Service (LoadBalancer/NodePort) | 28 | User-facing endpoint |
| Service (ClusterIP) | 20 | Internal service |
| StatefulSet | 25 | Stateful, hard to recover |
| Deployment | 18 | Standard workload |
| DaemonSet | 15 | Cluster-wide but usually non-critical path |
| Node | 35 | Infrastructure — affects all scheduled pods |
| PersistentVolume | 22 | Data loss risk |
| Secret | 12 | Auth dependency |
| ConfigMap | 8 | Configuration |
| CronJob | 10 | Batch workload |
| Pod | 5 | Ephemeral, usually replaceable |

**RedundancyPenalty** (lower replicas = higher risk):
```
if replicas == 1:  penalty = 2.0  (single instance!)
if replicas == 2:  penalty = 1.5
if replicas >= 3:  penalty = 1.0  (adequately redundant)
For non-replica resources: penalty = 1.0
```

**HealthMultiplier** (unhealthy = more critical to address):
```
healthy:  1.0
warning:  1.3
critical: 1.8
unknown:  1.2
```

**ConnectivityFactor** (normalized in-degree + out-degree):
```
ConnectivityFactor = 1.0 + Min(1.0, (inDegree + outDegree) / 20)
Range: 1.0 to 2.0
```

### Data Requirements

- Node kind, status, health ✅
- Replica count (from `computed.replicas`) ✅
- In/out degree (from graph edges) ✅
- No traffic data needed

### Pros & Cons

| Pros | Cons |
|---|---|
| Captures domain knowledge (StatefulSet > Pod) | Weights are hand-tuned, may not fit all clusters |
| Penalizes risky configurations (1 replica) | Doesn't capture transitive dependencies |
| Uses health status — unhealthy critical services rank highest | A well-configured leaf service may score higher than a poorly-positioned chokepoint |
| Extremely fast — O(V) with pre-computed degrees | Redundancy data may be incomplete for some resource types |
| No graph traversal needed | |

### Real-World Usefulness

AWS CloudWatch-style: "Your cluster has 3 high-risk resources: `postgres` StatefulSet (1 replica, score 92), `api-gateway` Ingress (critical health, score 88), `redis` StatefulSet (1 replica, warning, score 78)." The SRE immediately knows what to fix before it becomes an incident.

Powers a "Cluster Health Report" dashboard widget.

### Recommendation Scores

| Metric | Score |
|---|---|
| Accuracy | 6/10 — good for configuration risk, misses graph dynamics |
| Performance | 10/10 — O(V), no traversal |
| Practicality | 9/10 — immediately useful, easy to explain |

---

## Strategy 5: Unified Criticality Index (Hybrid Ensemble)

### Core Idea

No single strategy captures all dimensions of criticality. This strategy combines the four previous strategies into one unified score using weighted ensemble averaging, similar to how Google combines multiple ranking signals.

In SRE terms: "The definitive answer to 'how critical is this service?'"

### Algorithm

```
UnifiedScore(v) = w1 * PageRank(v)_norm +
                  w2 * BlastRadius(v)_norm +
                  w3 * Betweenness(v)_norm +
                  w4 * RiskScore(v)_norm

Default weights:
  w1 = 0.25  (graph centrality — structural importance)
  w2 = 0.30  (blast radius — failure impact, highest weight)
  w3 = 0.20  (traffic betweenness — communication criticality)
  w4 = 0.25  (structural risk — configuration awareness)

All sub-scores normalized to 0-100 before combining.
```

**Adaptive weighting** (optional, for advanced clusters):
- If traffic data is available (service mesh): increase w3 to 0.30, decrease w4 to 0.15
- If cluster is mostly stateless: decrease w4 to 0.15, increase w1 to 0.35
- If cluster has many single-replica workloads: increase w4 to 0.35, decrease w3 to 0.10

**Computation strategy** (for performance):

```
Phase 1 (parallel):
  - PageRank: 20 iterations over reversed graph          ~30ms
  - RiskScore: single pass over nodes                    ~5ms

Phase 2 (parallel, with eligible-node optimization):
  - BlastRadius: BFS per eligible node (cached)          ~100ms
  - Betweenness: entry→backend path sampling             ~80ms

Phase 3:
  - Normalize all scores to 0-100                        ~2ms
  - Weighted combination                                 ~2ms

Total: ~120ms for 1000 nodes / 10000 edges (parallel phases)
```

**Incremental update**: On graph change:
1. RiskScore: recompute only changed nodes — O(1)
2. PageRank: warm-start from cached scores, run 5 iterations — O(V+E)
3. BlastRadius: recompute for nodes within 2 hops of change — O(k*(V+E))
4. Betweenness: recompute only if entry/backend set changed — usually skip

### Data Requirements

- Everything from strategies 1-4 ✅
- All data already available in the Kubilitics graph model

### Pros & Cons

| Pros | Cons |
|---|---|
| Most comprehensive — covers structure, impact, traffic, configuration | Most complex to implement and tune |
| Robust: no single blind spot | Weights need tuning per cluster type |
| Degradable: if one signal is unavailable, others compensate | Harder to explain "why is this node scored 87?" |
| Matches how Google/Netflix rank service criticality internally | Computation cost is sum of all sub-strategies |

### Real-World Usefulness

The SRE dashboard shows a single "Criticality" column. `payment-service` scores 94 because it has high PageRank (many dependents), large blast radius (takes down checkout flow), high betweenness (on critical Ingress→DB path), and high risk score (StatefulSet, 2 replicas). The SRE doesn't need to think about which dimension matters — the unified score captures all of them.

Powers:
- Incident priority routing (PagerDuty integration)
- Change management risk assessment
- Capacity planning prioritization
- SLO/SLA tier assignment

### Recommendation Scores

| Metric | Score |
|---|---|
| Accuracy | 9/10 — best overall accuracy from ensemble |
| Performance | 7/10 — ~120ms with parallelism and caching |
| Practicality | 8/10 — most value, moderate implementation effort |

---

## Final Recommendation

### Phased Adoption

| Phase | Strategy | Why |
|---|---|---|
| **Phase 1 (Ship now)** | Strategy 4: Structural Risk Score | O(V), zero graph traversal, uses existing node data. Ship in 1 day. Gives immediate value. |
| **Phase 2 (Next sprint)** | Strategy 1: PageRank + Strategy 2: Blast Radius | PageRank adds transitive importance. Blast radius leverages existing `computeBlastRadius()`. Together they cover structure + impact. |
| **Phase 3 (With traffic data)** | Strategy 3: Traffic Betweenness | Add when real traffic metrics are available (Prometheus/service mesh). Until then, heuristic weights from TrafficOverlay work as approximation. |
| **Phase 4 (Production)** | Strategy 5: Unified Index | Combine all signals. Tune weights based on real cluster feedback. This becomes the single "Criticality Score" shown in the UI. |

### The One Score to Ship First

**Strategy 4 (Structural Risk Score)** — it's fast, needs no graph traversal, uses data already in every `TopologyNode`, and is immediately understandable by SREs. It answers "what's misconfigured and risky?" which is the most actionable question.

Then layer on PageRank + Blast Radius to evolve toward the Unified Index.

### Integration Points in Kubilitics

1. **Backend**: Add `CriticalityScore` field to `TopologyNode` struct in `node.go`
2. **Frontend**: New `CriticalityOverlay` in `topology-engine/overlays/` (follows existing overlay pattern)
3. **Scoring engine**: New `topology-engine/scoring/` module with pluggable strategies
4. **UI**: Sort/filter by criticality in topology view, color nodes by score tier
5. **API**: Expose `/api/v2/topology/criticality` endpoint for external integrations
