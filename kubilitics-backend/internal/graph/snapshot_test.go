package graph

import (
	"testing"
	"time"

	"github.com/kubilitics/kubilitics-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefKey(t *testing.T) {
	ref := models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "nginx"}
	assert.Equal(t, "Deployment/default/nginx", refKey(ref))

	// Cluster-scoped resource (empty namespace)
	ref2 := models.ResourceRef{Kind: "ClusterRole", Namespace: "", Name: "admin"}
	assert.Equal(t, "ClusterRole//admin", refKey(ref2))
}

func TestBfsWalk(t *testing.T) {
	// A -> B -> C
	adj := map[string]map[string]bool{
		"A": {"B": true},
		"B": {"C": true},
		"C": {},
	}

	reached := bfsWalk(adj, "A")
	assert.True(t, reached["B"])
	assert.True(t, reached["C"])
	assert.False(t, reached["A"], "start node should not be in result")
	assert.Len(t, reached, 2)

	// Walk from C should yield nothing
	reached2 := bfsWalk(adj, "C")
	assert.Len(t, reached2, 0)
}

func TestBfsWalkWithDepth(t *testing.T) {
	// A -> B -> C -> D
	adj := map[string]map[string]bool{
		"A": {"B": true},
		"B": {"C": true},
		"C": {"D": true},
	}

	depths := bfsWalkWithDepth(adj, "A")
	assert.Equal(t, 1, depths["B"])
	assert.Equal(t, 2, depths["C"])
	assert.Equal(t, 3, depths["D"])
	_, hasA := depths["A"]
	assert.False(t, hasA)
}

func TestShortestPath(t *testing.T) {
	adj := map[string]map[string]bool{
		"A": {"B": true, "C": true},
		"B": {"D": true},
		"C": {"D": true},
	}

	path := shortestPath(adj, "A", "D")
	require.NotNil(t, path)
	assert.Equal(t, "A", path[0])
	assert.Equal(t, "D", path[len(path)-1])
	assert.Len(t, path, 3) // A -> B/C -> D

	// No path
	path2 := shortestPath(adj, "D", "A")
	assert.Nil(t, path2)

	// Same node
	path3 := shortestPath(adj, "A", "A")
	assert.Equal(t, []string{"A"}, path3)
}

func TestCriticalityLevel(t *testing.T) {
	// Recalibrated thresholds: LOW < 20, MEDIUM 20-45, HIGH 45-70, CRITICAL > 70
	assert.Equal(t, "critical", criticalityLevel(100))
	assert.Equal(t, "critical", criticalityLevel(71))
	assert.Equal(t, "high", criticalityLevel(70))
	assert.Equal(t, "high", criticalityLevel(45))
	assert.Equal(t, "medium", criticalityLevel(44.9))
	assert.Equal(t, "medium", criticalityLevel(20))
	assert.Equal(t, "low", criticalityLevel(19.9))
	assert.Equal(t, "low", criticalityLevel(0))
}

// buildTestSnapshot creates a small graph: Service -> Deployment -> ReplicaSet -> Pod
func buildTestSnapshot() *GraphSnapshot {
	svc := models.ResourceRef{Kind: "Service", Namespace: "default", Name: "web-svc"}
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "web"}
	rs := models.ResourceRef{Kind: "ReplicaSet", Namespace: "default", Name: "web-abc"}
	pod := models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "web-abc-xyz"}

	svcKey := refKey(svc)
	depKey := refKey(dep)
	rsKey := refKey(rs)
	podKey := refKey(pod)

	edges := []models.BlastDependencyEdge{
		{Source: svc, Target: dep, Type: "selects"},
		{Source: dep, Target: rs, Type: "owns"},
		{Source: rs, Target: pod, Type: "owns"},
	}

	return &GraphSnapshot{
		Nodes: map[string]models.ResourceRef{
			svcKey: svc,
			depKey: dep,
			rsKey:  rs,
			podKey: pod,
		},
		// Forward: what does X depend on (X -> Y means X depends on Y)
		Forward: map[string]map[string]bool{
			svcKey: {depKey: true},
			depKey: {rsKey: true},
			rsKey:  {podKey: true},
		},
		// Reverse: what depends on X (if X fails, who is affected)
		Reverse: map[string]map[string]bool{
			depKey: {svcKey: true},
			rsKey:  {depKey: true},
			podKey: {rsKey: true},
		},
		Edges: edges,
		NodeScores: map[string]float64{
			depKey: 80.0,
			svcKey: 40.0,
			rsKey:  30.0,
			podKey: 10.0,
		},
		NodeRisks: map[string][]models.RiskIndicator{
			depKey: {
				{Severity: "warning", Title: "Single replica", Detail: "Only 1 replica configured"},
			},
		},
		NodeReplicas: map[string]int{
			depKey: 1,
			rsKey:  1,
			podKey: 1,
		},
		NodeHasHPA:     map[string]bool{},
		NodeHasPDB:     map[string]bool{},
		NodeIngress:    map[string][]string{},
		TotalWorkloads: 4,
		BuiltAt:        time.Now().UnixMilli(),
		BuildDuration:  50 * time.Millisecond,
		Namespaces:     map[string]bool{"default": true},
	}
}

