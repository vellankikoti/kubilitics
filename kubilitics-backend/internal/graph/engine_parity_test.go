package graph

import (
	"context"
	"testing"

	v2 "github.com/kubilitics/kubilitics-backend/internal/topology/v2"
	"github.com/kubilitics/kubilitics-backend/internal/topology/v2/relationships"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestEngineParity_OwnerSelectorIngressChain checks whether this backend's
// two independently-written relationship engines — internal/graph (used for
// blast radius) and internal/topology/v2/relationships (used for the
// resource topology view) — agree on the three relationship types they both
// implement: Deployment->ReplicaSet->Pod ownership, Service->Pod selector
// matching, and Ingress->Service routing. This is evidence-gathering for the
// "are these two engines actually consistent, or do they diverge" question
// flagged by the baseline investigation — it is deliberately NOT a general
// unification of the two engines, which remains out of scope.
func TestEngineParity_OwnerSelectorIngressChain(t *testing.T) {
	const ns = "prod"
	controllerTrue := true

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: ns, UID: "dep-uid"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
		},
	}
	replicaSet := appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "web-abc", Namespace: ns, UID: "rs-uid",
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "Deployment", Name: "web", UID: "dep-uid", Controller: &controllerTrue},
			},
		},
	}
	podLabels := map[string]string{"app": "web"}
	pod1 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "web-abc-1", Namespace: ns, Labels: podLabels,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet", Name: "web-abc", UID: "rs-uid", Controller: &controllerTrue},
			},
		},
	}
	pod2 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "web-abc-2", Namespace: ns, Labels: podLabels,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet", Name: "web-abc", UID: "rs-uid", Controller: &controllerTrue},
			},
		},
	}
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "web-svc", Namespace: ns},
		Spec:       corev1.ServiceSpec{Selector: podLabels},
	}
	pathType := networkingv1.PathTypePrefix
	ingress := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "web-ing", Namespace: ns},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{{
						PathType: &pathType,
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: "web-svc",
								Port: networkingv1.ServiceBackendPort{Number: 80},
							},
						},
					}},
				}},
			}},
		},
	}

	// --- Build via internal/graph (blast-radius engine) ---
	gRes := &ClusterResources{
		Deployments: []appsv1.Deployment{deployment},
		ReplicaSets: []appsv1.ReplicaSet{replicaSet},
		Pods:        []corev1.Pod{pod1, pod2},
		Services:    []corev1.Service{service},
		Ingresses:   []networkingv1.Ingress{ingress},
	}
	gSnap := BuildSnapshot(gRes, false, nil, nil)

	depKey := "Deployment/" + ns + "/web"
	rsKey := "ReplicaSet/" + ns + "/web-abc"
	pod1Key := "Pod/" + ns + "/web-abc-1"
	pod2Key := "Pod/" + ns + "/web-abc-2"
	svcKey := "Service/" + ns + "/web-svc"
	ingKey := "Ingress/" + ns + "/web-ing"

	gHasPath := func(from, to string) bool {
		return gSnap.Forward[from] != nil && gSnap.Forward[from][to]
	}

	// --- Build via internal/topology/v2 (resource topology engine) ---
	bundle := &v2.ResourceBundle{
		Deployments: []appsv1.Deployment{deployment},
		ReplicaSets: []appsv1.ReplicaSet{replicaSet},
		Pods:        []corev1.Pod{pod1, pod2},
		Services:    []corev1.Service{service},
		Ingresses:   []networkingv1.Ingress{ingress},
	}
	registry := relationships.NewDefaultRegistry()
	v2Edges, _, err := registry.MatchAll(context.Background(), bundle)
	if err != nil {
		t.Fatalf("v2 registry.MatchAll returned an error: %v", err)
	}
	v2HasEdge := func(from, to string) bool {
		for _, e := range v2Edges {
			if (e.Source == from && e.Target == to) || (e.Source == to && e.Target == from) {
				return true
			}
		}
		return false
	}

	// --- Ingress->Service: both engines must agree — there is no owner
	// abstraction involved (an Ingress's backend is always a Service by
	// definition), so disagreement here would be a real bug, not a
	// documented difference.
	t.Run("Ingress routes to Service", func(t *testing.T) {
		gHas := gHasPath(ingKey, svcKey) || gHasPath(svcKey, ingKey)
		v2Has := v2HasEdge(ingKey, svcKey)
		if gHas != v2Has || !gHas {
			t.Errorf("Ingress->Service disagreement or missing edge: internal/graph=%v, internal/topology/v2=%v", gHas, v2Has)
		}
	})

	// --- Everything below is an intentional, consistent divergence, not a
	// bug: internal/graph's blast-radius engine systematically collapses
	// every relationship to the nearest known STABLE OWNER, while
	// internal/topology/v2 preserves full node-level granularity for the
	// topology view. This is true for BOTH relationship types that can
	// reach a Pod indirectly:
	//
	//  1. Ownership: inferOwnerRefDeps never models ReplicaSet as a node at
	//     all (confirmed: ReplicaSets is only read for replica-count lookups
	//     in getReplicaCountFromResources, never for node registration or
	//     edge creation) — it links Deployment directly to Pod.
	//  2. Selection: inferSelectorDeps explicitly says "link to owner
	//     workload if known, otherwise to the pod" — so once a Pod's owner
	//     is known (case 1 above), a Service selecting that Pod gets linked
	//     to the Deployment, not the Pod, collapsing Service->Pod into
	//     Service->Deployment.
	//
	// Both are coherent with graph's actual purpose: blast radius answers
	// "which stable workloads are affected," not "show me every ephemeral
	// pod." v2's topology view needs the opposite — full granularity for a
	// human browsing the graph. Confirmed via repo-wide grep that
	// ReplicaSet is never a supported blast-radius target kind in
	// classify.go/snapshot.go, so neither gap has a practical correctness
	// consequence for blast radius today. See
	// TestEngineParity_GraphUsesDirectPodToDeploymentEdge for the positive
	// assertion that the direct Deployment->Pod edge does exist and serves
	// its purpose.
	t.Run("ownership and selection relationships intentionally collapse to owner (documented, not a bug)", func(t *testing.T) {
		v2HasRSToDep := v2HasEdge(rsKey, depKey)
		v2HasPodToRS := v2HasEdge(pod1Key, rsKey) && v2HasEdge(pod2Key, rsKey)
		v2HasSvcToPod := v2HasEdge(svcKey, pod1Key) && v2HasEdge(svcKey, pod2Key)
		if !v2HasRSToDep || !v2HasPodToRS || !v2HasSvcToPod {
			t.Errorf("expected internal/topology/v2 to preserve full granularity (RS->Deployment=%v, Pod->RS=%v, Service->Pod=%v) — if any of these become false, v2's own relationship matchers regressed, independent of internal/graph's design",
				v2HasRSToDep, v2HasPodToRS, v2HasSvcToPod)
		}

		gHasSvcToDep := gHasPath(svcKey, depKey) || gHasPath(depKey, svcKey)
		if !gHasSvcToDep {
			t.Errorf("expected internal/graph to collapse Service->Pod into Service->Deployment once the Pod's owner is known — it did not; the 'link to owner workload if known' behavior in inferSelectorDeps may have changed, which would mean this comment block is now stale")
		}
		gHasRSAnyEdge := len(gSnap.Forward[rsKey]) > 0 || len(gSnap.Reverse[rsKey]) > 0
		if gHasRSAnyEdge {
			t.Logf("internal/graph now has an edge touching ReplicaSet (%s) where previously it had none — this comment and the engine_parity tests should be revisited, since the documented divergence may no longer be accurate", rsKey)
		}
	})
}

