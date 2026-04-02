package graph

import (
	"testing"

	"github.com/kubilitics/kubilitics-backend/internal/models"
)

func TestComputeCriticalityScore_LowIsolatedLeaf(t *testing.T) {
	p := scoringParams{
		pageRank:         0.01,
		fanIn:            0,
		crossNsCount:     0,
		isDataStore:      false,
		isIngressExposed: false,
		isSPOF:           false,
		hasHPA:           true,
		hasPDB:           true,
	}
	score := computeCriticalityScore(p)
	if score < 0 || score > 10 {
		t.Errorf("expected low score 0-10 for isolated leaf, got %.2f", score)
	}
}

func TestComputeCriticalityScore_CriticalHighRisk(t *testing.T) {
	p := scoringParams{
		pageRank:         0.8,
		fanIn:            12,
		crossNsCount:     4,
		isDataStore:      false,
		isIngressExposed: true,
		isSPOF:           true,
		hasHPA:           false,
		hasPDB:           false,
	}
	score := computeCriticalityScore(p)
	if score < 75 || score > 100 {
		t.Errorf("expected critical score 75-100, got %.2f", score)
	}
}

func TestComputeCriticalityScore_MediumModerate(t *testing.T) {
	// pageRank=0.5 → 15, fanIn=5 → 15, crossNsCount=0, has HPA+PDB → 30 total
	p := scoringParams{
		pageRank:         0.5,
		fanIn:            5,
		crossNsCount:     0,
		isDataStore:      false,
		isIngressExposed: false,
		isSPOF:           false,
		hasHPA:           true,
		hasPDB:           true,
	}
	score := computeCriticalityScore(p)
	if score < 25 || score > 50 {
		t.Errorf("expected medium score 25-50, got %.2f", score)
	}
}

func TestComputeCriticalityScore_Cap100(t *testing.T) {
	p := scoringParams{
		pageRank:         10.0, // would contribute 300 without cap
		fanIn:            100,
		crossNsCount:     100,
		isDataStore:      true,
		isIngressExposed: true,
		isSPOF:           true,
		hasHPA:           false,
		hasPDB:           false,
	}
	score := computeCriticalityScore(p)
	if score != 100.0 {
		t.Errorf("expected score capped at 100, got %.2f", score)
	}
}

func TestComputeCriticalityScore_CrossNsSkippedWhenOne(t *testing.T) {
	// crossNsCount == 1 should NOT contribute cross-ns points
	withOne := scoringParams{
		pageRank:     0.0,
		fanIn:        0,
		crossNsCount: 1,
		hasHPA:       true,
		hasPDB:       true,
	}
	withZero := scoringParams{
		pageRank:     0.0,
		fanIn:        0,
		crossNsCount: 0,
		hasHPA:       true,
		hasPDB:       true,
	}
	if computeCriticalityScore(withOne) != computeCriticalityScore(withZero) {
		t.Errorf("crossNsCount=1 should not contribute to score")
	}
}

func TestSimplePageRank_SingleNode(t *testing.T) {
	nodes := map[string]models.ResourceRef{"a": {Kind: "Deployment", Name: "a", Namespace: "default"}}
	forward := map[string]map[string]bool{}
	reverse := map[string]map[string]bool{}
	ranks := simplePageRank(nodes, forward, reverse)
	if ranks["a"] != 1.0 {
		t.Errorf("single-node graph: expected rank 1.0, got %.4f", ranks["a"])
	}
}

func TestSimplePageRank_TwoNodes(t *testing.T) {
	// a -> b; b should accumulate more rank
	nodes := map[string]models.ResourceRef{
		"a": {Kind: "Deployment", Name: "a", Namespace: "default"},
		"b": {Kind: "Service", Name: "b", Namespace: "default"},
	}
	forward := map[string]map[string]bool{"a": {"b": true}}
	reverse := map[string]map[string]bool{"b": {"a": true}}
	ranks := simplePageRank(nodes, forward, reverse)
	if ranks["b"] < ranks["a"] {
		t.Errorf("node b (target) should have higher rank than a (source), a=%.4f b=%.4f", ranks["a"], ranks["b"])
	}
}

func TestSimplePageRank_NormalizesToOne(t *testing.T) {
	nodes := map[string]models.ResourceRef{
		"x": {Kind: "Deployment", Name: "x", Namespace: "default"},
		"y": {Kind: "Service", Name: "y", Namespace: "default"},
		"z": {Kind: "ConfigMap", Name: "z", Namespace: "default"},
	}
	// x -> y -> z chain
	forward := map[string]map[string]bool{
		"x": {"y": true},
		"y": {"z": true},
	}
	reverse := map[string]map[string]bool{
		"y": {"x": true},
		"z": {"y": true},
	}
	ranks := simplePageRank(nodes, forward, reverse)
	maxRank := 0.0
	for _, v := range ranks {
		if v > maxRank {
			maxRank = v
		}
	}
	if maxRank < 0.999 || maxRank > 1.001 {
		t.Errorf("max rank should be normalized to ~1.0, got %.4f", maxRank)
	}
}

func TestSimplePageRank_EmptyGraph(t *testing.T) {
	ranks := simplePageRank(
		map[string]models.ResourceRef{},
		map[string]map[string]bool{},
		map[string]map[string]bool{},
	)
	if len(ranks) != 0 {
		t.Errorf("empty graph should return empty map")
	}
}

func TestComputeBaseScore_EqualsComputeCriticalityScore(t *testing.T) {
	// computeBaseScore and computeCriticalityScore should return identical results
	p := scoringParams{
		pageRank:         0.5,
		fanIn:            4,
		crossNsCount:     3,
		isDataStore:      true,
		isIngressExposed: false,
		isSPOF:           false,
		hasHPA:           false,
		hasPDB:           true,
	}
	base := computeBaseScore(p)
	crit := computeCriticalityScore(p)
	if base != crit {
		t.Errorf("computeBaseScore (%.2f) should equal computeCriticalityScore (%.2f)", base, crit)
	}
}

func TestApplyFailureMode_PodCrashAttenuates(t *testing.T) {
	baseScore := 46.0

	// 3-replica pod-crash: 46 * (1/3) = ~15.3 -> LOW
	adjusted := applyFailureMode(baseScore, FailureModePodCrash, 3)
	if adjusted >= 20.0 {
		t.Errorf("pod-crash with 3 replicas: expected < 20 (LOW), got %.2f", adjusted)
	}

	// 5-replica pod-crash: 46 * (1/5) = 9.2
	adjusted5 := applyFailureMode(baseScore, FailureModePodCrash, 5)
	if adjusted5 >= 20.0 {
		t.Errorf("pod-crash with 5 replicas: expected < 20, got %.2f", adjusted5)
	}
	if adjusted5 >= adjusted {
		t.Errorf("5-replica pod-crash (%.2f) should be less than 3-replica (%.2f)", adjusted5, adjusted)
	}
}

func TestApplyFailureMode_WorkloadDeletionNoAttenuation(t *testing.T) {
	baseScore := 46.0
	adjusted := applyFailureMode(baseScore, FailureModeWorkloadDeletion, 3)
	if adjusted != baseScore {
		t.Errorf("workload-deletion should not attenuate: expected %.2f, got %.2f", baseScore, adjusted)
	}
}
