package v2

import "testing"

func newTestResponse() *TopologyResponse {
	return &TopologyResponse{
		Metadata: TopologyMetadata{ClusterID: "test", Mode: ViewModeNamespace},
		Nodes: []TopologyNode{
			{ID: "Namespace/default", Kind: "Namespace", Name: "default"},
			{ID: "Namespace/kube-system", Kind: "Namespace", Name: "kube-system"},
			{ID: "Node/worker-1", Kind: "Node", Name: "worker-1"},
			{ID: "Deployment/default/app-a", Kind: "Deployment", Name: "app-a", Namespace: "default"},
			{ID: "Pod/default/app-a-pod-0", Kind: "Pod", Name: "app-a-pod-0", Namespace: "default"},
			{ID: "Service/default/svc-a", Kind: "Service", Name: "svc-a", Namespace: "default"},
			{ID: "ConfigMap/default/cm-a", Kind: "ConfigMap", Name: "cm-a", Namespace: "default"},
			{ID: "ServiceAccount/default/sa-a", Kind: "ServiceAccount", Name: "sa-a", Namespace: "default"},
			{ID: "Role/default/role-a", Kind: "Role", Name: "role-a", Namespace: "default"},
			{ID: "RoleBinding/default/rb-a", Kind: "RoleBinding", Name: "rb-a", Namespace: "default"},
			{ID: "Deployment/kube-system/coredns", Kind: "Deployment", Name: "coredns", Namespace: "kube-system"},
		},
		Edges: []TopologyEdge{
			{ID: "e1", Source: "Deployment/default/app-a", Target: "Pod/default/app-a-pod-0", RelationshipType: "ownerRef", RelationshipCategory: "ownership"},
			{ID: "e2", Source: "Service/default/svc-a", Target: "Pod/default/app-a-pod-0", RelationshipType: "selector", RelationshipCategory: "networking"},
			{ID: "e3", Source: "Pod/default/app-a-pod-0", Target: "ConfigMap/default/cm-a", RelationshipType: "volume_mount", RelationshipCategory: "configuration"},
			{ID: "e4", Source: "Pod/default/app-a-pod-0", Target: "Node/worker-1", RelationshipType: "scheduling", RelationshipCategory: "scheduling"},
			{ID: "e5", Source: "RoleBinding/default/rb-a", Target: "Role/default/role-a", RelationshipType: "role_binding", RelationshipCategory: "rbac"},
			{ID: "e6", Source: "ServiceAccount/default/sa-a", Target: "RoleBinding/default/rb-a", RelationshipType: "role_binding", RelationshipCategory: "rbac"},
		},
		Groups: []TopologyGroup{
			{ID: "group-ns-default", Label: "default", Type: "namespace", Members: []string{"Deployment/default/app-a", "Pod/default/app-a-pod-0"}},
		},
	}
}

func TestViewFilter_Cluster(t *testing.T) {
	resp := newTestResponse()
	filter := &ViewFilter{}
	result := filter.Filter(resp, Options{Mode: ViewModeCluster})
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Cluster mode shows: Namespace, Node, Deployment, StatefulSet, DaemonSet
	for _, n := range result.Nodes {
		switch n.Kind {
		case "Namespace", "Node", "Deployment", "StatefulSet", "DaemonSet":
			// OK
		default:
			t.Errorf("unexpected kind in cluster view: %s", n.Kind)
		}
	}
}

func TestViewFilter_Namespace(t *testing.T) {
	resp := newTestResponse()
	filter := &ViewFilter{}
	result := filter.Filter(resp, Options{Mode: ViewModeNamespace, Namespace: "default"})
	for _, n := range result.Nodes {
		if n.Namespace != "default" && n.Namespace != "" {
			t.Errorf("expected namespace default or cluster-scoped, got %s for %s", n.Namespace, n.ID)
		}
	}
}

func TestViewFilter_Resource(t *testing.T) {
	resp := newTestResponse()
	filter := &ViewFilter{}
	result := filter.Filter(resp, Options{
		Mode:     ViewModeResource,
		Resource: "Pod/default/app-a-pod-0",
		Depth:    1,
	})
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Should include the pod and its direct connections
	nodeIDs := make(map[string]bool)
	for _, n := range result.Nodes {
		nodeIDs[n.ID] = true
	}
	if !nodeIDs["Pod/default/app-a-pod-0"] {
		t.Error("missing focus node")
	}
	// At depth 1, should include direct neighbors
	if !nodeIDs["Deployment/default/app-a"] {
		t.Error("missing deployment connected to pod")
	}
	if !nodeIDs["Service/default/svc-a"] {
		t.Error("missing service connected to pod")
	}
}

