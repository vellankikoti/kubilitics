package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/kubilitics/kubilitics-backend/internal/cluster/identity"
	"github.com/kubilitics/kubilitics-backend/internal/cluster/presence"
)

func TestManager_SnapshotDedupesAcrossSources(t *testing.T) {
	a := &fakeSource{clusters: []DiscoveredCluster{
		{Identity: identity.LogicalIdentity{Name: "prod", ServerURL: "https://x"}, Source: "kubeconfig"},
	}}
	b := &fakeSource{clusters: []DiscoveredCluster{
		{Identity: identity.LogicalIdentity{Name: "prod", ServerURL: "https://x/"}, Source: "secret"},
	}}
	m := NewManager([]DiscoverySource{a, b})
	if err := m.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap := m.Snapshot()
	if len(snap.Discovered) != 1 {
		t.Fatalf("dedup failed: %+v", snap.Discovered)
	}
	if snap.Discovered[0].Source != "kubeconfig" {
		t.Fatalf("expected first-wins (kubeconfig); got %q", snap.Discovered[0].Source)
	}
}

// VALID-05 regression: the "merged kubeconfig" scenario — a cluster seen by
// BOTH ManualSource (has SessionID, registered via a non-default path that
// ManualSource now also reports per the Enumerate fix above) and a second,
// first-wins source that lacks the path. Refresh()'s existing enrichment
// logic (lines ~84-89) must carry KubeconfigPath across from whichever
// source has it, even though the first-seen source's own entry wins overall.
func TestManager_Refresh_EnrichesKubeconfigPathAcrossSources(t *testing.T) {
	firstWins := &fakeSource{clusters: []DiscoveredCluster{
		{Identity: identity.LogicalIdentity{Name: "lab", ServerURL: "https://lab"}, Source: "kubeconfig"},
		// no KubeconfigPath on this one — simulates a source that doesn't know it
	}}
	hasPath := &fakeSource{clusters: []DiscoveredCluster{
		{
			Identity:       identity.LogicalIdentity{Name: "lab", ServerURL: "https://lab"},
			Source:         "manual",
			SessionID:      "uuid-lab",
			KubeconfigPath: "/tmp/kubilitics-lab/kubeconfig.yaml",
			ContextName:    "kind-kubilitics-scale-lab",
		},
	}}
	m := NewManager([]DiscoverySource{firstWins, hasPath})
	if err := m.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap := m.Snapshot()
	if len(snap.Registered) != 1 {
		t.Fatalf("want 1 registered entry (enriched with SessionID from the second source): %+v", snap.Registered)
	}
	got := snap.Registered[0]
	if got.KubeconfigPath != "/tmp/kubilitics-lab/kubeconfig.yaml" {
		t.Fatalf("KubeconfigPath not enriched across sources: %q", got.KubeconfigPath)
	}
	if got.ContextName != "kind-kubilitics-scale-lab" {
		t.Fatalf("ContextName not enriched across sources: %q", got.ContextName)
	}
}

// Task A regression: a DiscoveredCluster carrying SessionID must be
// promoted into presence.RegisteredCluster with SessionID + Provider
// preserved. Discovered-only entries (no SessionID) appear in Discovered
// but NOT in Registered.
func TestManager_SnapshotPromotesManualSourceToRegistered(t *testing.T) {
	src := &fakeSource{clusters: []DiscoveredCluster{
		{
			Identity:  identity.LogicalIdentity{Name: "prod", ServerURL: "https://prod"},
			Source:    "manual",
			SessionID: "uuid-prod",
			Provider:  "eks",
		},
		{
			Identity: identity.LogicalIdentity{Name: "dev", ServerURL: "https://dev"},
			Source:   "kubeconfig",
			// no SessionID — file-sourced only, not yet registered.
		},
	}}
	m := NewManager([]DiscoverySource{src})
	if err := m.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap := m.Snapshot()
	if len(snap.Discovered) != 2 {
		t.Fatalf("both entries must appear in discovered: %+v", snap.Discovered)
	}
	if len(snap.Registered) != 1 {
		t.Fatalf("only the manual entry with SessionID should be promoted: %+v", snap.Registered)
	}
	if snap.Registered[0].SessionID != "uuid-prod" {
		t.Fatalf("SessionID not promoted: %q", snap.Registered[0].SessionID)
	}
	if snap.Registered[0].Provider != "eks" {
		t.Fatalf("Provider not promoted: %q", snap.Registered[0].Provider)
	}
	if snap.Registered[0].Identity.Name != "prod" {
		t.Fatalf("identity not promoted: %+v", snap.Registered[0].Identity)
	}
}

