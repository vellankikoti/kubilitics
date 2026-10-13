package graph

import (
	"strings"
	"testing"
	"time"

	"github.com/kubilitics/kubilitics-backend/internal/models"
	"github.com/kubilitics/kubilitics-backend/internal/otel"
	policyv1 "k8s.io/api/policy/v1"
)

func TestComputeResilience_WellProtected(t *testing.T) {
	detail := computeResilience(ResilienceInput{
		Kind: "Deployment", Replicas: 5, HasHPA: true, HasPDB: true, HasController: true,
	})
	if detail.Score < 80 {
		t.Errorf("expected resilience >= 80 for well-protected deployment, got %d", detail.Score)
	}
}

func TestComputeResilience_SingleReplica(t *testing.T) {
	detail := computeResilience(ResilienceInput{
		Kind: "Deployment", Replicas: 1, HasHPA: false, HasPDB: false, HasController: true,
	})
	if detail.Score > 70 {
		t.Errorf("expected resilience <= 70 for single replica no HPA/PDB, got %d", detail.Score)
	}
}

func TestComputeResilience_NakedPod(t *testing.T) {
	detail := computeResilience(ResilienceInput{
		Kind: "Pod", Replicas: 0, HasHPA: false, HasPDB: false, HasController: false,
	})
	if detail.Score > 80 {
		t.Errorf("expected low resilience for naked pod, got %d", detail.Score)
	}
}

func TestComputeResilience_DaemonSet(t *testing.T) {
	detail := computeResilience(ResilienceInput{
		Kind: "DaemonSet", Replicas: 10, HasHPA: false, HasPDB: false, HasController: true,
	})
	if detail.Score < 70 {
		t.Errorf("expected DaemonSet resilience >= 70, got %d", detail.Score)
	}
}

func TestComputeExposure_IngressExposed(t *testing.T) {
	detail := computeExposure(ExposureInput{
		IsIngressExposed: true, ConsumerCount: 3, CrossNsCount: 2,
		TraceDataAvailable: true, IsCriticalSystem: false,
	})
	if detail.Score < 50 {
		t.Errorf("expected high exposure for ingress+consumers, got %d", detail.Score)
	}
}

func TestComputeExposure_CriticalSystem(t *testing.T) {
	detail := computeExposure(ExposureInput{
		IsIngressExposed: false, ConsumerCount: 0, CrossNsCount: 1,
		TraceDataAvailable: false, IsCriticalSystem: true,
	})
	if detail.Score < 80 {
		t.Errorf("expected critical system exposure >= 80, got %d", detail.Score)
	}
}

func TestComputeRecovery_StatefulSet(t *testing.T) {
	detail := computeRecovery(RecoveryInput{
		Kind: "StatefulSet", Replicas: 3, HasController: true, HasPVC: true, IsControlPlane: false,
	})
	if detail.Score > 70 {
		t.Errorf("expected StatefulSet recovery < 70, got %d", detail.Score)
	}
}

func TestComputeOverallCriticality_LowImpact(t *testing.T) {
	scores := models.SubScores{
		Resilience: models.SubScoreDetail{Score: 50},
		Exposure:   models.SubScoreDetail{Score: 5},
		Recovery:   models.SubScoreDetail{Score: 90},
		Impact:     models.SubScoreDetail{Score: 0},
	}
	crit := computeOverallCriticality(scores)
	if crit > 25 {
		t.Errorf("expected low criticality for zero-impact workload, got %.1f", crit)
	}
}

func TestComputeOverallCriticality_HighImpact(t *testing.T) {
	scores := models.SubScores{
		Resilience: models.SubScoreDetail{Score: 10},
		Exposure:   models.SubScoreDetail{Score: 80},
		Recovery:   models.SubScoreDetail{Score: 20},
		Impact:     models.SubScoreDetail{Score: 80},
	}
	crit := computeOverallCriticality(scores)
	if crit < 60 {
		t.Errorf("expected high criticality for high-impact exposed workload, got %.1f", crit)
	}
}

