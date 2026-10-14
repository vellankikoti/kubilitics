package graph

import (
	"math"
	"strings"
	"testing"

	"github.com/kubilitics/kubilitics-backend/internal/models"
	"github.com/kubilitics/kubilitics-backend/internal/otel"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestComputeLostPods_PodCrash(t *testing.T) {
	snap := &GraphSnapshot{}
	snap.EnsureMaps()
	snap.Nodes["Pod/default/pod-1"] = models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "pod-1"}

	lost := computeLostPods(snap, models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "pod-1"}, FailureModePodCrash)
	if len(lost) != 1 || !lost["Pod/default/pod-1"] {
		t.Errorf("expected exactly pod-1 lost, got %v", lost)
	}
}

func TestComputeLostPods_WorkloadDeletion(t *testing.T) {
	snap := &GraphSnapshot{}
	snap.EnsureMaps()
	snap.Nodes["Deployment/default/app"] = models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "app"}
	snap.Nodes["Pod/default/app-abc-1"] = models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "app-abc-1"}
	snap.Nodes["Pod/default/app-abc-2"] = models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "app-abc-2"}
	snap.Nodes["Pod/default/other-pod"] = models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "other-pod"}
	snap.PodOwners = map[string]string{
		"Pod/default/app-abc-1": "Deployment/default/app",
		"Pod/default/app-abc-2": "Deployment/default/app",
		"Pod/default/other-pod": "Deployment/default/other",
	}

	lost := computeLostPods(snap, models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "app"}, FailureModeWorkloadDeletion)
	if len(lost) != 2 {
		t.Errorf("expected 2 lost pods, got %d: %v", len(lost), lost)
	}
}