func TestComputeBlastRadius_SimpleChain(t *testing.T) {
	snap := buildTestSnapshot()
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "web"}

	result, err := snap.ComputeBlastRadius(dep)
	require.NoError(t, err)
	require.NotNil(t, result)

	// The deployment is depended on by the Service (via reverse).
	// Reverse graph: dep <- svc. So affected = {svc}, totalAffected = 1
	assert.Equal(t, 1, result.TotalAffected)
	assert.Equal(t, "Service", result.Waves[0].Resources[0].Kind)
	assert.Equal(t, 1, result.Waves[0].Depth)
	assert.Equal(t, "direct", result.Waves[0].Resources[0].Impact)

	// Fan-in: what depends on deployment = Service (1)
	assert.Equal(t, 1, result.FanIn)
	// Fan-out: what deployment depends on = ReplicaSet (1)
	assert.Equal(t, 1, result.FanOut)

	// Criticality
	assert.Equal(t, 80.0, result.CriticalityScore)
	assert.Equal(t, "critical", result.CriticalityLevel)

	// SPOF: 1 replica, no HPA, has dependents
	assert.True(t, result.IsSPOF)

	// Blast radius percent: 1 affected / 3 reachable subgraph nodes = 33.33%
	// (reachable subgraph from Deployment: Service via reverse, ReplicaSet + Pod via forward = 3)
	assert.InDelta(t, 33.33, result.BlastRadiusPercent, 0.1)

	// Graph stats
	assert.Equal(t, 4, result.GraphNodeCount)
	assert.Equal(t, 3, result.GraphEdgeCount)
}