func TestViewFilter_Resource_Depth2(t *testing.T) {
	resp := newTestResponse()
	filter := &ViewFilter{}
	result := filter.Filter(resp, Options{
		Mode:     ViewModeResource,
		Resource: "Pod/default/app-a-pod-0",
		Depth:    2,
	})
	nodeIDs := make(map[string]bool)
	for _, n := range result.Nodes {
		nodeIDs[n.ID] = true
	}
	// Focus node always present
	if !nodeIDs["Pod/default/app-a-pod-0"] {
		t.Error("missing focus node")
	}
	// ConfigMap at depth 1 via configuration edge — included and expandable
	if !nodeIDs["ConfigMap/default/cm-a"] {
		t.Error("missing configmap at depth 1")
	}
	// Node at depth 1 via scheduling edge — included as LEAF (not expanded)
	if !nodeIDs["Node/worker-1"] {
		t.Error("missing node at depth 1 (leaf via scheduling edge)")
	}
}

// TestViewFilter_Resource_NoNamespaceExplosion verifies that selecting a pod does
// NOT pull in its namespace's other resources (Bug 1 from topology spec).
func TestViewFilter_Resource_NoNamespaceExplosion(t *testing.T) {
	resp := &TopologyResponse{
		Metadata: TopologyMetadata{ClusterID: "test"},
		Nodes: []TopologyNode{
			{ID: "Namespace/default", Kind: "Namespace", Name: "default"},
			{ID: "Pod/default/my-pod", Kind: "Pod", Name: "my-pod", Namespace: "default"},
			{ID: "Pod/default/other-pod", Kind: "Pod", Name: "other-pod", Namespace: "default"},
			{ID: "Service/default/other-svc", Kind: "Service", Name: "other-svc", Namespace: "default"},
		},
		Edges: []TopologyEdge{
			{ID: "e1", Source: "Namespace/default", Target: "Pod/default/my-pod", RelationshipCategory: "containment"},
			{ID: "e2", Source: "Namespace/default", Target: "Pod/default/other-pod", RelationshipCategory: "containment"},
			{ID: "e3", Source: "Namespace/default", Target: "Service/default/other-svc", RelationshipCategory: "containment"},
		},
	}
	filter := &ViewFilter{}
	result := filter.Filter(resp, Options{Mode: ViewModeResource, Resource: "Pod/default/my-pod", Depth: 2})
	nodeIDs := make(map[string]bool)
	for _, n := range result.Nodes {
		nodeIDs[n.ID] = true
	}
	if !nodeIDs["Pod/default/my-pod"] {
		t.Error("missing focus pod")
	}
	// Namespace is a leaf — included via containment edge but not expanded
	if !nodeIDs["Namespace/default"] {
		t.Error("missing namespace as leaf node")
	}
	// MUST NOT include other pods/services via namespace explosion
	if nodeIDs["Pod/default/other-pod"] {
		t.Error("BUG: namespace explosion — other-pod should NOT be included")
	}
	if nodeIDs["Service/default/other-svc"] {
		t.Error("BUG: namespace explosion — other-svc should NOT be included")
	}
}

// TestViewFilter_Resource_NoNodeExplosion verifies that a pod's Node connection
// does NOT pull in all workloads scheduled on that node (Bug 2 from topology spec).
func TestViewFilter_Resource_NoNodeExplosion(t *testing.T) {
	resp := &TopologyResponse{
		Metadata: TopologyMetadata{ClusterID: "test"},
		Nodes: []TopologyNode{
			{ID: "Node/worker-1", Kind: "Node", Name: "worker-1"},
			{ID: "Pod/default/my-pod", Kind: "Pod", Name: "my-pod", Namespace: "default"},
			{ID: "Pod/default/unrelated-pod", Kind: "Pod", Name: "unrelated-pod", Namespace: "default"},
		},
		Edges: []TopologyEdge{
			{ID: "e1", Source: "Pod/default/my-pod", Target: "Node/worker-1", RelationshipCategory: "scheduling"},
			{ID: "e2", Source: "Pod/default/unrelated-pod", Target: "Node/worker-1", RelationshipCategory: "scheduling"},
		},
	}
	filter := &ViewFilter{}
	result := filter.Filter(resp, Options{Mode: ViewModeResource, Resource: "Pod/default/my-pod", Depth: 3})
	nodeIDs := make(map[string]bool)
	for _, n := range result.Nodes {
		nodeIDs[n.ID] = true
	}
	if !nodeIDs["Pod/default/my-pod"] {
		t.Error("missing focus pod")
	}
	if !nodeIDs["Node/worker-1"] {
		t.Error("missing node as leaf")
	}
	if nodeIDs["Pod/default/unrelated-pod"] {
		t.Error("BUG: node explosion — unrelated-pod should NOT be included")
	}
}