// VALID-05 regression (docs/VALID-05-INVESTIGATION.md): KubeconfigPath and
// ContextName must survive the promotion from discovery.DiscoveredCluster
// into presence.RegisteredCluster. Before this fix, RegisteredCluster had no
// such fields at all, so the frontend's click-to-connect flow always fell
// back to a hardcoded "~/.kube/config" guess — breaking any cluster
// registered via a non-default/custom KUBECONFIG path.
func TestManager_Snapshot_PropagatesKubeconfigPathAndContext(t *testing.T) {
	src := &fakeSource{clusters: []DiscoveredCluster{
		{
			Identity:       identity.LogicalIdentity{Name: "lab", ServerURL: "https://lab"},
			Source:         "manual",
			SessionID:      "uuid-lab",
			KubeconfigPath: "/tmp/kubilitics-lab/kubeconfig.yaml",
			ContextName:    "kind-kubilitics-scale-lab",
		},
	}}
	m := NewManager([]DiscoverySource{src})
	if err := m.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap := m.Snapshot()
	if len(snap.Registered) != 1 {
		t.Fatalf("want 1 registered entry: %+v", snap.Registered)
	}
	got := snap.Registered[0]
	if got.KubeconfigPath != "/tmp/kubilitics-lab/kubeconfig.yaml" {
		t.Fatalf("KubeconfigPath not propagated to presence.RegisteredCluster: %q", got.KubeconfigPath)
	}
	if got.ContextName != "kind-kubilitics-scale-lab" {
		t.Fatalf("ContextName not propagated to presence.RegisteredCluster: %q", got.ContextName)
	}
}

// VALID-05 regression: a cluster discovered ONLY via the default
// ~/.kube/config path (the common case, and what every prior test already
// exercised) must keep working exactly as before — KubeconfigPath/
// ContextName simply stay empty, no behavior change, no panic.
func TestManager_Snapshot_EmptyKubeconfigPathStaysEmpty(t *testing.T) {
	src := &fakeSource{clusters: []DiscoveredCluster{
		{
			Identity:  identity.LogicalIdentity{Name: "default-cluster", ServerURL: "https://default"},
			Source:    "manual",
			SessionID: "uuid-default",
		},
	}}
	m := NewManager([]DiscoverySource{src})
	if err := m.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap := m.Snapshot()
	if len(snap.Registered) != 1 {
		t.Fatalf("want 1 registered entry: %+v", snap.Registered)
	}
	if got := snap.Registered[0]; got.KubeconfigPath != "" || got.ContextName != "" {
		t.Fatalf("expected empty path/context, got: %+v", got)
	}
}

// HEALTH-1 (docs/PRODUCTION-RELIABILITY-AUDIT.md): Snapshot() previously
// hardcoded Reachable: true unconditionally for every registered cluster.
// With no checker wired, it must now fail closed (false), not silently
// preserve the old bug.
func TestManager_Snapshot_FailsClosedWithoutReachabilityChecker(t *testing.T) {
	src := &fakeSource{clusters: []DiscoveredCluster{
		{
			Identity:  identity.LogicalIdentity{Name: "prod", ServerURL: "https://prod"},
			Source:    "manual",
			SessionID: "uuid-prod",
		},
	}}
	m := NewManager([]DiscoverySource{src})
	if err := m.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap := m.Snapshot()
	if len(snap.Registered) != 1 {
		t.Fatalf("expected 1 registered cluster: %+v", snap.Registered)
	}
	if snap.Registered[0].Reachable {
		t.Fatal("expected Reachable=false when no ReachabilityChecker is wired — must fail closed, not default to true")
	}
	if len(snap.Connected) != 0 {
		t.Fatalf("expected Connected to stay empty when unreachable: %+v", snap.Connected)
	}
}

