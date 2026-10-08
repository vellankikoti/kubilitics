package k8s

import (
	"errors"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// LOADING-5 (docs/PRODUCTION-RELIABILITY-AUDIT.md): WaitForCacheSync(im.stopCh)
// previously blocked forever if a resource's informer never finished its
// initial sync — nothing but an explicit Stop() could unblock it. These tests
// prove the bounded wait actually gives up, the happy path is unaffected, and
// the background retry loop exits promptly on Stop() instead of leaking.

func TestInformerManager_WaitForSync_TimesOutIfNeverStarted(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	// Simulates the audit's own example: a resource type whose watch/list is
	// permanently denied (e.g. RBAC) — the reflector retries forever and
	// never calls store.Replace(), so HasSynced() can never become true.
	clientset.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("simulated permission denied — initial sync can never complete")
	})

	client := NewClientForTest(clientset)
	im := NewInformerManager(client)
	defer im.Stop()

	im.setupPodInformer()
	im.factory.Start(im.stopCh) // started, but will never successfully sync

	start := time.Now()
	ok, _ := im.waitForSync(100 * time.Millisecond) // crdFactory is nil for test clients, so crdSynced is trivially true
	elapsed := time.Since(start)

	if ok {
		t.Fatal("expected waitForSync to report not-synced when the informer's list always fails")
	}
	if elapsed > 1*time.Second {
		t.Fatalf("waitForSync took %v to give up, expected to bail out around 100ms — it may be blocking indefinitely again", elapsed)
	}
}

func TestInformerManager_WaitForSync_SucceedsWhenStarted(t *testing.T) {
	client := NewClientForTest(fake.NewSimpleClientset())
	im := NewInformerManager(client)
	defer im.Stop()

	im.setupPodInformer()
	im.factory.Start(im.stopCh)

	mainSynced, _ := im.waitForSync(5 * time.Second) // crdFactory is nil for test clients, so crdSynced is trivially true
	if !mainSynced {
		t.Fatal("expected waitForSync to succeed once factory.Start was called against a fake (near-instant) clientset")
	}
}

func TestInformerManager_RetrySyncInBackground_StopsPromptlyOnStop(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("simulated permission denied — initial sync can never complete")
	})
	client := NewClientForTest(clientset)
	im := NewInformerManager(client)
	im.setupPodInformer()
	im.factory.Start(im.stopCh) // started, but never syncs — the retry loop would otherwise run until the 30s ticker

	done := make(chan struct{})
	go func() {
		im.retrySyncInBackground()
		close(done)
	}()

	time.Sleep(20 * time.Millisecond) // let the goroutine enter its select loop
	im.Stop()

	select {
	case <-done:
		// Returned promptly via the im.stopCh case, not the 30s ticker.
	case <-time.After(1 * time.Second):
		t.Fatal("retrySyncInBackground did not return promptly after Stop() — background goroutine leak")
	}
}
