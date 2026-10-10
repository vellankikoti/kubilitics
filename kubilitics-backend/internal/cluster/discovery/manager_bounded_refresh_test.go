package discovery

// LOADING-4 Phase B (docs/LOADING4-BOUNDED-IO-SWEEP.md): Manager.Refresh is
// called from cmd/server/main.go at startup, on every cluster
// add/remove/reconnect, and from a 60s ticker — previously always with a
// bare context.Background(). KubernetesSecretSource.Enumerate (registered
// only in the Helm hub/agent in-cluster deployment mode) performs a real
// Secrets().List(ctx,...) call; a hung in-cluster API server could block
// Refresh (and, at startup, the whole backend becoming ready) indefinitely.
// This test proves Refresh itself respects a bounded context — the actual
// production fix is wrapping every call site with one (cmd/server/main.go's
// refreshDiscoveryBounded), which cannot be unit-tested directly from the
// `main` package, so the fix is verified at the level that IS testable:
// Manager.Refresh's own behavior when a source hangs.

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kubilitics/kubilitics-backend/internal/cluster/identity"
)

// hangingSource blocks in Enumerate until the caller's context is done,
// then returns ctx.Err() — proving real cancellation propagation (this
// Enumerate implementation is hand-written, not routed through any fake
// client-go reactor, so there is no "Action has no context" limitation
// here as there was for the PipelineManager/buildClusterSummary fixes).
type hangingSource struct {
	name        string
	started     chan struct{}
	startedOnce sync.Once
}

func newHangingSource(name string) *hangingSource {
	return &hangingSource{name: name, started: make(chan struct{})}
}

func (h *hangingSource) Name() string { return h.name }
func (h *hangingSource) Enumerate(ctx context.Context) ([]DiscoveredCluster, error) {
	// TestManager_Refresh_ConcurrentCallsWithHangingSource_RaceFree calls
	// Enumerate concurrently on the same instance — a plain
	// select-default-close on h.started is check-then-act and can double
	// close when two goroutines both see it open. sync.Once makes the
	// first-closer-wins race safe instead of just unlikely.
	h.startedOnce.Do(func() { close(h.started) })
	<-ctx.Done()
	return nil, ctx.Err()
}
func (h *hangingSource) Watch(ctx context.Context) (<-chan DiscoveryEvent, error) {
	return nil, nil
}

// A hanging source must not block Refresh past the caller's own context
// deadline, and a healthy source listed AFTER the hanging one must still
// be enumerated once the hanging source's call returns (it does not abort
// the whole Refresh — existing "one broken source should not blank out
// others" behavior, confirmed still intact with a HANG, not just an error).
func TestManager_Refresh_HangingSourceIsBoundedByCallerContext(t *testing.T) {
	hanging := newHangingSource("hanging")
	healthy := &fakeSource{clusters: []DiscoveredCluster{
		{Identity: identity.LogicalIdentity{Name: "healthy-cluster", ServerURL: "https://x"}, Source: "fake"},
	}}
	m := NewManager([]DiscoverySource{hanging, healthy})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := m.Refresh(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Refresh returned an error (expected nil — per-source errors, including a timed-out hang, must not abort the whole refresh): %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("Refresh took %v with one permanently-hanging source — expected close to the caller's 200ms context deadline, not an unbounded hang", elapsed)
	}

	snap := m.Snapshot()
	found := false
	for _, c := range snap.Discovered {
		if c.Identity.Name == "healthy-cluster" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected the healthy source (listed after the hanging one) to still be enumerated and present in the snapshot")
	}
}

// Multiple concurrent Refresh calls against a hanging source must not
// deadlock or accumulate goroutines beyond what's actually in flight — run
// under -race.
func TestManager_Refresh_ConcurrentCallsWithHangingSource_RaceFree(t *testing.T) {
	hanging := newHangingSource("hanging")
	m := NewManager([]DiscoverySource{hanging})

	const n = 5
	done := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			defer cancel()
			done <- m.Refresh(ctx)
		}()
	}
	for i := 0; i < n; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Refresh returned error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("a concurrent Refresh call never returned")
		}
	}
}