func TestCriticalityLevelV2(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{80, "critical"}, {55, "high"}, {30, "medium"}, {10, "low"},
	}
	for _, tt := range tests {
		got := criticalityLevelV2(tt.score)
		if got != tt.want {
			t.Errorf("criticalityLevelV2(%.0f) = %s, want %s", tt.score, got, tt.want)
		}
	}
}

func TestComputeResilience_ZeroReplicas(t *testing.T) {
	zero := computeResilience(ResilienceInput{
		Kind: "Deployment", Replicas: 0, HasHPA: false, HasPDB: false, HasController: true,
	})
	one := computeResilience(ResilienceInput{
		Kind: "Deployment", Replicas: 1, HasHPA: false, HasPDB: false, HasController: true,
	})
	if zero.Score >= one.Score {
		t.Errorf("expected 0 replicas (%d) to score lower than 1 replica (%d)", zero.Score, one.Score)
	}
}

func TestComputeExposure_CrossNamespace(t *testing.T) {
	detail := computeExposure(ExposureInput{
		IsIngressExposed: false, ConsumerCount: 0, CrossNsCount: 2,
		K8sFanIn: 0, TraceDataAvailable: false, IsCriticalSystem: false,
	})
	if detail.Score < 10 {
		t.Errorf("expected CrossNsCount=2 to give exposure >= 10, got %d", detail.Score)
	}
}

func TestComputeExposure_TrafficWeighted(t *testing.T) {
	// High-traffic service: 10000 calls out of 50000 total = 20% of cluster traffic
	highTraffic := computeExposure(ExposureInput{
		IsIngressExposed:   false,
		ConsumerCount:      2,
		TraceDataAvailable: true,
		TotalCallsToTarget: 10000,
		ClusterTotalCalls:  50000,
	})

	// Low-traffic service: same consumer count but only 100 calls
	lowTraffic := computeExposure(ExposureInput{
		IsIngressExposed:   false,
		ConsumerCount:      2,
		TraceDataAvailable: true,
		TotalCallsToTarget: 100,
		ClusterTotalCalls:  50000,
	})

	if highTraffic.Score <= lowTraffic.Score {
		t.Errorf("high-traffic service (%d) should score higher than low-traffic (%d)",
			highTraffic.Score, lowTraffic.Score)
	}
}

func TestComputeExposure_TrafficWeighted_NoCallData(t *testing.T) {
	// Trace available but no call counts — should fall back to consumer count
	detail := computeExposure(ExposureInput{
		IsIngressExposed:   false,
		ConsumerCount:      3,
		TraceDataAvailable: true,
		TotalCallsToTarget: 0,
		ClusterTotalCalls:  0,
	})
	if detail.Score < 20 {
		t.Errorf("fallback to consumer count should give decent score, got %d", detail.Score)
	}
}

func TestComputeConfidence_FullData(t *testing.T) {
	snap := buildSnapshotScenario1()
	snap.PDBs = []policyv1.PodDisruptionBudget{{}} // just need non-empty
	snap.NodeHasHPA["Deployment/default/app"] = true
	snap.TotalWorkloads = 15

	score, note := computeConfidence(snap, nil)
	// endpoints(30) + PDBs(15) + HPAs(10) + large cluster(15) = 70 (no traces)
	if score < 60 {
		t.Errorf("full data minus traces should score >= 60, got %d", score)
	}
	if note == "" {
		t.Error("note should not be empty")
	}
}

func TestComputeConfidence_GraphOnly(t *testing.T) {
	snap := &GraphSnapshot{}
	snap.EnsureMaps()

	score, note := computeConfidence(snap, nil)
	if score != 0 {
		t.Errorf("empty snapshot should have 0 confidence, got %d", score)
	}
	if note != "Graph topology only — no runtime data available" {
		t.Errorf("unexpected note for empty snapshot: %s", note)
	}
}

