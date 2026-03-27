package v2

import "strings"

// ViewFilter applies view-mode specific filtering to a fully built TopologyResponse.
type ViewFilter struct{}

// Filter returns a new TopologyResponse with only the nodes and edges relevant to the given Options.
func (f *ViewFilter) Filter(resp *TopologyResponse, opts Options) *TopologyResponse {
	if resp == nil {
		return nil
	}
	switch opts.Mode {
	case ViewModeCluster:
		return f.filterCluster(resp)
	case ViewModeNamespace:
		return f.filterNamespace(resp, opts.Namespace)
	case ViewModeWorkload:
		return f.filterWorkload(resp, opts.Namespace)
	case ViewModeResource:
		return f.filterResource(resp, opts.Resource, opts.Namespace, opts.Depth)
	case ViewModeRBAC:
		return f.filterRBAC(resp, opts.Namespace)
	default:
		return resp
	}
}

func (f *ViewFilter) filterCluster(resp *TopologyResponse) *TopologyResponse {
	allowed := map[string]bool{
		"Namespace": true, "Node": true, "Deployment": true, "StatefulSet": true, "DaemonSet": true,
	}
	return filterByKinds(resp, allowed)
}

func (f *ViewFilter) filterNamespace(resp *TopologyResponse, ns string) *TopologyResponse {
	if ns == "" {
		return resp
	}
	nodeIDs := make(map[string]bool)
	var nodes []TopologyNode
	for _, n := range resp.Nodes {
		if n.Namespace == ns || n.Namespace == "" {
			nodes = append(nodes, n)
			nodeIDs[n.ID] = true
		}
	}
	edges := filterEdgesByNodes(resp.Edges, nodeIDs)
	groups := filterGroupsByNodes(resp.Groups, nodeIDs)
	return &TopologyResponse{Metadata: resp.Metadata, Nodes: nodes, Edges: edges, Groups: groups}
}

func (f *ViewFilter) filterWorkload(resp *TopologyResponse, ns string) *TopologyResponse {
	workloadKinds := map[string]bool{
		"Deployment": true, "StatefulSet": true, "DaemonSet": true,
		"ReplicaSet": true, "Pod": true, "Job": true, "CronJob": true,
		"Service": true, "HorizontalPodAutoscaler": true, "PodDisruptionBudget": true,
	}
	nodeIDs := make(map[string]bool)
	var nodes []TopologyNode
	for _, n := range resp.Nodes {
		if workloadKinds[n.Kind] && (ns == "" || n.Namespace == ns || n.Namespace == "") {
			nodes = append(nodes, n)
			nodeIDs[n.ID] = true
		}
	}
	connectedIDs := findConnectedNodes(resp.Edges, nodeIDs)
	for _, n := range resp.Nodes {
		if connectedIDs[n.ID] && !nodeIDs[n.ID] {
			nodes = append(nodes, n)
			nodeIDs[n.ID] = true
		}
	}
	edges := filterEdgesByNodes(resp.Edges, nodeIDs)
	groups := filterGroupsByNodes(resp.Groups, nodeIDs)
	return &TopologyResponse{Metadata: resp.Metadata, Nodes: nodes, Edges: edges, Groups: groups}
}

// staticHubKinds are resource kinds that act as structural hubs. They are included
// as leaf nodes when connected but NEVER used as expansion sources during BFS.
// This prevents namespace/node explosion (Bugs 1, 2 from the topology spec).
var staticHubKinds = map[string]bool{
	"Namespace":     true,
	"Node":          true,
	"LimitRange":    true,
	"ResourceQuota": true,
	"PriorityClass": true,
	"RuntimeClass":  true,
	"IngressClass":  true,
	"StorageClass":  true,
}

// leafOnlyCategories are edge relationship categories that connect to leaf-only
// nodes. Traversal adds the neighbor but does NOT queue it for further expansion.
// Per spec: containment (Namespace) and scheduling (Node) are leaf-only edges.
var leafOnlyCategories = map[string]bool{
	"containment": true,
	"scheduling":  true,
}

// dynamicHubThreshold — any non-workload node with more than this many connections
// in the full graph is treated as a dynamic hub (included but not expanded).
const dynamicHubThreshold = 5

