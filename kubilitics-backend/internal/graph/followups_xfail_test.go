package graph

import (
	"testing"

	"github.com/kubilitics/kubilitics-backend/internal/models"
	"github.com/kubilitics/kubilitics-backend/internal/otel"
	corev1 "k8s.io/api/core/v1"
)

// TestFollowupTopo1_BlastPercentDoesNotInflateFromFanOut is the pinned
// reproduction for FOLLOWUP-TOPO-1 (see KNOWN-ISSUES.md). It is skipped,
// not fixed: computeBlastRadiusPercent sums classificationWeight once per
// ServiceImpact, once per IngressImpact routing to that service, and once
// per OTel ConsumerImpact calling it — so ONE broken service with 1 ingress
// and 3 consumers contributes ~5x the weight of a service with no fan-out,
// even though exactly one workload actually broke. Divided by total
// workload count, this inflates the percentage by fan-out degree rather
// than by the number of distinct affected resources.
//
// Acceptance criterion this pins (not yet met): overlapping paths and
// taxonomies must not inflate the distinct affected-workload percentage.
// Un-skip this test once a fix lands; it should then assert something
// close to 1/10 = 10%, not today's measured 22%.
func TestFollowupTopo1_BlastPercentDoesNotInflateFromFanOut(t *testing.T) {
	svc := models.ResourceRef{Kind: "Service", Namespace: "default", Name: "checkout"}
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "checkout"}
	ing := models.ResourceRef{Kind: "Ingress", Namespace: "default", Name: "checkout-ing"}
	otherDep := models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "unrelated"}

	svcKey := refKey(svc)
	depKey := refKey(dep)
	ingKey := refKey(ing)
	otherKey := refKey(otherDep)

	snap := &GraphSnapshot{}
	snap.EnsureMaps()
	snap.Nodes[svcKey] = svc
	snap.Nodes[depKey] = dep
	snap.Nodes[ingKey] = ing
	snap.Nodes[otherKey] = otherDep
	snap.PodOwners[refKey(models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "checkout-pod"})] = depKey
	snap.NodeReplicas[depKey] = 1
	snap.ServiceEndpoints[svcKey] = []corev1.EndpointAddress{
		{IP: "10.0.0.1", TargetRef: &corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "checkout-pod"}},
	}
	snap.NodeIngress[ingKey] = []string{"checkout.example.com"}
	snap.Forward[ingKey] = map[string]bool{svcKey: true}
	snap.Forward[svcKey] = map[string]bool{depKey: true}
	snap.OTelServiceMap = &otel.ServiceMap{
		Edges: []otel.ServiceEdge{
			{Source: "web-frontend", Target: "checkout", Count: 100},
			{Source: "mobile-bff", Target: "checkout", Count: 100},
			{Source: "partner-api", Target: "checkout", Count: 100},
		},
	}
	snap.TotalWorkloads = 10 // 10 workloads cluster-wide; only "checkout" is broken

	result, err := snap.computeSingleResourceBlast(dep, FailureModeWorkloadDeletion)
	if err != nil {
		t.Fatal(err)
	}

	// 1 of 10 workloads broken should read close to 10%, not ~22% inflated
	// by counting the same failure once per service+ingress+consumer edge.
	if result.BlastRadiusPercent > 15.0 {
		t.Errorf("expected blast radius close to 10%% for 1/10 workloads broken, got %.1f%% (fan-out inflation)", result.BlastRadiusPercent)
	}
}

// TestFollowupTopo2_OTelMatchingIsNamespaceScoped is the pinned
// reproduction for FOLLOWUP-TOPO-2's FULL resolution (see KNOWN-ISSUES.md).
// A partial, safe mitigation has landed (see
// TestComputeConfidence_AmbiguousNameAcrossNamespacesGetsNoFalseTraceCredit
// and the nameIsAmbiguousAcrossNamespaces guard applied at computeConfidence,
// computeBlastRadiusPercent's traffic-hotspot amplification, and
// computeSingleResourceBlast's consumer-count/traffic matching): when a
// name is ambiguous across namespaces, NONE of the same-named targets get
// false trace credit, closing the false-positive direction.
//
// This specific test still cannot pass without a real architectural
// change, confirmed by direct inspection, not guessed: internal/otel's
// ServiceEdge (internal/otel/types.go) is Source/Target name-only, with no
// namespace field, and internal/otel/store.go's GetServiceMap SQL query
// (GROUP BY service_name) does not aggregate by namespace either — even
// though the underlying spans table DOES carry k8s_namespace per-span
// (populated from the "k8s.namespace.name" resource attribute in
// receiver.go). The data exists at ingestion time but is discarded before
// it reaches ServiceMap. The graph package's partial mitigation can only
// make an ambiguous name's trace credit go to NEITHER target (safe, but
// also takes credit away from the genuinely-traced one) — it cannot
// correctly attribute the trace to ONLY staging/checkout, which is what
// this test requires. Closing this fully means: add a namespace field to
// ServiceNode/ServiceEdge, and change GetServiceMap's two SQL queries to
// GROUP BY (service_name, k8s_namespace) instead of service_name alone —
// out of scope for a fix confined to internal/graph.
func TestFollowupTopo2_OTelMatchingIsNamespaceScoped(t *testing.T) {
	t.Skip("FOLLOWUP-TOPO-2: full fix needs internal/otel's ServiceMap to carry a namespace (GetServiceMap's SQL groups by service_name only, discarding the k8s_namespace already present on spans) — out of internal/graph's scope. Partial mitigation (ambiguous name -> no false credit to anyone) has landed; see TestComputeConfidence_AmbiguousNameAcrossNamespacesGetsNoFalseTraceCredit.")

	prodTarget := models.ResourceRef{Kind: "Deployment", Namespace: "prod", Name: "checkout"}
	stagingTarget := models.ResourceRef{Kind: "Deployment", Namespace: "staging", Name: "checkout"}

	snap := &GraphSnapshot{}
	snap.EnsureMaps()
	snap.Nodes[refKey(prodTarget)] = prodTarget
	snap.Nodes[refKey(stagingTarget)] = stagingTarget

	// Only the STAGING "checkout" is actually traced — the trace edge
	// carries no namespace, so a namespace-unaware matcher would also
	// credit the PROD "checkout" with trace coverage it doesn't have.
	snap.OTelServiceMap = &otel.ServiceMap{
		Edges: []otel.ServiceEdge{
			{Source: "staging-frontend", Target: "checkout", Count: 50},
		},
	}

	prodScore, prodNote := computeConfidence(snap, &prodTarget)
	stagingScore, _ := computeConfidence(snap, &stagingTarget)

	if prodScore == stagingScore {
		t.Errorf("prod/checkout (untraced) should not share staging/checkout's (traced) confidence score, both got %d", prodScore)
	}
	if prodNote != "Graph topology only — no runtime data available" {
		t.Errorf("prod/checkout should show no trace coverage, got note: %q", prodNote)
	}
}