func TestComputeBlastRadius_NotFound(t *testing.T) {
	snap := buildTestSnapshot()
	_, err := snap.ComputeBlastRadius(models.ResourceRef{Kind: "ConfigMap", Namespace: "default", Name: "missing"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestComputeBlastRadius_Pod_NoAffected(t *testing.T) {
	snap := buildTestSnapshot()
	pod := models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "web-abc-xyz"}

	result, err := snap.ComputeBlastRadius(pod)
	require.NoError(t, err)

	// Pod has reverse deps: ReplicaSet -> Deployment -> Service (3 total)
	assert.Equal(t, 3, result.TotalAffected)
}

func TestGetSummary(t *testing.T) {
	snap := buildTestSnapshot()

	summary := snap.GetSummary(2)
	require.Len(t, summary, 2)

	// Top score should be Deployment at 80
	assert.Equal(t, "Deployment", summary[0].Resource.Kind)
	assert.Equal(t, 80.0, summary[0].CriticalityScore)
	assert.Equal(t, "critical", summary[0].CriticalityLevel)

	// Second should be Service at 40
	assert.Equal(t, "Service", summary[1].Resource.Kind)
	assert.Equal(t, 40.0, summary[1].CriticalityScore)
}

func TestGetSummary_NoLimit(t *testing.T) {
	snap := buildTestSnapshot()
	summary := snap.GetSummary(0)
	assert.Len(t, summary, 4) // all nodes have scores
}

func TestStatus(t *testing.T) {
	snap := buildTestSnapshot()
	status := snap.Status()

	assert.True(t, status.Ready)
	assert.Equal(t, 4, status.NodeCount)
	assert.Equal(t, 3, status.EdgeCount)
	assert.Equal(t, 1, status.NamespaceCount)
	assert.True(t, status.StalenessMs >= 0)
}

func TestStatus_Empty(t *testing.T) {
	snap := &GraphSnapshot{
		Nodes:      map[string]models.ResourceRef{},
		Namespaces: map[string]bool{},
	}
	status := snap.Status()
	assert.False(t, status.Ready)
	assert.Equal(t, 0, status.NodeCount)
}

func TestBuildFailurePath(t *testing.T) {
	snap := buildTestSnapshot()

	// Pod failure propagates: Pod -> RS -> Dep -> Svc (in reverse graph)
	podKey := refKey(models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "web-abc-xyz"})
	svcKey := refKey(models.ResourceRef{Kind: "Service", Namespace: "default", Name: "web-svc"})

	hops := snap.buildFailurePath(podKey, svcKey)
	require.Len(t, hops, 3) // Pod->RS, RS->Dep, Dep->Svc
	assert.Equal(t, "Pod", hops[0].From.Kind)
	assert.Equal(t, "ReplicaSet", hops[0].To.Kind)
	assert.Equal(t, "Service", hops[2].To.Kind)
}

// --- T2: Failure Mode Tests ---

// buildThreeReplicaSnapshot creates a graph with a 3-replica Deployment
// to test failure-mode scoring. Graph: Ingress -> Service -> Deployment -> ConfigMap
func buildThreeReplicaSnapshot() *GraphSnapshot {
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "prod", Name: "api"}
	svc := models.ResourceRef{Kind: "Service", Namespace: "prod", Name: "api-svc"}
	ing := models.ResourceRef{Kind: "Ingress", Namespace: "prod", Name: "api-ing"}
	cm := models.ResourceRef{Kind: "ConfigMap", Namespace: "prod", Name: "api-config"}

	depKey := refKey(dep)
	svcKey := refKey(svc)
	ingKey := refKey(ing)
	cmKey := refKey(cm)

	edges := []models.BlastDependencyEdge{
		{Source: ing, Target: svc, Type: "routes"},
		{Source: svc, Target: dep, Type: "selects"},
		{Source: dep, Target: cm, Type: "mounts"},
	}

	// Base score for the deployment:
	// pageRank ~0.5 -> 15, fanIn=1 (svc) -> 3, crossNs=0, no datastore,
	// ingress not directly on dep, not SPOF (3 replicas), no HPA (+5), no PDB (+5)
	// = ~28 base score
	baseScore := 28.0

	return &GraphSnapshot{
		Nodes: map[string]models.ResourceRef{
			depKey: dep,
			svcKey: svc,
			ingKey: ing,
			cmKey:  cm,
		},
		Forward: map[string]map[string]bool{
			ingKey: {svcKey: true},
			svcKey: {depKey: true},
			depKey: {cmKey: true},
		},
		Reverse: map[string]map[string]bool{
			svcKey: {ingKey: true},
			depKey: {svcKey: true},
			cmKey:  {depKey: true},
		},
		Edges: edges,
		NodeScores: map[string]float64{
			depKey: baseScore,
			svcKey: 15.0,
			ingKey: 5.0,
			cmKey:  10.0,
		},
		NodeRisks:    map[string][]models.RiskIndicator{},
		NodeReplicas: map[string]int{depKey: 3, svcKey: 0, ingKey: 0, cmKey: 0},
		NodeHasHPA:   map[string]bool{},
		NodeHasPDB:   map[string]bool{},
		NodeIngress:  map[string][]string{},
		TotalWorkloads: 2,
		BuiltAt:        time.Now().UnixMilli(),
		BuildDuration:  10 * time.Millisecond,
		Namespaces:     map[string]bool{"prod": true},
	}
}

func TestComputeBlastRadius_PodCrash_ThreeReplicas_ScoresLow(t *testing.T) {
	snap := buildThreeReplicaSnapshot()
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "prod", Name: "api"}

	result, err := snap.ComputeBlastRadiusWithMode(dep, FailureModePodCrash)
	require.NoError(t, err)

	// With 3 replicas, pod-crash score = baseScore * (1/3) = ~9.3
	assert.Less(t, result.CriticalityScore, 20.0,
		"pod-crash in a 3-replica Deployment should score LOW (< 20), got %.2f", result.CriticalityScore)
	assert.Equal(t, "low", result.CriticalityLevel)
	assert.Equal(t, FailureModePodCrash, result.FailureMode)
}

