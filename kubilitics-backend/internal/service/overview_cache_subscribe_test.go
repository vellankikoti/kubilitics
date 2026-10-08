package service

import (
	"context"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubilitics/kubilitics-backend/internal/k8s"
)

// Batch 2 / Theme 2 #9: these tests cover Subscribe/notifyStream/
// StopClusterCache's interaction with the per-cluster lock-sharding refactor
// — specifically the two behaviors that were easy to lose when splitting one
// global map set into per-cluster entries: (1) Subscribe may be called
// before StartClusterCache for the same cluster, and (2) a listener
// registered via Subscribe must survive StopClusterCache (only the
// informer/overview are torn down, not the listener registry). Neither had
// any prior test coverage.

func TestSubscribe_BeforeStartClusterCache_ListenerSurvivesAndReceivesUpdateAfterStart(t *testing.T) {
	const clusterID = "c1"
	c := NewOverviewCache()

	ch, unsubscribe, err := c.Subscribe(clusterID)
	if err != nil {
		t.Fatalf("Subscribe before start: %v", err)
	}
	defer unsubscribe()

	// No overview yet — no initial push should have been sent.
	select {
	case <-ch:
		t.Fatal("expected no initial push before the cluster has started")
	case <-time.After(20 * time.Millisecond):
	}

	clientset := fake.NewSimpleClientset()
	client := k8s.NewClientForTest(clientset)
	if err := c.StartClusterCache(context.Background(), clusterID, client); err != nil {
		t.Fatalf("StartClusterCache: %v", err)
	}
	defer c.StopClusterCache(clusterID)

	c.notifyStream(clusterID)

	select {
	case snap := <-ch:
		if snap == nil {
			t.Fatal("expected a non-nil snapshot after start + notify")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected the pre-registered listener to receive an update after StartClusterCache")
	}
}

func TestSubscribe_InitialPushWhenOverviewAlreadyExists(t *testing.T) {
	const clusterID = "c1"
	c := NewOverviewCache()
	clientset := fake.NewSimpleClientset()
	client := k8s.NewClientForTest(clientset)
	if err := c.StartClusterCache(context.Background(), clusterID, client); err != nil {
		t.Fatalf("StartClusterCache: %v", err)
	}
	defer c.StopClusterCache(clusterID)

	ch, unsubscribe, err := c.Subscribe(clusterID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer unsubscribe()

	select {
	case snap := <-ch:
		if snap == nil {
			t.Fatal("expected a non-nil initial snapshot")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected an initial push since the overview already exists")
	}
}

func TestStopClusterCache_ListenerSurvivesStopAndReceivesUpdateAfterRestart(t *testing.T) {
	const clusterID = "c1"
	c := NewOverviewCache()
	clientset := fake.NewSimpleClientset()
	client := k8s.NewClientForTest(clientset)
	if err := c.StartClusterCache(context.Background(), clusterID, client); err != nil {
		t.Fatalf("StartClusterCache: %v", err)
	}

	ch, unsubscribe, err := c.Subscribe(clusterID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer unsubscribe()

	// Drain the initial push from Subscribe.
	<-ch

	c.StopClusterCache(clusterID)

	if _, ok := c.GetOverview(clusterID); ok {
		t.Fatal("expected GetOverview to report not-found after StopClusterCache")
	}

	// The listener itself must still be registered — restarting the same
	// cluster and notifying should reach it, proving Stop didn't drop it.
	if err := c.StartClusterCache(context.Background(), clusterID, client); err != nil {
		t.Fatalf("StartClusterCache (restart): %v", err)
	}
	defer c.StopClusterCache(clusterID)

	c.notifyStream(clusterID)

	select {
	case snap := <-ch:
		if snap == nil {
			t.Fatal("expected a non-nil snapshot after restart")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected the listener registered before Stop to still receive updates after a restart")
	}
}

func TestUnsubscribe_ClosesChannelAndStopsFutureDelivery(t *testing.T) {
	const clusterID = "c1"
	c := NewOverviewCache()
	clientset := fake.NewSimpleClientset()
	client := k8s.NewClientForTest(clientset)
	if err := c.StartClusterCache(context.Background(), clusterID, client); err != nil {
		t.Fatalf("StartClusterCache: %v", err)
	}
	defer c.StopClusterCache(clusterID)

	ch, unsubscribe, err := c.Subscribe(clusterID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	<-ch // drain initial push

	unsubscribe()

	_, stillOpen := <-ch
	if stillOpen {
		t.Fatal("expected channel to be closed after unsubscribe")
	}

	// Calling unsubscribe a second time must not panic (double-close guard).
	unsubscribe()
}

func TestSubscribe_ListenerLimitEnforced(t *testing.T) {
	const clusterID = "c1"
	c := NewOverviewCache()

	var unsubs []func()
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	for i := 0; i < maxListenersPerCluster; i++ {
		_, unsub, err := c.Subscribe(clusterID)
		if err != nil {
			t.Fatalf("Subscribe #%d: unexpected error %v", i, err)
		}
		unsubs = append(unsubs, unsub)
	}

	if _, _, err := c.Subscribe(clusterID); err != ErrTooManyListeners {
		t.Fatalf("expected ErrTooManyListeners at the limit, got %v", err)
	}
}