// filterResource builds a resource-centric view using edge-type-aware, hub-aware
// BFS traversal. This is the core USP topology per the final specification.
//
// Modes (mapped from depth parameter):
//
//	depth=1 (Direct):   immediate neighbors only (1 hop)
//	depth=2 (Extended): expand one level from direct nodes (2 hops)
//	depth=3 (Full):     complete dependency graph (unlimited hops)
//
// Invariants enforced:
//   - Selected resource is ALWAYS the center
//   - Every node shown MUST belong to its dependency chain
//   - Static/dynamic hubs are included as leaves, never expanded (except focus)
//   - Containment/scheduling edges produce leaf nodes, never expansion sources
//   - No namespace explosion, no node explosion, no cross-resource leakage
func (f *ViewFilter) filterResource(resp *TopologyResponse, resource, ns string, depth int) *TopologyResponse {
	if resource == "" {
		return resp
	}
	if depth <= 0 {
		depth = 2
	}

	// --- Find focus node ---
	var focusID string
	for _, n := range resp.Nodes {
		if n.ID == resource {
			focusID = n.ID
			break
		}
		if matchesResourceQuery(n, resource, ns) {
			focusID = n.ID
			break
		}
	}
	if focusID == "" {
		return &TopologyResponse{Metadata: resp.Metadata}
	}

	// --- Build indexes ---
	nodeKind := make(map[string]string, len(resp.Nodes))
	for _, n := range resp.Nodes {
		nodeKind[n.ID] = n.Kind
	}

	typedAdj := buildTypedAdjacency(resp.Edges)

	// Compute dynamic hubs: non-workload nodes with fan-out > threshold.
	dynamicHubs := computeDynamicHubs(typedAdj, nodeKind)

	// isHub returns true if a node is a static or dynamic hub.
	isHub := func(nodeID string) bool {
		if staticHubKinds[nodeKind[nodeID]] {
			return true
		}
		return dynamicHubs[nodeID]
	}

	// --- Effective depth: Full mode = unlimited (capped at 100 for safety) ---
	maxDepth := depth
	if depth >= 3 {
		maxDepth = 100
	}

	// --- Edge-type-aware, hub-aware BFS ---
	type bfsEntry struct {
		id    string
		depth int
	}
	visited := make(map[string]int) // nodeID → depth at which it was visited
	visited[focusID] = 0
	queue := []bfsEntry{{id: focusID, depth: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.depth >= maxDepth {
			continue
		}

		// Hub nodes are leaf-only: do NOT expand from them (unless they are the focus).
		if current.id != focusID && isHub(current.id) {
			continue
		}

		for _, neighbor := range typedAdj[current.id] {
			if _, seen := visited[neighbor.nodeID]; seen {
				continue
			}

			// Add the neighbor at the next depth level.
			neighborDepth := current.depth + 1
			visited[neighbor.nodeID] = neighborDepth

			// Leaf-only edge categories: include the neighbor but do NOT queue for expansion.
			// Also, if the neighbor is a hub, include it but don't expand.
			if leafOnlyCategories[neighbor.category] || isHub(neighbor.nodeID) {
				continue // included in visited, but not queued
			}

			queue = append(queue, bfsEntry{id: neighbor.nodeID, depth: neighborDepth})
		}
	}

	// --- Build filtered response ---
	nodeIDs := make(map[string]bool, len(visited))
	for id := range visited {
		nodeIDs[id] = true
	}
	var nodes []TopologyNode
	for _, n := range resp.Nodes {
		if nodeIDs[n.ID] {
			nodes = append(nodes, n)
		}
	}
	edges := filterEdgesByNodes(resp.Edges, nodeIDs)
	groups := filterGroupsByNodes(resp.Groups, nodeIDs)
	meta := resp.Metadata
	meta.FocusResource = focusID
	return &TopologyResponse{Metadata: meta, Nodes: nodes, Edges: edges, Groups: groups}
}

// adjacencyEntry tracks a neighbor and the category of the edge connecting to it.
type adjacencyEntry struct {
	nodeID   string
	category string
}

// buildTypedAdjacency builds a bidirectional adjacency map that preserves edge categories.
func buildTypedAdjacency(edges []TopologyEdge) map[string][]adjacencyEntry {
	adj := make(map[string][]adjacencyEntry, len(edges)*2)
	for _, e := range edges {
		adj[e.Source] = append(adj[e.Source], adjacencyEntry{nodeID: e.Target, category: e.RelationshipCategory})
		adj[e.Target] = append(adj[e.Target], adjacencyEntry{nodeID: e.Source, category: e.RelationshipCategory})
	}
	return adj
}

// computeDynamicHubs identifies non-workload, non-static-hub nodes with fan-out
// exceeding dynamicHubThreshold. These are treated as hubs at runtime.
func computeDynamicHubs(adj map[string][]adjacencyEntry, nodeKind map[string]string) map[string]bool {
	// Workload kinds are never dynamic hubs — they are core dependency chain members.
	neverDynamicHub := map[string]bool{
		"Deployment": true, "StatefulSet": true, "DaemonSet": true,
		"ReplicaSet": true, "CronJob": true, "Job": true,
		"Pod": true, "Service": true, "Endpoints": true,
		"EndpointSlice": true, "PersistentVolumeClaim": true,
		"PersistentVolume": true, "HorizontalPodAutoscaler": true,
		"PodDisruptionBudget": true, "Role": true, "ClusterRole": true,
		"RoleBinding": true, "ClusterRoleBinding": true,
		"Ingress": true, "NetworkPolicy": true,
	}

	hubs := make(map[string]bool)
	for nodeID, neighbors := range adj {
		kind := nodeKind[nodeID]
		if staticHubKinds[kind] || neverDynamicHub[kind] {
			continue
		}
		if len(neighbors) > dynamicHubThreshold {
			hubs[nodeID] = true
		}
	}
	return hubs
}

func (f *ViewFilter) filterRBAC(resp *TopologyResponse, ns string) *TopologyResponse {
	rbacKinds := map[string]bool{
		"ServiceAccount": true, "Role": true, "RoleBinding": true,
		"ClusterRole": true, "ClusterRoleBinding": true,
	}
	nodeIDs := make(map[string]bool)
	var nodes []TopologyNode
	for _, n := range resp.Nodes {
		if rbacKinds[n.Kind] && (ns == "" || n.Namespace == ns || n.Namespace == "") {
			nodes = append(nodes, n)
			nodeIDs[n.ID] = true
		}
	}
	edges := filterEdgesByNodes(resp.Edges, nodeIDs)
	groups := filterGroupsByNodes(resp.Groups, nodeIDs)
	return &TopologyResponse{Metadata: resp.Metadata, Nodes: nodes, Edges: edges, Groups: groups}
}

func filterByKinds(resp *TopologyResponse, allowed map[string]bool) *TopologyResponse {
	nodeIDs := make(map[string]bool)
	var nodes []TopologyNode
	for _, n := range resp.Nodes {
		if allowed[n.Kind] {
			nodes = append(nodes, n)
			nodeIDs[n.ID] = true
		}
	}
	edges := filterEdgesByNodes(resp.Edges, nodeIDs)
	groups := filterGroupsByNodes(resp.Groups, nodeIDs)
	return &TopologyResponse{Metadata: resp.Metadata, Nodes: nodes, Edges: edges, Groups: groups}
}

func filterEdgesByNodes(edges []TopologyEdge, nodeIDs map[string]bool) []TopologyEdge {
	var out []TopologyEdge
	for _, e := range edges {
		if nodeIDs[e.Source] && nodeIDs[e.Target] {
			out = append(out, e)
		}
	}
	return out
}

func filterGroupsByNodes(groups []TopologyGroup, nodeIDs map[string]bool) []TopologyGroup {
	var out []TopologyGroup
	for _, g := range groups {
		var members []string
		for _, m := range g.Members {
			if nodeIDs[m] {
				members = append(members, m)
			}
		}
		if len(members) > 0 {
			g2 := g
			g2.Members = members
			out = append(out, g2)
		}
	}
	return out
}

func findConnectedNodes(edges []TopologyEdge, seeds map[string]bool) map[string]bool {
	connected := make(map[string]bool)
	for _, e := range edges {
		if seeds[e.Source] {
			connected[e.Target] = true
		}
		if seeds[e.Target] {
			connected[e.Source] = true
		}
	}
	return connected
}

func buildAdjacency(edges []TopologyEdge) map[string][]string {
	adj := make(map[string][]string)
	for _, e := range edges {
		adj[e.Source] = append(adj[e.Source], e.Target)
		adj[e.Target] = append(adj[e.Target], e.Source)
	}
	return adj
}

func matchesResourceQuery(n TopologyNode, resource, ns string) bool {
	parts := strings.Split(resource, "/")
	switch len(parts) {
	case 3:
		return strings.EqualFold(n.Kind, parts[0]) && n.Namespace == parts[1] && n.Name == parts[2]
	case 2:
		return strings.EqualFold(n.Kind, parts[0]) && n.Name == parts[1] && (ns == "" || n.Namespace == ns)
	case 1:
		return n.Name == parts[0] && (ns == "" || n.Namespace == ns)
	}
	return false
}

