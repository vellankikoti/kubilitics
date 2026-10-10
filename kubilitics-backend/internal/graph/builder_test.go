package graph

import (
	"fmt"
	"testing"
	"time"

	"github.com/kubilitics/kubilitics-backend/internal/models"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildSnapshot_RegistersAllResourceKinds(t *testing.T) {
	replicas := int32(2)
	res := &ClusterResources{
		Pods: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "web-abc", Namespace: "default", Labels: map[string]string{"app": "web"}}},
		},
		Deployments: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
				},
			},
		},
		Services: []corev1.Service{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "web-svc", Namespace: "default"},
				Spec:       corev1.ServiceSpec{Selector: map[string]string{"app": "web"}},
			},
		},
		ConfigMaps: []corev1.ConfigMap{
			{ObjectMeta: metav1.ObjectMeta{Name: "app-config", Namespace: "default"}},
		},
	}

	snap := BuildSnapshot(res, false, nil, nil)

	// Verify all 4 resource kinds appear as nodes
	expected := map[string]bool{
		"Pod/default/web-abc":          true,
		"Deployment/default/web":       true,
		"Service/default/web-svc":      true,
		"ConfigMap/default/app-config": true,
	}

	for key := range expected {
		if _, ok := snap.Nodes[key]; !ok {
			t.Errorf("expected node %s to be registered, but it was not found", key)
		}
	}

	// Verify basic snapshot fields
	// TotalWorkloads = Deployments + StatefulSets + DaemonSets + Services + Jobs + CronJobs = 1 + 1 = 2
	if snap.TotalWorkloads != 2 {
		t.Errorf("expected TotalWorkloads=2 (1 deployment + 1 service), got %d", snap.TotalWorkloads)
	}
	if snap.BuiltAt == 0 {
		t.Error("expected BuiltAt to be non-zero")
	}
	if len(snap.Namespaces) == 0 {
		t.Error("expected at least one namespace")
	}
	if !snap.Namespaces["default"] {
		t.Error("expected 'default' namespace to be tracked")
	}

	// Verify scores are populated for all nodes
	for key := range snap.Nodes {
		if _, ok := snap.NodeScores[key]; !ok {
			t.Errorf("expected NodeScores to contain key %s", key)
		}
	}

	// Verify edges were created (Service selects Deployment via pod)
	if len(snap.Edges) == 0 {
		t.Error("expected at least one edge (Service -> Deployment)")
	}
}

// buildWideFanInBundle returns a ClusterResources with one Service and
// `ingressCount` Ingresses that all route to it via DefaultBackend. Blast
// radius of the Service then has `ingressCount` affected nodes — the shape
// that made computeSingleResourceBlast's old per-affected-node
// buildFailurePath (a fresh BFS over the whole Reverse graph, every time)
// expensive: each of those re-runs touches the entire graph, not just the
// one hop to its own Ingress.
func buildWideFanInBundle(ingressCount int) *ClusterResources {
	res := &ClusterResources{
		Services: []corev1.Service{
			{ObjectMeta: metav1.ObjectMeta{Name: "shared-svc", Namespace: "default"}},
		},
	}
	for i := 0; i < ingressCount; i++ {
		res.Ingresses = append(res.Ingresses, networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("ing-%d", i), Namespace: "default"},
			Spec: networkingv1.IngressSpec{
				DefaultBackend: &networkingv1.IngressBackend{
					Service: &networkingv1.IngressServiceBackend{Name: "shared-svc"},
				},
			},
		})
	}
	return res
}

// TestComputeSingleResourceBlast_WideFanIn_CompletesQuickly is a scale
// regression guard for the bfsParents/pathFromParents optimization. Measured
// directly (git-stash the fix and rerun to reproduce): at 2000 affected
// nodes the old per-node fresh-BFS approach took ~880ms; the fix brings it
// to ~40ms — a ~22x improvement at this modest scale, growing further at
// real cluster scale since the old approach was O(affectedNodes * (V+E)).
// 300ms is a deliberately generous ceiling (comfortably above the fixed
// ~40ms, comfortably below the broken ~880ms) so this stays stable on
// slower CI hardware while still catching a reintroduced O(n*(V+E)) pattern.
func TestComputeSingleResourceBlast_WideFanIn_CompletesQuickly(t *testing.T) {
	const ingressCount = 2000
	snap := BuildSnapshot(buildWideFanInBundle(ingressCount), false, nil, nil)
	target := models.ResourceRef{Kind: "Service", Namespace: "default", Name: "shared-svc"}

	start := time.Now()
	result, err := snap.ComputeBlastRadius(target)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ComputeBlastRadius: %v", err)
	}
	if result.TotalAffected != ingressCount {
		t.Fatalf("expected %d affected nodes, got %d", ingressCount, result.TotalAffected)
	}
	if elapsed > 300*time.Millisecond {
		t.Errorf("ComputeBlastRadius with %d affected nodes took %v — expected well under 300ms; "+
			"likely a reintroduced per-affected-node BFS re-run", ingressCount, elapsed)
	}
}