func TestComputeBlastRadius_WorkloadDeletion_ThreeReplicas_ScoresMediumOrHigher(t *testing.T) {
	snap := buildThreeReplicaSnapshot()
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "prod", Name: "api"}

	result, err := snap.ComputeBlastRadiusWithMode(dep, FailureModeWorkloadDeletion)
	require.NoError(t, err)

	// Workload deletion: full base score (no replica attenuation)
	assert.GreaterOrEqual(t, result.CriticalityScore, 20.0,
		"workload-deletion should score MEDIUM or higher, got %.2f", result.CriticalityScore)
	assert.Equal(t, FailureModeWorkloadDeletion, result.FailureMode)
}

func TestComputeBlastRadius_DefaultMode_IsWorkloadDeletion(t *testing.T) {
	snap := buildThreeReplicaSnapshot()
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "prod", Name: "api"}

	// ComputeBlastRadius (no mode) should default to workload-deletion
	result, err := snap.ComputeBlastRadius(dep)
	require.NoError(t, err)

	assert.Equal(t, FailureModeWorkloadDeletion, result.FailureMode)
}

func TestComputeBlastRadius_InvalidMode_DefaultsToWorkloadDeletion(t *testing.T) {
	snap := buildThreeReplicaSnapshot()
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "prod", Name: "api"}

	result, err := snap.ComputeBlastRadiusWithMode(dep, "invalid-mode")
	require.NoError(t, err)

	assert.Equal(t, FailureModeWorkloadDeletion, result.FailureMode)
}

func TestComputeBlastRadius_NamespaceDeletion_ScoresCritical(t *testing.T) {
	snap := buildThreeReplicaSnapshot()
	// Use any resource in "prod" namespace as the target
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "prod", Name: "api"}

	result, err := snap.ComputeBlastRadiusWithMode(dep, FailureModeNamespaceDeletion)
	require.NoError(t, err)

	// Namespace deletion sums all workload scores in the namespace.
	// With Service (15) + Deployment (28) = 43, this should be capped or at least MEDIUM+
	assert.Equal(t, FailureModeNamespaceDeletion, result.FailureMode)
	// The aggregate should be at least the individual workload score
	assert.GreaterOrEqual(t, result.CriticalityScore, 20.0,
		"namespace-deletion should aggregate workload scores")
}

func TestComputeBlastRadius_PodCrash_SingleReplica_FullImpact(t *testing.T) {
	snap := buildTestSnapshot() // uses 1-replica Deployment
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "web"}

	result, err := snap.ComputeBlastRadiusWithMode(dep, FailureModePodCrash)
	require.NoError(t, err)

	// With 1 replica, pod-crash = base score * (1/1) = full impact
	// Same as workload-deletion
	resultWD, _ := snap.ComputeBlastRadiusWithMode(dep, FailureModeWorkloadDeletion)
	assert.InDelta(t, resultWD.CriticalityScore, result.CriticalityScore, 0.01,
		"pod-crash with 1 replica should equal workload-deletion score")
}

// --- T4: Blast Percent Denominator Tests ---

func TestBlastPercent_UsesReachableSubgraph(t *testing.T) {
	snap := buildTestSnapshot()
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "web"}

	result, err := snap.ComputeBlastRadius(dep)
	require.NoError(t, err)

	// The deployment's reachable subgraph (forward + reverse BFS):
	// Forward: RS, Pod. Reverse: Service. Total = 3
	// Affected (reverse BFS only): Service = 1
	// Blast % = 1/3 * 100 = 33.33%
	assert.InDelta(t, 33.33, result.BlastRadiusPercent, 0.1,
		"blast %% should use reachable subgraph as denominator, not total workloads")
}