// TestViewFilter_Resource_NoCrossResourceLeakage verifies that a pod's topology
// does not include sibling pods or unrelated services (Bug 3 from topology spec).
func TestViewFilter_Resource_NoCrossResourceLeakage(t *testing.T) {
	resp := &TopologyResponse{
		Metadata: TopologyMetadata{ClusterID: "test"},
		Nodes: []TopologyNode{
			{ID: "Deployment/default/app", Kind: "Deployment", Name: "app", Namespace: "default"},
			{ID: "ReplicaSet/default/app-rs", Kind: "ReplicaSet", Name: "app-rs", Namespace: "default"},
			{ID: "Pod/default/app-pod-0", Kind: "Pod", Name: "app-pod-0", Namespace: "default"},
			{ID: "Pod/default/app-pod-1", Kind: "Pod", Name: "app-pod-1", Namespace: "default"},
			{ID: "Service/default/app-svc", Kind: "Service", Name: "app-svc", Namespace: "default"},
			{ID: "Service/default/other-svc", Kind: "Service", Name: "other-svc", Namespace: "default"},
		},
		Edges: []TopologyEdge{
			{ID: "e1", Source: "Deployment/default/app", Target: "ReplicaSet/default/app-rs", RelationshipCategory: "ownership"},
			{ID: "e2", Source: "ReplicaSet/default/app-rs", Target: "Pod/default/app-pod-0", RelationshipCategory: "ownership"},
			{ID: "e3", Source: "ReplicaSet/default/app-rs", Target: "Pod/default/app-pod-1", RelationshipCategory: "ownership"},
			{ID: "e4", Source: "Service/default/app-svc", Target: "Pod/default/app-pod-0", RelationshipCategory: "networking"},
			// other-svc is NOT connected to app-pod-0
		},
	}
	filter := &ViewFilter{}
	// Focus on pod-0, depth=2 (Extended)
	result := filter.Filter(resp, Options{Mode: ViewModeResource, Resource: "Pod/default/app-pod-0", Depth: 2})
	nodeIDs := make(map[string]bool)
	for _, n := range result.Nodes {
		nodeIDs[n.ID] = true
	}
	if !nodeIDs["Pod/default/app-pod-0"] {
		t.Error("missing focus pod")
	}
	if !nodeIDs["ReplicaSet/default/app-rs"] {
		t.Error("missing ReplicaSet at depth 1")
	}
	if !nodeIDs["Service/default/app-svc"] {
		t.Error("missing Service at depth 1")
	}
	if !nodeIDs["Deployment/default/app"] {
		t.Error("missing Deployment at depth 2 via ownership chain")
	}
	// other-svc has no connection to app-pod-0 — MUST NOT appear
	if nodeIDs["Service/default/other-svc"] {
		t.Error("BUG: cross-resource leakage — other-svc should NOT be included")
	}
}

// TestViewFilter_Resource_FullMode verifies Full mode (depth=3) follows the complete
// dependency chain without hub expansion.
func TestViewFilter_Resource_FullMode(t *testing.T) {
	resp := &TopologyResponse{
		Metadata: TopologyMetadata{ClusterID: "test"},
		Nodes: []TopologyNode{
			{ID: "Ingress/default/my-ingress", Kind: "Ingress", Name: "my-ingress", Namespace: "default"},
			{ID: "Service/default/my-svc", Kind: "Service", Name: "my-svc", Namespace: "default"},
			{ID: "Endpoints/default/my-ep", Kind: "Endpoints", Name: "my-ep", Namespace: "default"},
			{ID: "Pod/default/my-pod", Kind: "Pod", Name: "my-pod", Namespace: "default"},
			{ID: "ReplicaSet/default/my-rs", Kind: "ReplicaSet", Name: "my-rs", Namespace: "default"},
			{ID: "Deployment/default/my-deploy", Kind: "Deployment", Name: "my-deploy", Namespace: "default"},
			{ID: "ConfigMap/default/my-cm", Kind: "ConfigMap", Name: "my-cm", Namespace: "default"},
			{ID: "Node/worker-1", Kind: "Node", Name: "worker-1"},
		},
		Edges: []TopologyEdge{
			{ID: "e1", Source: "Ingress/default/my-ingress", Target: "Service/default/my-svc", RelationshipCategory: "networking"},
			{ID: "e2", Source: "Service/default/my-svc", Target: "Endpoints/default/my-ep", RelationshipCategory: "networking"},
			{ID: "e3", Source: "Endpoints/default/my-ep", Target: "Pod/default/my-pod", RelationshipCategory: "networking"},
			{ID: "e4", Source: "ReplicaSet/default/my-rs", Target: "Pod/default/my-pod", RelationshipCategory: "ownership"},
			{ID: "e5", Source: "Deployment/default/my-deploy", Target: "ReplicaSet/default/my-rs", RelationshipCategory: "ownership"},
			{ID: "e6", Source: "Pod/default/my-pod", Target: "ConfigMap/default/my-cm", RelationshipCategory: "configuration"},
			{ID: "e7", Source: "Pod/default/my-pod", Target: "Node/worker-1", RelationshipCategory: "scheduling"},
		},
	}
	filter := &ViewFilter{}
	// Full mode from the Service — should reach entire chain
	result := filter.Filter(resp, Options{Mode: ViewModeResource, Resource: "Service/default/my-svc", Depth: 3})
	nodeIDs := make(map[string]bool)
	for _, n := range result.Nodes {
		nodeIDs[n.ID] = true
	}
	// Full chain should be reachable
	for _, expected := range []string{
		"Service/default/my-svc", "Ingress/default/my-ingress",
		"Endpoints/default/my-ep", "Pod/default/my-pod",
		"ReplicaSet/default/my-rs", "Deployment/default/my-deploy",
		"ConfigMap/default/my-cm", "Node/worker-1",
	} {
		if !nodeIDs[expected] {
			t.Errorf("Full mode: missing %s", expected)
		}
	}
}