func BenchmarkComputeSingleResourceBlast_WideFanIn(b *testing.B) {
	snap := BuildSnapshot(buildWideFanInBundle(2000), false, nil, nil)
	target := models.ResourceRef{Kind: "Service", Namespace: "default", Name: "shared-svc"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = snap.ComputeBlastRadius(target)
	}
}

// TestBuildSnapshot_FailurePathUsesRealEdgeTypes is a correctness seal for
// precomputing the edge-type lookup index in BuildSnapshot (instead of
// snapshot.go's edgeType() doing a full linear scan of s.Edges per hop, per
// affected node — O(affectedNodes * pathLength * totalEdges) in
// computeSingleResourceBlast, the dominant cost in Blast Radius at scale).
// Runs buildFailurePath against a snapshot built by the real BuildSnapshot
// (not the hand-constructed buildTestSnapshot() helper other tests use,
// which never populates the new index and exercises the fallback path
// instead) to prove the indexed lookup finds the real edges the inference
// functions produced, not just the "dependency" fallback placeholder.
func TestBuildSnapshot_FailurePathUsesRealEdgeTypes(t *testing.T) {
	replicas := int32(1)
	res := &ClusterResources{
		Pods: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "web-abc", Namespace: "default", Labels: map[string]string{"app": "web"}}},
		},
		Deployments: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
				},
			},
		},
		Services: []corev1.Service{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "web-svc", Namespace: "default"},
				Spec:       corev1.ServiceSpec{Selector: map[string]string{"app": "web"}},
			},
		},
	}
	snap := BuildSnapshot(res, false, nil, nil)

	podKey := refKey(models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "web-abc"})
	svcKey := refKey(models.ResourceRef{Kind: "Service", Namespace: "default", Name: "web-svc"})

	hops := snap.buildFailurePath(podKey, svcKey)
	if len(hops) == 0 {
		t.Fatal("expected a non-empty failure path from Pod to Service")
	}
	for _, h := range hops {
		if h.EdgeType == "" || h.EdgeType == "dependency" {
			t.Errorf("hop %s/%s -> %s/%s: EdgeType %q looks like the no-match fallback, expected a real inferred type",
				h.From.Kind, h.From.Name, h.To.Kind, h.To.Name, h.EdgeType)
		}
	}
}

// TestBuildSnapshot_ServicePodLabelsCorrectAcrossMultipleServices is a
// correctness seal for optimizing Step 5c's service-pod-label lookup, which
// did a linear scan of res.Pods per endpoint address (O(endpoints*pods) —
// near-O(n²) in practice, since most pods sit behind exactly one service).
// Multiple services/pods here so an index-based rewrite can't accidentally
// mix up which pod's labels belong to which service.
func TestBuildSnapshot_ServicePodLabelsCorrectAcrossMultipleServices(t *testing.T) {
	res := &ClusterResources{
		Pods: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "web-1", Namespace: "default", Labels: map[string]string{"app": "web", "tier": "frontend"}}},
			{ObjectMeta: metav1.ObjectMeta{Name: "api-1", Namespace: "default", Labels: map[string]string{"app": "api", "tier": "backend"}}},
			{ObjectMeta: metav1.ObjectMeta{Name: "web-1", Namespace: "other-ns", Labels: map[string]string{"app": "web", "tier": "other-ns-frontend"}}},
		},
		Endpoints: []corev1.Endpoints{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "web-svc", Namespace: "default"},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{
						{TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "web-1", Namespace: "default"}},
					},
				}},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "api-svc", Namespace: "default"},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{
						{TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "api-1", Namespace: "default"}},
					},
				}},
			},
			{
				// Same pod name as default/web-1, but a different namespace —
				// must not be cross-matched by name alone.
				ObjectMeta: metav1.ObjectMeta{Name: "web-svc", Namespace: "other-ns"},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{
						{TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "web-1", Namespace: "other-ns"}},
					},
				}},
			},
		},
	}

	snap := BuildSnapshot(res, false, nil, nil)

	if got := snap.ServicePodLabels["Service/default/web-svc"]["tier"]; got != "frontend" {
		t.Errorf("Service/default/web-svc: expected tier=frontend, got %q", got)
	}
	if got := snap.ServicePodLabels["Service/default/api-svc"]["tier"]; got != "backend" {
		t.Errorf("Service/default/api-svc: expected tier=backend, got %q", got)
	}
	if got := snap.ServicePodLabels["Service/other-ns/web-svc"]["tier"]; got != "other-ns-frontend" {
		t.Errorf("Service/other-ns/web-svc: expected tier=other-ns-frontend, got %q — cross-namespace pod-name collision", got)
	}
}