func TestClassifyServiceImpact_Broken(t *testing.T) {
	endpoints := map[string][]corev1.EndpointAddress{
		"Service/default/svc-a": {
			{IP: "10.0.0.1", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-1", Namespace: "default"}},
		},
	}
	lostPods := map[string]bool{"Pod/default/pod-1": true}
	impacts := classifyServiceImpact(endpoints, lostPods, nil, nil)
	if len(impacts) != 1 {
		t.Fatalf("expected 1 impact, got %d", len(impacts))
	}
	if impacts[0].Classification != "broken" {
		t.Errorf("expected broken, got %s", impacts[0].Classification)
	}
}

func TestClassifyServiceImpact_SelfHealing(t *testing.T) {
	endpoints := map[string][]corev1.EndpointAddress{
		"Service/default/svc-a": {
			{IP: "10.0.0.1", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-1", Namespace: "default"}},
			{IP: "10.0.0.2", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-2", Namespace: "default"}},
			{IP: "10.0.0.3", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-3", Namespace: "default"}},
			{IP: "10.0.0.4", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-4", Namespace: "default"}},
		},
	}
	lostPods := map[string]bool{"Pod/default/pod-1": true}
	impacts := classifyServiceImpact(endpoints, lostPods, nil, nil)
	if len(impacts) != 1 {
		t.Fatalf("expected 1 impact, got %d", len(impacts))
	}
	if impacts[0].Classification != "self-healing" {
		t.Errorf("expected self-healing, got %s", impacts[0].Classification)
	}
}

func TestClassifyServiceImpact_DegradedWithPDB(t *testing.T) {
	minAvail := intstr.FromInt32(2)
	pdbs := []policyv1.PodDisruptionBudget{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "my-pdb", Namespace: "default"},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MinAvailable: &minAvail,
				Selector:     &metav1.LabelSelector{MatchLabels: map[string]string{"app": "svc-a"}},
			},
		},
	}
	endpoints := map[string][]corev1.EndpointAddress{
		"Service/default/svc-a": {
			{IP: "10.0.0.1", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-1", Namespace: "default"}},
			{IP: "10.0.0.2", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-2", Namespace: "default"}},
			{IP: "10.0.0.3", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-3", Namespace: "default"}},
		},
	}
	lostPods := map[string]bool{"Pod/default/pod-1": true, "Pod/default/pod-2": true}
	impacts := classifyServiceImpact(endpoints, lostPods, pdbs, nil)
	if len(impacts) != 1 {
		t.Fatalf("expected 1 impact, got %d", len(impacts))
	}
	if impacts[0].Classification != "degraded" {
		t.Errorf("expected degraded, got %s", impacts[0].Classification)
	}
	if impacts[0].ThresholdSource != "pdb:my-pdb" {
		t.Errorf("expected pdb source, got %s", impacts[0].ThresholdSource)
	}
}

func TestClassifyServiceImpact_NoImpact(t *testing.T) {
	endpoints := map[string][]corev1.EndpointAddress{
		"Service/default/svc-a": {
			{IP: "10.0.0.1", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-2", Namespace: "default"}},
		},
	}
	lostPods := map[string]bool{"Pod/default/pod-1": true}
	impacts := classifyServiceImpact(endpoints, lostPods, nil, nil)
	if len(impacts) != 0 {
		t.Errorf("expected 0 impacts, got %d", len(impacts))
	}
}

func TestComputeBlastRadiusPercent(t *testing.T) {
	// Three genuinely distinct services (not the same resource observed
	// through overlapping taxonomies) — computeBlastRadiusPercent dedupes
	// by resource identity (FOLLOWUP-TOPO-1 fix), so this must use 3
	// different Service refs to still exercise additive weighting across
	// real, distinct impacts.
	impacts := []models.ServiceImpact{
		{Classification: "broken", Service: models.ResourceRef{Kind: "Service", Namespace: "default", Name: "svc-broken"}},
		{Classification: "degraded", Service: models.ResourceRef{Kind: "Service", Namespace: "default", Name: "svc-degraded"}},
		{Classification: "self-healing", Service: models.ResourceRef{Kind: "Service", Namespace: "default", Name: "svc-self-healing"}},
	}
	pct := computeBlastRadiusPercent(impacts, nil, nil, 10, nil)
	expected := 15.0
	if pct != expected {
		t.Errorf("expected %.1f%%, got %.1f%%", expected, pct)
	}
}

// TestComputeBlastRadiusPercent_DistinctConsumerStillAddsBlast guards against
// the FOLLOWUP-TOPO-1 fix over-correcting: a genuinely distinct consumer
// workload (a different Deployment, not the same resource observed twice)
// must still add its own weight, not be swallowed by the same-resource dedup.
func TestComputeBlastRadiusPercent_DistinctConsumerStillAddsBlast(t *testing.T) {
	svcImpacts := []models.ServiceImpact{
		{Classification: "broken", Service: models.ResourceRef{Kind: "Service", Namespace: "default", Name: "checkout"}},
	}
	consumerImpacts := []models.ConsumerImpact{
		{Classification: "degraded", Workload: models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "other-app"}, DependsOn: "checkout"},
	}

	pct := computeBlastRadiusPercent(svcImpacts, nil, consumerImpacts, 10, nil)

	// 1.0 (broken checkout) + 0.5 (degraded, distinct consumer) = 1.5/10 = 15%
	if pct != 15.0 {
		t.Errorf("expected 15.0%% (distinct service + distinct consumer both counted), got %.1f%%", pct)
	}
}

// TestComputeBlastRadiusPercent_IngressFoldsOntoBackingService is the
// narrower unit-level pin for FOLLOWUP-TOPO-1's ingress-side fold: an
// Ingress impact for the SAME backing Service as an existing ServiceImpact
// must not add a second distinct unit of weight.
func TestComputeBlastRadiusPercent_IngressFoldsOntoBackingService(t *testing.T) {
	svc := models.ResourceRef{Kind: "Service", Namespace: "default", Name: "checkout"}
	ing := models.ResourceRef{Kind: "Ingress", Namespace: "default", Name: "checkout-ing"}

	snap := &GraphSnapshot{}
	snap.EnsureMaps()
	snap.Nodes[refKey(svc)] = svc
	snap.Nodes[refKey(ing)] = ing
	snap.Forward[refKey(ing)] = map[string]bool{refKey(svc): true}

	svcImpacts := []models.ServiceImpact{{Classification: "broken", Service: svc}}
	ingImpacts := []models.IngressImpact{{Classification: "broken", Ingress: ing}}

	withIngress := computeBlastRadiusPercent(svcImpacts, ingImpacts, nil, 10, snap)
	withoutIngress := computeBlastRadiusPercent(svcImpacts, nil, nil, 10, snap)

	if withIngress != withoutIngress {
		t.Errorf("ingress routing to an already-counted service should not change the percentage: with=%.1f%% without=%.1f%%", withIngress, withoutIngress)
	}
}

// TestComputeConfidence_AmbiguousNameAcrossNamespacesGetsNoFalseTraceCredit
// is the safe mitigation for FOLLOWUP-TOPO-2 that IS achievable within the
// graph package: OTel service-map edges carry no namespace (confirmed at
// internal/otel/types.go's ServiceEdge — Source/Target are bare strings),
// so when a name exists under more than one namespace in the snapshot,
// computeConfidence can no longer tell which one a same-named trace edge
// really belongs to. Rather than crediting ALL same-named targets with
// trace coverage (today's false-positive behavior), an ambiguous name
// gets credited to NONE of them — a conservative fix that closes the
// false-positive direction without fabricating namespace data that
// doesn't exist. This does NOT fully resolve FOLLOWUP-TOPO-2 (the
// genuinely-traced target also loses credit when ambiguous); full
// resolution needs internal/otel's ServiceMap to carry a namespace,
// which is out of this fix's scope — see TestFollowupTopo2_* (skipped)
// and the task report for the precise remaining gap.
func TestComputeConfidence_AmbiguousNameAcrossNamespacesGetsNoFalseTraceCredit(t *testing.T) {
	prodTarget := models.ResourceRef{Kind: "Deployment", Namespace: "prod", Name: "checkout"}
	stagingTarget := models.ResourceRef{Kind: "Deployment", Namespace: "staging", Name: "checkout"}

	snap := &GraphSnapshot{}
	snap.EnsureMaps()
	snap.Nodes[refKey(prodTarget)] = prodTarget
	snap.Nodes[refKey(stagingTarget)] = stagingTarget
	snap.OTelServiceMap = &otel.ServiceMap{
		Edges: []otel.ServiceEdge{
			{Source: "staging-frontend", Target: "checkout", Count: 50},
		},
	}

	prodScore, prodNote := computeConfidence(snap, &prodTarget)
	if strings.Contains(prodNote, "traces") {
		t.Errorf("ambiguous name %q should not silently credit prod with trace coverage meant for staging, got note %q (score %d)", "checkout", prodNote, prodScore)
	}

	// Unambiguous case: a uniquely-named resource keeps its real trace credit.
	uniqueTarget := models.ResourceRef{Kind: "Deployment", Namespace: "prod", Name: "billing"}
	uniqueSnap := &GraphSnapshot{}
	uniqueSnap.EnsureMaps()
	uniqueSnap.Nodes[refKey(uniqueTarget)] = uniqueTarget
	uniqueSnap.OTelServiceMap = &otel.ServiceMap{
		Edges: []otel.ServiceEdge{{Source: "web", Target: "billing", Count: 50}},
	}
	_, uniqueNote := computeConfidence(uniqueSnap, &uniqueTarget)
	if !strings.Contains(uniqueNote, "traces") {
		t.Errorf("unambiguous name should keep real trace credit, got note %q", uniqueNote)
	}
}

func TestComputeBlastRadiusPercent_ZeroDenominator(t *testing.T) {
	pct := computeBlastRadiusPercent(nil, nil, nil, 0, nil)
	if pct != 0 {
		t.Errorf("expected 0, got %f", pct)
	}
}

func TestComputeBlastRadiusPercent_Amplification(t *testing.T) {
	// Build a snapshot where svc-a and svc-c both depend on svc-b (svc-b has 2 upstream callers)
	snap := &GraphSnapshot{}
	snap.EnsureMaps()

	svcA := models.ResourceRef{Kind: "Service", Namespace: "default", Name: "svc-a"}
	svcB := models.ResourceRef{Kind: "Service", Namespace: "default", Name: "svc-b"}
	svcC := models.ResourceRef{Kind: "Service", Namespace: "default", Name: "svc-c"}
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "dep-b"}

	snap.Nodes[refKey(svcA)] = svcA
	snap.Nodes[refKey(svcB)] = svcB
	snap.Nodes[refKey(svcC)] = svcC
	snap.Nodes[refKey(dep)] = dep

	// svc-a and svc-c both depend on svc-b (reverse: svc-b has 2 upstream callers)
	snap.Reverse[refKey(svcB)] = map[string]bool{refKey(svcA): true, refKey(svcC): true}

	snap.TotalWorkloads = 4

	// Broken service WITH upstream callers (amplified)
	impactsAmplified := []models.ServiceImpact{
		{Service: svcB, Classification: "broken", TotalEndpoints: 1, RemainingEndpoints: 0},
	}
	pctAmplified := computeBlastRadiusPercent(impactsAmplified, nil, nil, snap.TotalWorkloads, snap)

	// Same broken service but NO upstream callers (not amplified)
	snapNoUpstream := &GraphSnapshot{}
	snapNoUpstream.EnsureMaps()
	snapNoUpstream.TotalWorkloads = 4
	pctFlat := computeBlastRadiusPercent(impactsAmplified, nil, nil, snapNoUpstream.TotalWorkloads, snapNoUpstream)

	if pctAmplified <= pctFlat {
		t.Errorf("amplified blast radius (%.1f%%) should be > flat (%.1f%%)", pctAmplified, pctFlat)
	}

	// Verify specific values: flat = 1.0/4 * 100 = 25%, amplified = 1.15/4 * 100 = 28.75%
	if pctFlat != 25.0 {
		t.Errorf("flat blast radius expected 25.0%%, got %.1f%%", pctFlat)
	}
	expectedAmplified := (1.0 * 1.15 / 4.0) * 100.0
	if math.Abs(pctAmplified-expectedAmplified) > 0.01 {
		t.Errorf("amplified blast radius expected %.2f%%, got %.2f%%", expectedAmplified, pctAmplified)
	}
}

func TestFindMatchingPDBByLabels_MatchesLabels(t *testing.T) {
	minAvail := intstr.FromInt32(1)
	pdbs := []policyv1.PodDisruptionBudget{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "my-pdb", Namespace: "default"},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MinAvailable: &minAvail,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "payment"},
				},
			},
		},
	}
	svcRef := models.ResourceRef{Kind: "Service", Namespace: "default", Name: "totally-different-name"}
	podLabels := map[string]string{"app": "payment", "version": "v2"}

	pdb, found := findMatchingPDBByLabels(pdbs, svcRef, podLabels, 3)
	if !found {
		t.Fatal("expected PDB match via labels even though service name doesn't contain 'payment'")
	}
	if pdb.Name != "my-pdb" {
		t.Errorf("expected my-pdb, got %s", pdb.Name)
	}
}
