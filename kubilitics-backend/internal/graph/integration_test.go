package graph

import (
	"testing"
	"time"

	"github.com/kubilitics/kubilitics-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

// buildSnapshotScenario1 creates a cluster with:
//   - 1 Deployment "app" with 3 replicas
//   - 3 Pods owned by the Deployment
//   - 1 Service "app-svc" with 3 ready endpoints pointing to the 3 pods
func buildSnapshotScenario1() *GraphSnapshot {
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "app"}
	pod1 := models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "pod-1"}
	pod2 := models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "pod-2"}
	pod3 := models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "pod-3"}
	svc := models.ResourceRef{Kind: "Service", Namespace: "default", Name: "app-svc"}

	depKey := refKey(dep)
	pod1Key := refKey(pod1)
	pod2Key := refKey(pod2)
	pod3Key := refKey(pod3)
	svcKey := refKey(svc)

	snap := &GraphSnapshot{}
	snap.EnsureMaps()

	// Register nodes
	snap.Nodes[depKey] = dep
	snap.Nodes[pod1Key] = pod1
	snap.Nodes[pod2Key] = pod2
	snap.Nodes[pod3Key] = pod3
	snap.Nodes[svcKey] = svc

	// Pod owners: all 3 pods owned by the Deployment
	snap.PodOwners[pod1Key] = depKey
	snap.PodOwners[pod2Key] = depKey
	snap.PodOwners[pod3Key] = depKey

	// Replicas
	snap.NodeReplicas[depKey] = 3

	// Service endpoints: 3 ready endpoints pointing to the 3 pods
	snap.ServiceEndpoints[svcKey] = []corev1.EndpointAddress{
		{
			IP: "10.0.0.1",
			TargetRef: &corev1.ObjectReference{
				Kind:      "Pod",
				Namespace: "default",
				Name:      "pod-1",
			},
		},
		{
			IP: "10.0.0.2",
			TargetRef: &corev1.ObjectReference{
				Kind:      "Pod",
				Namespace: "default",
				Name:      "pod-2",
			},
		},
		{
			IP: "10.0.0.3",
			TargetRef: &corev1.ObjectReference{
				Kind:      "Pod",
				Namespace: "default",
				Name:      "pod-3",
			},
		},
	}

	// Graph edges: Service depends on Deployment, Deployment owns Pods
	snap.Forward[svcKey] = map[string]bool{depKey: true}
	snap.Reverse[depKey] = map[string]bool{svcKey: true}

	snap.Edges = []models.BlastDependencyEdge{
		{Source: svc, Target: dep, Type: "selects"},
	}

	snap.TotalWorkloads = 2 // Deployment + Service
	snap.BuiltAt = time.Now().UnixMilli()
	snap.Namespaces["default"] = true

	return snap
}

// buildSnapshotScenario2 creates a cluster with:
//   - 1 Deployment "api" with 1 replica
//   - 1 Pod owned by it
//   - 1 Service "api-svc" with 1 endpoint
func buildSnapshotScenario2() *GraphSnapshot {
	dep := models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "api"}
	pod := models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "api-pod-1"}
	svc := models.ResourceRef{Kind: "Service", Namespace: "default", Name: "api-svc"}

	depKey := refKey(dep)
	podKey := refKey(pod)
	svcKey := refKey(svc)

	snap := &GraphSnapshot{}
	snap.EnsureMaps()

	// Register nodes
	snap.Nodes[depKey] = dep
	snap.Nodes[podKey] = pod
	snap.Nodes[svcKey] = svc

	// Pod owners: 1 pod owned by the Deployment
	snap.PodOwners[podKey] = depKey

	// Replicas: single replica
	snap.NodeReplicas[depKey] = 1

	// Service endpoints: 1 endpoint pointing to the single pod
	snap.ServiceEndpoints[svcKey] = []corev1.EndpointAddress{
		{
			IP: "10.0.0.1",
			TargetRef: &corev1.ObjectReference{
				Kind:      "Pod",
				Namespace: "default",
				Name:      "api-pod-1",
			},
		},
	}

	// Graph edges
	snap.Forward[svcKey] = map[string]bool{depKey: true}
	snap.Reverse[depKey] = map[string]bool{svcKey: true}

	snap.Edges = []models.BlastDependencyEdge{
		{Source: svc, Target: dep, Type: "selects"},
	}

	snap.TotalWorkloads = 2 // Deployment + Service
	snap.BuiltAt = time.Now().UnixMilli()
	snap.Namespaces["default"] = true

	return snap
}