func TestReachableSubgraphSize(t *testing.T) {
	// A -> B -> C, D (isolated)
	forward := map[string]map[string]bool{
		"A": {"B": true},
		"B": {"C": true},
	}
	reverse := map[string]map[string]bool{
		"B": {"A": true},
		"C": {"B": true},
	}

	// From A: forward reaches B, C; reverse reaches nothing. Subgraph = {B, C} = 2
	assert.Equal(t, 2, reachableSubgraphSize(forward, reverse, "A"))

	// From B: forward reaches C; reverse reaches A. Subgraph = {A, C} = 2
	assert.Equal(t, 2, reachableSubgraphSize(forward, reverse, "B"))

	// From C: forward reaches nothing; reverse reaches B, A. Subgraph = {A, B} = 2
	assert.Equal(t, 2, reachableSubgraphSize(forward, reverse, "C"))

	// From D (isolated, not in any adjacency): subgraph = 0
	assert.Equal(t, 0, reachableSubgraphSize(forward, reverse, "D"))
}

// --- T2: Scoring Function Tests ---

func TestApplyFailureMode_PodCrash(t *testing.T) {
	// 3 replicas: score should be 1/3 of base
	result := applyFailureMode(60.0, FailureModePodCrash, 3)
	assert.InDelta(t, 20.0, result, 0.01)

	// 1 replica: full score
	result = applyFailureMode(60.0, FailureModePodCrash, 1)
	assert.InDelta(t, 60.0, result, 0.01)

	// 0 replicas: full score (edge case)
	result = applyFailureMode(60.0, FailureModePodCrash, 0)
	assert.InDelta(t, 60.0, result, 0.01)
}

func TestApplyFailureMode_WorkloadDeletion(t *testing.T) {
	// Always full score regardless of replicas
	result := applyFailureMode(60.0, FailureModeWorkloadDeletion, 3)
	assert.InDelta(t, 60.0, result, 0.01)

	result = applyFailureMode(60.0, FailureModeWorkloadDeletion, 1)
	assert.InDelta(t, 60.0, result, 0.01)
}

func TestApplyFailureMode_UnknownDefaultsToFull(t *testing.T) {
	result := applyFailureMode(60.0, "unknown", 3)
	assert.InDelta(t, 60.0, result, 0.01)
}

func TestValidFailureMode(t *testing.T) {
	assert.True(t, ValidFailureMode(FailureModePodCrash))
	assert.True(t, ValidFailureMode(FailureModeWorkloadDeletion))
	assert.True(t, ValidFailureMode(FailureModeNamespaceDeletion))
	assert.False(t, ValidFailureMode("invalid"))
	assert.False(t, ValidFailureMode(""))
}

// --- T5: Remediation Integration Tests ---

func TestComputeBlastRadius_IncludesRemediations(t *testing.T) {
	snap := buildTestSnapshot() // 1-replica Deployment, no HPA, no PDB
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "web"}

	result, err := snap.ComputeBlastRadius(dep)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Remediations, "1-replica deployment without HPA/PDB should have remediations")

	// Check specific remediation types
	types := make(map[string]bool)
	for _, r := range result.Remediations {
		types[r.Type] = true
	}
	assert.True(t, types["increase-replicas"] || types["resolve-critical-spof"],
		"should recommend increasing replicas or resolving SPOF")
	assert.True(t, types["add-pdb"], "should recommend adding PDB")
	assert.True(t, types["add-hpa"], "should recommend adding HPA")
}

func TestComputeBlastRadius_RemediationsNonNil(t *testing.T) {
	// Even resources with no remediations should return empty slice, not nil
	snap := buildThreeReplicaSnapshot()
	// The ConfigMap has 0 replicas, so PDB/HPA won't trigger
	cm := models.ResourceRef{Kind: "ConfigMap", Namespace: "prod", Name: "api-config"}

	result, err := snap.ComputeBlastRadius(cm)
	require.NoError(t, err)

	assert.NotNil(t, result.Remediations, "remediations should never be nil")
}

func TestComputeBlastRadius_FailureModeField(t *testing.T) {
	snap := buildThreeReplicaSnapshot()
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "prod", Name: "api"}

	modes := []string{FailureModePodCrash, FailureModeWorkloadDeletion}
	for _, mode := range modes {
		result, err := snap.ComputeBlastRadiusWithMode(dep, mode)
		require.NoError(t, err)
		assert.Equal(t, mode, result.FailureMode,
			"result should reflect the requested failure mode")
	}
}
