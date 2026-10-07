package v2

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/kubilitics/kubilitics-backend/internal/k8s"
)

// newPaginatedFakeClientset constructs a fake clientset seeded with n
// Deployments and a reactor that paginates List responses in pages of
// pageSize — the fake clientset's default ObjectTracker ignores
// ListOptions.Limit/Continue entirely (always returns everything in one
// page), so without this reactor a test cannot distinguish a collector that
// paginates correctly from one that silently truncates at Limit, which is
// exactly the BLASTRADIUS-2 bug.
// regardless of how many objects actually exist — proving the collector
// follows Continue tokens to completion rather than stopping at one page.
func newPaginatedFakeClientset(totalDeployments, pageSize int) *k8sfake.Clientset {
	objs := make([]runtime.Object, 0, totalDeployments)
	for i := 0; i < totalDeployments; i++ {
		objs = append(objs, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deploy-" + strconv.Itoa(i),
				Namespace: "default",
			},
		})
	}
	cs := k8sfake.NewSimpleClientset(objs...)

	cs.PrependReactor("list", "deployments", func(action k8stesting.Action) (bool, runtime.Object, error) {
		listAction := action.(k8stesting.ListActionImpl)
		start := 0
		if c := listAction.GetListOptions().Continue; c != "" {
			v, err := strconv.Atoi(c)
			if err != nil {
				return false, nil, err
			}
			start = v
		}
		end := start + pageSize
		full := &appsv1.DeploymentList{}
		for i := 0; i < totalDeployments; i++ {
			full.Items = append(full.Items, appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "deploy-" + strconv.Itoa(i), Namespace: "default"},
			})
		}
		if start > len(full.Items) {
			start = len(full.Items)
		}
		if end > len(full.Items) {
			end = len(full.Items)
		}
		page := &appsv1.DeploymentList{Items: full.Items[start:end]}
		if end < len(full.Items) {
			page.Continue = strconv.Itoa(end)
		}
		return true, page, nil
	})

	return cs
}

// TestCollectFromClient_PaginatesBeyondSinglePage is the BLASTRADIUS-2
// regression test. Before the fix, every resource type except Pods/Events
// used a single ListOptions{Limit: 500} call and discarded the returned
// Continue token, so any resource type with more than 500 live instances was
// silently truncated. 1200 deployments across a 500-item page size requires
// 3 pages; a non-paginating collector would return exactly 500.
func TestCollectFromClient_PaginatesBeyondSinglePage(t *testing.T) {
	const total = 1200
	cs := newPaginatedFakeClientset(total, collectPageSize)
	client := k8s.NewClientForTest(cs)

	bundle, err := CollectFromClient(context.Background(), client, "", nil)
	if err != nil {
		t.Fatalf("CollectFromClient returned error: %v", err)
	}
	if len(bundle.Deployments) != total {
		t.Fatalf("expected %d deployments (fully paginated), got %d — BLASTRADIUS-2 regression (silent truncation at page size)", total, len(bundle.Deployments))
	}
	for _, rt := range bundle.FailedResources {
		if rt == "deployments" {
			t.Fatalf("deployments should not be reported in FailedResources when pagination completes normally")
		}
	}
}

// TestCollectFromClient_SafetyCapFlagsIncompleteResourceType proves that if
// a resource type's Continue token never terminates (a pathological or
// misbehaving API server), the collector stops at collectMaxPages instead of
// looping forever, AND reports the resource type via the existing
// FailedResources mechanism rather than silently returning a partial,
// confident-looking result.
func TestCollectFromClient_SafetyCapFlagsIncompleteResourceType(t *testing.T) {
	cs := k8sfake.NewSimpleClientset()
	cs.PrependReactor("list", "deployments", func(action k8stesting.Action) (bool, runtime.Object, error) {
		// Always return a full page AND a non-empty Continue token — simulates
		// a server that never terminates pagination.
		items := make([]appsv1.Deployment, collectPageSize)
		for i := range items {
			items[i] = appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: strconv.Itoa(i), Namespace: "default"}}
		}
		return true, &appsv1.DeploymentList{Items: items, ListMeta: metav1.ListMeta{Continue: "more"}}, nil
	})
	client := k8s.NewClientForTest(cs)

	bundle, err := CollectFromClient(context.Background(), client, "", nil)
	if err != nil {
		t.Fatalf("CollectFromClient returned error: %v", err)
	}
	if want := collectMaxPages * collectPageSize; len(bundle.Deployments) != want {
		t.Fatalf("expected the safety cap to stop collection at %d items, got %d", want, len(bundle.Deployments))
	}
	found := false
	for _, rt := range bundle.FailedResources {
		if rt == "deployments" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected 'deployments' to be recorded in FailedResources when the safety cap is hit, so callers can tell the graph is incomplete rather than genuinely complete")
	}
}