// TestViewFilter_Resource_DirectVsExtended verifies Direct and Extended produce different results.
// Uses Deployment as focus — Direct sees ReplicaSet/Service, Extended adds Pod/ConfigMap/etc.
func TestViewFilter_Resource_DirectVsExtended(t *testing.T) {
	resp := &TopologyResponse{
		Metadata: TopologyMetadata{ClusterID: "test"},
		Nodes: []TopologyNode{
			{ID: "Deployment/default/app", Kind: "Deployment", Name: "app", Namespace: "default"},
			{ID: "ReplicaSet/default/app-rs", Kind: "ReplicaSet", Name: "app-rs", Namespace: "default"},
			{ID: "Pod/default/app-pod", Kind: "Pod", Name: "app-pod", Namespace: "default"},
			{ID: "Service/default/app-svc", Kind: "Service", Name: "app-svc", Namespace: "default"},
			{ID: "ConfigMap/default/app-cm", Kind: "ConfigMap", Name: "app-cm", Namespace: "default"},
		},
		Edges: []TopologyEdge{
			{ID: "e1", Source: "Deployment/default/app", Target: "ReplicaSet/default/app-rs", RelationshipCategory: "ownership"},
			{ID: "e2", Source: "ReplicaSet/default/app-rs", Target: "Pod/default/app-pod", RelationshipCategory: "ownership"},
			{ID: "e3", Source: "Service/default/app-svc", Target: "Pod/default/app-pod", RelationshipCategory: "networking"},
			{ID: "e4", Source: "Pod/default/app-pod", Target: "ConfigMap/default/app-cm", RelationshipCategory: "configuration"},
		},
	}
	filter := &ViewFilter{}

	direct := filter.Filter(resp, Options{Mode: ViewModeResource, Resource: "Deployment/default/app", Depth: 1})
	extended := filter.Filter(resp, Options{Mode: ViewModeResource, Resource: "Deployment/default/app", Depth: 2})

	directIDs := make(map[string]bool)
	for _, n := range direct.Nodes {
		directIDs[n.ID] = true
	}
	extendedIDs := make(map[string]bool)
	for _, n := range extended.Nodes {
		extendedIDs[n.ID] = true
	}

	// Direct: Deployment + ReplicaSet (1 hop via ownership)
	if !directIDs["Deployment/default/app"] || !directIDs["ReplicaSet/default/app-rs"] {
		t.Error("Direct mode missing Deployment or ReplicaSet")
	}

	// Extended: should additionally include Pod (2 hops via ownership chain)
	if !extendedIDs["Pod/default/app-pod"] {
		t.Error("Extended mode should include Pod at depth 2")
	}

	if len(extended.Nodes) <= len(direct.Nodes) {
		t.Errorf("Extended should have more nodes than Direct: extended=%d, direct=%d",
			len(extended.Nodes), len(direct.Nodes))
	}
}

func TestViewFilter_RBAC(t *testing.T) {
	resp := newTestResponse()
	filter := &ViewFilter{}
	result := filter.Filter(resp, Options{Mode: ViewModeRBAC})
	for _, n := range result.Nodes {
		switch n.Kind {
		case "ServiceAccount", "Role", "RoleBinding", "ClusterRole", "ClusterRoleBinding":
			// OK
		default:
			t.Errorf("unexpected kind in RBAC view: %s", n.Kind)
		}
	}
}

func TestViewFilter_NilResponse(t *testing.T) {
	filter := &ViewFilter{}
	result := filter.Filter(nil, Options{Mode: ViewModeCluster})
	if result != nil {
		t.Error("expected nil result for nil input")
	}
}