// TestEngineParity_GraphUsesDirectPodToDeploymentEdge documents the actual,
// deliberate difference found by TestEngineParity_OwnerSelectorIngressChain:
// internal/graph's inferOwnerRefDeps matches Pods directly to Deployments by
// selector (skipping ReplicaSet as an intermediate node entirely), while
// internal/topology/v2 walks the real OwnerReference chain through
// ReplicaSet. This is NOT a bug in internal/graph — a Deployment's blast
// radius only cares "which Pods does this affect," and the direct edge
// answers that correctly — but it means internal/graph's Forward/Reverse
// maps never contain a ReplicaSet node at all for this chain, which a
// consumer expecting ReplicaSet-level detail (e.g. "which ReplicaSet owns
// this Pod") would not find there.
func TestEngineParity_GraphUsesDirectPodToDeploymentEdge(t *testing.T) {
	const ns = "prod"
	controllerTrue := true
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: ns, UID: "dep-uid"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
		},
	}
	replicaSet := appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "web-abc", Namespace: ns, UID: "rs-uid",
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "Deployment", Name: "web", UID: "dep-uid", Controller: &controllerTrue},
			},
		},
	}
	pod1 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "web-abc-1", Namespace: ns, Labels: map[string]string{"app": "web"},
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet", Name: "web-abc", UID: "rs-uid", Controller: &controllerTrue},
			},
		},
	}
	gRes := &ClusterResources{
		Deployments: []appsv1.Deployment{deployment},
		ReplicaSets: []appsv1.ReplicaSet{replicaSet},
		Pods:        []corev1.Pod{pod1},
	}
	gSnap := BuildSnapshot(gRes, false, nil, nil)
	depKey := "Deployment/" + ns + "/web"
	pod1Key := "Pod/" + ns + "/web-abc-1"
	rsKey := "ReplicaSet/" + ns + "/web-abc"

	directEdge := gSnap.Forward[depKey] != nil && gSnap.Forward[depKey][pod1Key]
	if !directEdge {
		t.Errorf("expected internal/graph to connect Pod directly to Deployment via selector-matching (bypassing ReplicaSet) — it did not; this contradicts the documented 'selector-based direct match' design in inferOwnerRefDeps and needs real investigation, not just documentation")
	}
	rsIsNode := false
	for k := range gSnap.Nodes {
		if k == rsKey {
			rsIsNode = true
		}
	}
	t.Logf("ReplicaSet registered as a node in internal/graph's snapshot: %v (expected: true, since BuildSnapshot registers ALL resources as nodes even without edges — but it has NO Forward/Reverse edge to either Pod or Deployment)", rsIsNode)
}