func TestComputeExposure_ErrorRate(t *testing.T) {
	// Service with high dependency error rate
	highError := computeExposure(ExposureInput{
		IsIngressExposed:    false,
		ConsumerCount:       2,
		TraceDataAvailable:  true,
		TotalCallsToTarget:  1000,
		ClusterTotalCalls:   5000,
		DependencyErrorRate: 0.5, // 50% errors
	})

	// Same service with no errors
	noError := computeExposure(ExposureInput{
		IsIngressExposed:    false,
		ConsumerCount:       2,
		TraceDataAvailable:  true,
		TotalCallsToTarget:  1000,
		ClusterTotalCalls:   5000,
		DependencyErrorRate: 0.0,
	})

	if highError.Score <= noError.Score {
		t.Errorf("50%% error rate service (%d) should score higher exposure than 0%% (%d)",
			highError.Score, noError.Score)
	}
}

// TestComputeConfidence_IsPerTargetNotClusterGlobal is the Phase 2 audit
// regression test (Checkpoint A suspected bug, confirmed here). Two targets
// in the SAME snapshot: "well-covered" has a PDB, an HPA, and is traced by
// OTel; "bare" has none of that and is never mentioned in any trace edge.
// Before classifyImpact threaded the target into computeConfidence, the
// function only read cluster-wide snapshot fields (snap.PDBs non-empty,
// snap.NodeHasHPA true for ANY node, snap.OTelServiceMap non-empty at all),
// so both targets got the exact same confidence score and "Based on: ..."
// note — an SRE querying blast radius for the untraced, unprotected "bare"
// resource during an incident would see the same false reassurance as for
// the well-instrumented one.
func TestComputeConfidence_IsPerTargetNotClusterGlobal(t *testing.T) {
	wellCovered := models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "well-covered"}
	bare := models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "bare"}
	wcKey := refKey(wellCovered)
	bareKey := refKey(bare)

	snap := &GraphSnapshot{}
	snap.EnsureMaps()
	snap.Nodes[wcKey] = wellCovered
	snap.Nodes[bareKey] = bare
	snap.NodeHasPDB[wcKey] = true
	snap.NodeHasHPA[wcKey] = true
	snap.PDBs = []policyv1.PodDisruptionBudget{{}}
	snap.OTelServiceMap = &otel.ServiceMap{
		Edges: []otel.ServiceEdge{{Source: "ingress", Target: "well-covered", Count: 500}},
	}
	snap.BuiltAt = time.Now().UnixMilli()
	snap.TotalWorkloads = 2

	wcScore, wcNote := computeConfidence(snap, &wellCovered)
	bareScore, bareNote := computeConfidence(snap, &bare)

	if wcScore <= bareScore {
		t.Errorf("well-covered target (PDB+HPA+traced) should score higher confidence than bare target with none of that, got well-covered=%d bare=%d", wcScore, bareScore)
	}
	if wcNote == bareNote {
		t.Errorf("well-covered and bare targets should not share an identical confidence note, both got: %q", wcNote)
	}
}

func TestComputeConfidence_StaleTraces(t *testing.T) {
	// Fresh snapshot (just built)
	freshSnap := buildSnapshotScenario1()
	freshSnap.OTelServiceMap = &otel.ServiceMap{
		Edges: []otel.ServiceEdge{{Source: "a", Target: "b", Count: 10}},
	}
	freshSnap.BuiltAt = time.Now().UnixMilli() // fresh
	freshScore, _ := computeConfidence(freshSnap, nil)

	// Stale snapshot (3 hours old)
	staleSnap := buildSnapshotScenario1()
	staleSnap.OTelServiceMap = &otel.ServiceMap{
		Edges: []otel.ServiceEdge{{Source: "a", Target: "b", Count: 10}},
	}
	staleSnap.BuiltAt = time.Now().Add(-3 * time.Hour).UnixMilli() // 3 hours old
	staleScore, staleNote := computeConfidence(staleSnap, nil)

	if staleScore >= freshScore {
		t.Errorf("stale trace data (%d) should give lower confidence than fresh (%d)",
			staleScore, freshScore)
	}
	if !strings.Contains(staleNote, "old") {
		t.Errorf("stale note should mention age, got: %s", staleNote)
	}
}