// TestScenario_PodCrashWithReplicas tests that crashing one pod from a 3-replica
// Deployment causes minimal impact: the service should be classified "self-healing"
// and the blast radius should be 0 (or near 0).
func TestScenario_PodCrashWithReplicas(t *testing.T) {
	snap := buildSnapshotScenario1()

	pod1 := models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "pod-1"}
	result, err := snap.ComputeBlastRadiusWithMode(pod1, FailureModePodCrash)
	require.NoError(t, err)
	require.NotNil(t, result)

	// With 3 replicas and only 1 lost, the service remains above the 50% threshold
	// → classification must be "self-healing", weight = 0.0 → blast radius = 0%
	assert.InDelta(t, 0.0, result.BlastRadiusPercent, 0.01,
		"blast radius should be ~0%% when only 1 of 3 replicas is lost")

	// Verify the affected service is classified as self-healing
	require.NotEmpty(t, result.AffectedServices,
		"should have at least one service impact entry")
	found := false
	for _, si := range result.AffectedServices {
		if si.Service.Name == "app-svc" {
			found = true
			assert.Equal(t, "self-healing", si.Classification,
				"app-svc should be self-healing: 2 of 3 endpoints remain")
		}
	}
	assert.True(t, found, "app-svc should appear in AffectedServices")
}

// TestScenario_SingleReplicaPodCrash tests that crashing the only pod of a
// single-replica Deployment results in a broken service and non-zero blast radius.
func TestScenario_SingleReplicaPodCrash(t *testing.T) {
	snap := buildSnapshotScenario2()

	pod := models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "api-pod-1"}
	result, err := snap.ComputeBlastRadiusWithMode(pod, FailureModePodCrash)
	require.NoError(t, err)
	require.NotNil(t, result)

	// With only 1 replica and it lost, the service must be "broken"
	require.NotEmpty(t, result.AffectedServices,
		"should have at least one service impact entry")
	found := false
	for _, si := range result.AffectedServices {
		if si.Service.Name == "api-svc" {
			found = true
			assert.Equal(t, "broken", si.Classification,
				"api-svc should be broken: 0 of 1 endpoints remain")
		}
	}
	assert.True(t, found, "api-svc should appear in AffectedServices")

	// Blast radius must be strictly greater than 0
	assert.Greater(t, result.BlastRadiusPercent, 0.0,
		"blast radius should be >0%% when the single replica is lost")
}

// TestScenario_ServiceLosingAllEndpoints is a placeholder for future implementation.
func TestScenario_ServiceLosingAllEndpoints(t *testing.T) {
	t.Skip("TODO: implement scenario — service loses all endpoints via workload deletion")
}

// TestScenario_IngressLosingBackend is a placeholder for future implementation.
func TestScenario_IngressLosingBackend(t *testing.T) {
	t.Skip("TODO: implement scenario — ingress loses all backend services")
}

// TestScenario_NamespaceDeletion is a placeholder for future implementation.
func TestScenario_NamespaceDeletion(t *testing.T) {
	t.Skip("TODO: implement scenario — namespace deletion cascades to all workloads")
}

// TestScenario_StatefulSetFailure is a placeholder for future implementation.
func TestScenario_StatefulSetFailure(t *testing.T) {
	t.Skip("TODO: implement scenario — StatefulSet pod failure with PVC dependency")
}

// TestScenario_ControlPlaneComponent is a placeholder for future implementation.
func TestScenario_ControlPlaneComponent(t *testing.T) {
	t.Skip("TODO: implement scenario — control-plane component failure triggers 100%% blast radius override")
}
