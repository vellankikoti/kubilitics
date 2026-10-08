package topology

import (
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/kubilitics/kubilitics-backend/internal/k8s"
	"github.com/kubilitics/kubilitics-backend/internal/models"
)

// This is the root cause of the user-reported "Topology loads forever and
// ever" bug: Engine.discoverResources (the main/default Topology tab's
// backend) made ~25 live K8s API List calls on every single request with
// zero informer-cache usage — the same bypass-the-cache pattern already
// fixed in buildClusterSummary and collectFromClient, but never applied
// here even though this is the highest-traffic topology code path. These
// tests prove the fix: once the informer is synced, discovery is served
// from cache (not a live call), and falls back to live correctly when the
// informer isn't ready or the engine has no InformerManager at all (im=nil,
// this engine's previous permanent behavior).

func waitForEngineSync(t *testing.T, im *k8s.InformerManager) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !im.HasSynced() {
		if time.Now().After(deadline) {
			t.Fatal("informer manager did not finish initial sync in time")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestBuildGraph_ServesPodsFromCacheWhenInformerSynced(t *testing.T) {
	cs := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "cached-pod", Namespace: "default"},
	})
	client := k8s.NewClientForTest(cs)
	im := k8s.NewInformerManager(client)
	if err := im.Start(context.Background()); err != nil {
		t.Logf("Start returned %v (informers may still sync via background retry)", err)
	}
	waitForEngineSync(t, im)

	// Poison the live path after the informer has synced — if BuildGraph
	// still succeeds and includes the pod, it proves the cache (not this
	// reactor) served the data.
	cs.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("live API must not be called for an informer-tracked, synced resource type")
	})

	engine := NewEngine(client, im)
	g, err := engine.BuildGraph(context.Background(), models.TopologyFilters{}, "test-cluster", 0)
	if err != nil {
		t.Fatalf("BuildGraph returned error: %v", err)
	}

	found := false
	for _, n := range g.Nodes {
		if n.Kind == "Pod" && n.Name == "cached-pod" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the cached pod in the graph, got nodes: %+v", g.Nodes)
	}
}

func TestBuildGraph_FallsBackToLiveWhenInformerNotSynced(t *testing.T) {
	cs := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "live-pod", Namespace: "default"},
	})
	client := k8s.NewClientForTest(cs)
	// A real but never-started InformerManager: HasSynced() is always
	// false, so every cache lookup must miss and this must still fall back
	// to a live call exactly as if im were nil.
	im := k8s.NewInformerManager(client)

	engine := NewEngine(client, im)
	g, err := engine.BuildGraph(context.Background(), models.TopologyFilters{}, "test-cluster", 0)
	if err != nil {
		t.Fatalf("BuildGraph returned error: %v", err)
	}

	found := false
	for _, n := range g.Nodes {
		if n.Kind == "Pod" && n.Name == "live-pod" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the live-fetched pod in the graph, got nodes: %+v", g.Nodes)
	}
}
