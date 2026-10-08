package topology

// Phase 2 (docs/TOPOLOGY-SCALE-INVESTIGATION.md): inferStorageRelationships
// used to re-fetch each PVC/PV individually via a live K8s Get call whenever
// EITHER volumeName or storageClassName was empty — but an unbound
// (Pending) PVC legitimately has an empty volumeName forever, so this fired
// on every single such PVC, every request, regardless of whether discovery
// had already recorded everything actually available. At ~75 PVCs,
// throttled by client-go's default QPS=5/Burst=10 limiter (no custom
// QPS/Burst configured anywhere in this codebase), this was the ENTIRE
// 16-19 second GetTopology (V1) bottleneck at ~2,248-pod scale —
// live-reproduced before and after. These tests prove the fallback only
// fires when discovery genuinely never ran for that node (extra == nil),
// never merely because one field is legitimately absent.

import (
	"context"
	"sync/atomic"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	"github.com/kubilitics/kubilitics-backend/internal/k8s"
	"github.com/kubilitics/kubilitics-backend/internal/models"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestInferStorageRelationships_PendingPVCWithExtra_DoesNotRefetch(t *testing.T) {
	var pvcGetCount, pvGetCount int32
	cs := k8sfake.NewSimpleClientset(
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pending"}},
	)
	cs.PrependReactor("get", "persistentvolumeclaims", func(k8stesting.Action) (bool, runtime.Object, error) {
		atomic.AddInt32(&pvcGetCount, 1)
		return false, nil, nil // not handled -> fall through to default tracker behavior
	})
	cs.PrependReactor("get", "persistentvolumes", func(k8stesting.Action) (bool, runtime.Object, error) {
		atomic.AddInt32(&pvGetCount, 1)
		return false, nil, nil
	})

	client := k8s.NewClientForTest(cs)
	engine := NewEngine(client, nil)
	graph := NewGraph(0)
	graph.AddNode(models.TopologyNode{
		ID: "PersistentVolumeClaim/default/pending", Kind: "PersistentVolumeClaim", Namespace: "default", Name: "pending",
	})
	graph.AddNode(models.TopologyNode{
		ID: "StorageClass/synthetic-sc", Kind: "StorageClass", Name: "synthetic-sc",
	})
	// Discovery already recorded this PVC's extra data: storageClassName
	// present, volumeName legitimately absent (Pending, never bound).
	graph.SetNodeExtra("PersistentVolumeClaim/default/pending", map[string]interface{}{
		"storageClassName": "synthetic-sc",
	})

	ri := NewRelationshipInferencer(engine, graph)
	if err := ri.inferStorageRelationships(context.Background()); err != nil {
		t.Fatalf("inferStorageRelationships: %v", err)
	}

	if got := atomic.LoadInt32(&pvcGetCount); got != 0 {
		t.Fatalf("expected 0 live PVC Get calls (extra was already populated), got %d — the N+1 regression has returned", got)
	}
	if !graph.EdgeMap["PersistentVolumeClaim/default/pending->StorageClass/synthetic-sc:stores"] {
		t.Fatal("expected PVC->StorageClass edge to still be created from the cached extra data")
	}
}

// When discovery genuinely never ran for a node (extra == nil — e.g. a
// hand-built test graph, or a future code path that doesn't go through
// discoverPersistentVolumeClaims), the live-Get fallback must still work —
// this is not a removal of the fallback, only a correction of when it fires.
func TestInferStorageRelationships_NoExtraAtAll_StillFallsBackToLiveGet(t *testing.T) {
	var pvcGetCount int32
	cs := k8sfake.NewSimpleClientset(
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "legacy"},
			Spec:       corev1.PersistentVolumeClaimSpec{VolumeName: "pv-legacy"},
		},
	)
	cs.PrependReactor("get", "persistentvolumeclaims", func(k8stesting.Action) (bool, runtime.Object, error) {
		atomic.AddInt32(&pvcGetCount, 1)
		return false, nil, nil
	})

	client := k8s.NewClientForTest(cs)
	engine := NewEngine(client, nil)
	graph := NewGraph(0)
	graph.AddNode(models.TopologyNode{ID: "PersistentVolumeClaim/default/legacy", Kind: "PersistentVolumeClaim", Namespace: "default", Name: "legacy"})
	graph.AddNode(models.TopologyNode{ID: "PersistentVolume/pv-legacy", Kind: "PersistentVolume", Name: "pv-legacy"})
	// No SetNodeExtra call at all for this node.

	ri := NewRelationshipInferencer(engine, graph)
	if err := ri.inferStorageRelationships(context.Background()); err != nil {
		t.Fatalf("inferStorageRelationships: %v", err)
	}

	if got := atomic.LoadInt32(&pvcGetCount); got != 1 {
		t.Fatalf("expected exactly 1 live Get call (extra was nil), got %d", got)
	}
	if !graph.EdgeMap["PersistentVolumeClaim/default/legacy->PersistentVolume/pv-legacy:stores"] {
		t.Fatal("expected PVC->PV edge from the live-fetched volumeName")
	}
}