// Batch 1 gap (Theme 1 #3/#4): collectFromClient previously always
// live-listed every resource type, unlike buildClusterSummary/
// GetClusterSummary which were already routed through the informer cache
// in Batch 1. These tests prove the fix: once a resource type's informer
// has synced, collectFromClient serves it from cache instead of a live API
// call — verified by making the live path return an error/different data
// and confirming the cache's data (not the live path's) is what comes back.

func waitForSynced(t *testing.T, im *k8s.InformerManager) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !im.HasSynced() {
		if time.Now().After(deadline) {
			t.Fatal("informer manager did not finish initial sync in time")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestCollectFromClient_ServesPodsFromCacheWhenInformerSynced(t *testing.T) {
	cs := k8sfake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "cached-pod", Namespace: "default"},
	})
	client := k8s.NewClientForTest(cs)
	im := k8s.NewInformerManager(client)
	if err := im.Start(context.Background()); err != nil {
		t.Logf("Start returned %v (informers may still sync via background retry)", err)
	}
	waitForSynced(t, im)

	// After the informer has synced, make the LIVE path fail. If
	// collectFromClient still succeeds and returns the pod, it proves the
	// cache — not this reactor — served the data.
	cs.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("live API must not be called for an informer-tracked, synced resource type")
	})

	bundle, err := CollectFromClient(context.Background(), client, "", im)
	if err != nil {
		t.Fatalf("CollectFromClient returned error: %v", err)
	}
	if len(bundle.Pods) != 1 || bundle.Pods[0].Name != "cached-pod" {
		t.Fatalf("expected the single cached pod, got %+v", bundle.Pods)
	}
	for _, rt := range bundle.FailedResources {
		if rt == "pods" {
			t.Fatal("pods must not be recorded as failed — it should have been served from cache, never reaching the failing live reactor")
		}
	}
}

func TestCollectFromClient_FallsBackToLiveWhenInformerNotSynced(t *testing.T) {
	cs := k8sfake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "live-pod", Namespace: "default"},
	})
	client := k8s.NewClientForTest(cs)
	// A real but never-started InformerManager: HasSynced() is always false,
	// so every ListFromCache call must report a miss and this must still
	// fall back to a live call exactly as if im were nil.
	im := k8s.NewInformerManager(client)

	bundle, err := CollectFromClient(context.Background(), client, "", im)
	if err != nil {
		t.Fatalf("CollectFromClient returned error: %v", err)
	}
	if len(bundle.Pods) != 1 || bundle.Pods[0].Name != "live-pod" {
		t.Fatalf("expected the live-fetched pod, got %+v", bundle.Pods)
	}
}

func TestCollectFromClient_UntrackedKindAlwaysLiveEvenWithSyncedInformer(t *testing.T) {
	// EndpointSlices is deliberately NOT one of the 27 kinds InformerManager
	// tracks (Theme 4's informer-coverage gap, separate from this fix) — it
	// must always go live, synced informer or not.
	cs := k8sfake.NewSimpleClientset()
	client := k8s.NewClientForTest(cs)
	im := k8s.NewInformerManager(client)
	if err := im.Start(context.Background()); err != nil {
		t.Logf("Start returned %v", err)
	}
	waitForSynced(t, im)

	liveCalled := false
	cs.PrependReactor("list", "endpointslices", func(k8stesting.Action) (bool, runtime.Object, error) {
		liveCalled = true
		return false, nil, nil // let the fake's default tracker handle it
	})

	_, err := CollectFromClient(context.Background(), client, "", im)
	if err != nil {
		t.Fatalf("CollectFromClient returned error: %v", err)
	}
	if !liveCalled {
		t.Fatal("expected EndpointSlices (an untracked kind) to still go through the live API even with a fully synced informer manager")
	}
}
