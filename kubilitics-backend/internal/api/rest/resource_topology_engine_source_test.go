package rest

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/kubilitics/kubilitics-backend/internal/graph"
	"github.com/kubilitics/kubilitics-backend/internal/k8s"
	"github.com/kubilitics/kubilitics-backend/internal/models"
)

// newCountingDeploymentClient builds a fake k8s.Client seeded with one
// Deployment and a counter that increments on every live "list deployments"
// call, so the test can distinguish "sourced from the engine's informer
// cache" (counter stays 0) from "re-fetched live" (counter increments).
func newCountingDeploymentClient(clusterID string) (*k8s.Client, *int32) {
	var liveListCount int32
	cs := k8sfake.NewSimpleClientset(
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
	)
	cs.PrependReactor("list", "deployments", func(action k8stesting.Action) (bool, runtime.Object, error) {
		atomic.AddInt32(&liveListCount, 1)
		return false, nil, nil // let the default tracker still serve it
	})
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), nil)
	client := k8s.NewClientForTest(cs)
	client.Dynamic = dyn
	client.SetClusterID(clusterID)
	return client, &liveListCount
}

func newResourceTopologyTestHandler(t *testing.T, clusterID string, client *k8s.Client) *Handler {
	t.Helper()
	cs := &mockClusterService{
		clusterMap: map[string]*models.Cluster{clusterID: {ID: clusterID, Name: clusterID}},
		clientMap:  map[string]*k8s.Client{clusterID: client},
	}
	return NewHandler(cs, noopTopologyService{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func requestResourceTopology(h *Handler, clusterID string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "/clusters/"+clusterID+"/topology/resource/Deployment/default/web", nil)
	return serve(h, r)
}

// TestGetResourceTopology_UsesRunningEngineInsteadOfLiveFetch is the
// BLASTRADIUS-1 regression test. Before the fix, GetResourceTopology always
// called CollectFromClient, which live-lists every resource type on each
// topology cache miss even when this cluster's ClusterGraphEngine was
// already maintaining an informer-cached copy of the same data for
// blast-radius scoring. With the fix, once the engine is running and ready,
// the overlapping resource types (deployments included) are sourced from the
// engine's cache and the live "list deployments" call is skipped entirely.
func TestGetResourceTopology_UsesRunningEngineInsteadOfLiveFetch(t *testing.T) {
	const clusterID = "engine-sourced-cluster"
	client, liveListCount := newCountingDeploymentClient(clusterID)
	h := newResourceTopologyTestHandler(t, clusterID, client)

	h.graphEngineMgr = graph.NewEngineLifecycleManager(time.Hour, nil)
	engine, err := h.graphEngineMgr.EnsureActive(context.Background(), clusterID, client.Clientset, slog.Default())
	if err != nil {
		t.Fatalf("EnsureActive: %v", err)
	}
	deadline := time.After(5 * time.Second)
	for !engine.Status().Ready {
		select {
		case <-deadline:
			t.Fatal("graph engine never became ready")
		case <-time.After(10 * time.Millisecond):
		}
	}

	// The engine's own informer bootstrap already issued one live "list
	// deployments" call to seed its cache (that's how informers populate
	// themselves) — baseline captures that so the assertion below measures
	// only what GetResourceTopology itself does, not the engine's own sync.
	baseline := atomic.LoadInt32(liveListCount)

	w := requestResourceTopology(h, clusterID)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := atomic.LoadInt32(liveListCount) - baseline; got != 0 {
		t.Fatalf("expected 0 additional live 'list deployments' calls from GetResourceTopology (sourced from the running graph engine instead), got %d — BLASTRADIUS-1 regression", got)
	}
	if !strings.Contains(w.Body.String(), "web") {
		t.Fatalf("expected response to still contain the deployment's data sourced from the engine, got: %s", w.Body.String())
	}
}

// TestGetResourceTopology_LiveFetchesWhenNoEngineRunning is the companion
// baseline: when no graph engine is running for the cluster (the common case
// for a cluster whose Blast Radius tab has never been opened), the handler
// must still live-fetch and succeed — proving the engine-sourcing path in
// the test above is a genuine behavioral difference, not dead code that
// never runs either way.
func TestGetResourceTopology_LiveFetchesWhenNoEngineRunning(t *testing.T) {
	const clusterID = "no-engine-cluster"
	client, liveListCount := newCountingDeploymentClient(clusterID)
	h := newResourceTopologyTestHandler(t, clusterID, client)

	w := requestResourceTopology(h, clusterID)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := atomic.LoadInt32(liveListCount); got == 0 {
		t.Fatal("expected at least one live 'list deployments' call when no graph engine is running")
	}
	if !strings.Contains(w.Body.String(), "web") {
		t.Fatalf("expected response to contain the deployment's data sourced live, got: %s", w.Body.String())
	}
}

// TestGetResourceTopology_ScopesLiveCollectionToResourceNamespace covers a
// real production bug: GetResourceTopology always built the v2 Options with
// Namespace left empty, so when no graph engine was running (the common
// case), CollectFromClient listed every namespaced resource type across
// EVERY namespace in the cluster just to show a 1-2 hop neighborhood around
// one resource — the "fetches everything, times out, shows nothing" failure
// users hit on larger/multi-namespace clusters. Seeds two namespaces with a
// Deployment each and asserts the live "list deployments" call made for a
// resource-topology request on namespace "target-ns" is namespace-scoped to
// "target-ns", not cluster-wide (""), and never touches "other-ns" at all.
func TestGetResourceTopology_ScopesLiveCollectionToResourceNamespace(t *testing.T) {
	const clusterID = "ns-scoped-cluster"
	cs := k8sfake.NewSimpleClientset(
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "target-ns"}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "other-ns"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "target-ns"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "other-ns"}},
	)

	var listedNamespaces []string
	cs.PrependReactor("list", "deployments", func(action k8stesting.Action) (bool, runtime.Object, error) {
		listAction := action.(k8stesting.ListActionImpl)
		listedNamespaces = append(listedNamespaces, listAction.GetNamespace())
		return false, nil, nil // let the default tracker still serve it
	})

	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), nil)
	client := k8s.NewClientForTest(cs)
	client.Dynamic = dyn
	client.SetClusterID(clusterID)

	h := newResourceTopologyTestHandler(t, clusterID, client)
	// No graph engine started — forces the live CollectFromClient path,
	// which is exactly what the Namespace-scoping fix targets.
	r := httptest.NewRequest(http.MethodGet, "/clusters/"+clusterID+"/topology/resource/Deployment/target-ns/web", nil)
	w := serve(h, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "web") {
		t.Fatalf("expected response to contain the target deployment, got: %s", w.Body.String())
	}
	for _, ns := range listedNamespaces {
		if ns != "target-ns" {
			t.Fatalf("expected every live 'list deployments' call to be scoped to namespace %q, got a call scoped to %q (listedNamespaces=%v) — this is the cluster-wide-fetch regression", "target-ns", ns, listedNamespaces)
		}
	}
	if len(listedNamespaces) == 0 {
		t.Fatal("expected at least one live 'list deployments' call")
	}
}
