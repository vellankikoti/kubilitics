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
// reproduction for FOLLOWUP-TOPO-2 (see KNOWN-ISSUES.md). It is skipped,
// not fixed: OTel service-map edge matching (computeConfidence's trace
// check, computeBlastRadiusPercent's traffic-hotspot amplification, and
// consumer-count matching in computeSingleResourceBlast) all compare
// edge.Target/edge.Source to a bare resource NAME via strings.EqualFold,
// with no namespace component — so a trace edge for "checkout" in
// namespace "staging" is indistinguishable from a resource named
// "checkout" in namespace "prod".
//
// Acceptance criterion this pins (not yet met): identical names in
// different namespaces remain isolated unless an explicitly supported
// cross-namespace relationship establishes a connection. Un-skip this test
// once OTel edges carry (or are resolved against) a namespace and the
// match is namespace-aware; it should then assert the prod target's
// confidence/trace signal is unaffected by a same-named staging service.
func TestFollowupTopo2_OTelMatchingIsNamespaceScoped(t *testing.T) {
	t.Skip("FOLLOWUP-TOPO-2: known OTel name-only cross-namespace collision, deferred — see KNOWN-ISSUES.md")

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