func TestBuildSnapshot_EmptyResources(t *testing.T) {
	res := &ClusterResources{}

	snap := BuildSnapshot(res, false, nil, nil)

	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if len(snap.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(snap.Nodes))
	}
	if len(snap.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(snap.Edges))
	}
	if snap.TotalWorkloads != 0 {
		t.Errorf("expected TotalWorkloads=0, got %d", snap.TotalWorkloads)
	}
	if snap.BuiltAt == 0 {
		t.Error("expected BuiltAt to be non-zero")
	}
	if snap.Nodes == nil {
		t.Error("expected Nodes map to be initialized (not nil)")
	}
	if snap.Forward == nil {
		t.Error("expected Forward map to be initialized (not nil)")
	}
	if snap.Reverse == nil {
		t.Error("expected Reverse map to be initialized (not nil)")
	}
}

func TestBuildSnapshot_ReplicaCount(t *testing.T) {
	replicas := int32(3)
	res := &ClusterResources{
		Deployments: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "prod"},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
				},
			},
		},
	}

	snap := BuildSnapshot(res, false, nil, nil)

	key := "Deployment/prod/api"
	if got := snap.NodeReplicas[key]; got != 3 {
		t.Errorf("expected replica count 3 for %s, got %d", key, got)
	}
}

// --- C-BE-4: collectResources nil response doesn't crash BuildSnapshot ---

func TestBuildSnapshot_NilResources(t *testing.T) {
	// BuildSnapshot should not panic when passed nil ClusterResources
	snap := BuildSnapshot(nil, false, nil, nil)

	if snap == nil {
		t.Fatal("expected non-nil snapshot from nil resources")
	}
	if snap.Nodes == nil {
		t.Error("expected Nodes map to be initialized")
	}
	if snap.Forward == nil {
		t.Error("expected Forward map to be initialized")
	}
	if snap.Reverse == nil {
		t.Error("expected Reverse map to be initialized")
	}
	if snap.NodeScores == nil {
		t.Error("expected NodeScores map to be initialized")
	}
	if snap.NodeRisks == nil {
		t.Error("expected NodeRisks map to be initialized")
	}
	if snap.NodeReplicas == nil {
		t.Error("expected NodeReplicas map to be initialized")
	}
	if snap.NodeHasHPA == nil {
		t.Error("expected NodeHasHPA map to be initialized")
	}
	if snap.NodeHasPDB == nil {
		t.Error("expected NodeHasPDB map to be initialized")
	}
	if snap.NodeIngress == nil {
		t.Error("expected NodeIngress map to be initialized")
	}
	if snap.BuiltAt == 0 {
		t.Error("expected BuiltAt to be non-zero")
	}
	if len(snap.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(snap.Nodes))
	}
}

func TestGetReplicaCountFromResources(t *testing.T) {
	replicas := int32(5)
	res := &ClusterResources{
		Deployments: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
				Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
			},
		},
	}

	got := getReplicaCountFromResources(res, "Deployment", "web", "default")
	if got != 5 {
		t.Errorf("expected 5, got %d", got)
	}

	got = getReplicaCountFromResources(res, "Service", "web", "default")
	if got != 0 {
		t.Errorf("expected 0 for Service kind, got %d", got)
	}

	got = getReplicaCountFromResources(res, "Deployment", "missing", "default")
	if got != 0 {
		t.Errorf("expected 0 for missing deployment, got %d", got)
	}
}