// HEALTH-1/HEALTH-2: with a checker wired, Reachable and the staleness
// timestamps reflect exactly what the checker reports per cluster — and,
// critically, one cluster's checker result must never leak onto another's
// (cluster isolation for the health dimension).
func TestManager_Snapshot_UsesWiredReachabilityChecker_PerClusterIsolation(t *testing.T) {
	src := &fakeSource{clusters: []DiscoveredCluster{
		{Identity: identity.LogicalIdentity{Name: "healthy", ServerURL: "https://healthy"}, Source: "manual", SessionID: "uuid-healthy"},
		{Identity: identity.LogicalIdentity{Name: "down", ServerURL: "https://down"}, Source: "manual", SessionID: "uuid-down"},
	}}
	m := NewManager([]DiscoverySource{src})
	if err := m.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}

	lastChecked := time.Now()
	lastSuccess := lastChecked.Add(-1 * time.Minute)
	m.SetReachabilityChecker(func(sessionID string) ReachabilityStatus {
		switch sessionID {
		case "uuid-healthy":
			return ReachabilityStatus{Reachable: true, LastCheckedAt: lastChecked, LastSuccessAt: lastSuccess}
		case "uuid-down":
			return ReachabilityStatus{Reachable: false, LastCheckedAt: lastChecked, LastError: "connection refused"}
		default:
			t.Fatalf("unexpected sessionID passed to checker: %q", sessionID)
			return ReachabilityStatus{}
		}
	})

	snap := m.Snapshot()
	if len(snap.Registered) != 2 {
		t.Fatalf("expected 2 registered clusters: %+v", snap.Registered)
	}

	byID := map[string]presence.RegisteredCluster{}
	for _, r := range snap.Registered {
		byID[r.SessionID] = r
	}

	healthy := byID["uuid-healthy"]
	if !healthy.Reachable {
		t.Error("uuid-healthy: expected Reachable=true")
	}
	if healthy.LastError != "" {
		t.Errorf("uuid-healthy: expected no LastError, got %q — leaked from uuid-down?", healthy.LastError)
	}
	if healthy.LastSuccessAt == "" {
		t.Error("uuid-healthy: expected LastSuccessAt to be populated")
	}

	down := byID["uuid-down"]
	if down.Reachable {
		t.Error("uuid-down: expected Reachable=false")
	}
	if down.LastError != "connection refused" {
		t.Errorf("uuid-down: expected LastError=\"connection refused\", got %q", down.LastError)
	}
	if down.LastSuccessAt != "" {
		t.Errorf("uuid-down: expected no LastSuccessAt, got %q — leaked from uuid-healthy?", down.LastSuccessAt)
	}
	if down.LastCheckedAt == "" {
		t.Error("uuid-down: expected LastCheckedAt to be populated even though the check failed")
	}

	// Connected must contain exactly the reachable cluster, never the down
	// one — Connected was previously hardcoded to always be empty.
	if len(snap.Connected) != 1 {
		t.Fatalf("expected exactly 1 connected cluster: %+v", snap.Connected)
	}
	if snap.Connected[0].SessionID != "uuid-healthy" {
		t.Errorf("expected connected cluster to be uuid-healthy, got %q", snap.Connected[0].SessionID)
	}
	if snap.Connected[0].ConnectedAt == "" {
		t.Error("expected ConnectedAt to be populated for the connected cluster")
	}
}

func TestManager_WatchFansInFromSources(t *testing.T) {
	a := &fakeSource{events: make(chan DiscoveryEvent, 4)}
	m := NewManager([]DiscoverySource{a})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	events := m.Events(ctx)
	a.events <- DiscoveryEvent{Kind: EventAdd, Cluster: DiscoveredCluster{
		Identity: identity.LogicalIdentity{Name: "c1", ServerURL: "https://c1"}, Source: "fake",
	}}
	select {
	case e := <-events:
		if e.Kind != EventAdd || e.Cluster.Identity.Name != "c1" {
			t.Fatalf("unexpected: %+v", e)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("event not forwarded within 500ms")
	}
}

// TestManager_Events_RefreshesSnapshotBeforeForwarding is the regression
// seal for the real root cause behind "cluster recreated externally
// doesn't show up for up to 60s": a watch source's own internal
// prev/curr diff (e.g. KubeconfigFileSource re-enumerating after an
// fsnotify fire) is entirely separate from Manager.discovered, which only
// gets rebuilt by Refresh() — previously triggered only by an explicit
// cluster mutation or the 60s defensive tick, never by a raw watch event.
// The frontend's SSE handler calls fetchSnapshot() (GET /api/v1/presence,
// i.e. Snapshot()) the instant it receives *any* event, trusting that the
// backend's canonical state already reflects what triggered the event.
// If Snapshot() hasn't been refreshed yet, that immediate re-fetch
// silently returns stale pre-change data and the UI looks like nothing
// happened — until the next 60s tick catches up.
func TestManager_Events_RefreshesSnapshotBeforeForwarding(t *testing.T) {
	a := &fakeSource{clusters: nil, events: make(chan DiscoveryEvent, 4)}
	m := NewManager([]DiscoverySource{a})
	if err := m.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(m.Snapshot().Discovered) != 0 {
		t.Fatal("expected empty snapshot before the source has anything")
	}

	// Simulate what KubeconfigFileSource does: the underlying data changed
	// (recreated cluster now enumerable) and the source emits a raw event
	// about it — WITHOUT the Manager having been told to Refresh().
	a.clusters = []DiscoveredCluster{
		{Identity: identity.LogicalIdentity{Name: "recreated", ServerURL: "https://recreated"}, Source: "kubeconfig"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	events := m.Events(ctx)
	a.events <- DiscoveryEvent{Kind: EventAdd, Cluster: a.clusters[0]}

	select {
	case <-events:
		// fall through to the assertion below — this mirrors exactly what
		// the frontend does on es.onmessage: fetch the snapshot the
		// instant an event is observed, not after some extra delay.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("event not forwarded within 500ms")
	}

	snap := m.Snapshot()
	if len(snap.Discovered) != 1 || snap.Discovered[0].Identity.Name != "recreated" {
		t.Fatalf("Snapshot() was stale at the moment the event was forwarded: %+v", snap.Discovered)
	}
}
