package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kubilitics/kubilitics-backend/internal/k8s"
)

// ClusterLifecycleManager implements the hybrid informer lifecycle
// (docs/INFORMER-LIFECYCLE-IMPLEMENTATION.md): REGISTERED CLUSTER ≠ ACTIVE
// CLUSTER. Registration (AddCluster, backend startup via LoadClustersFromRepo)
// persists metadata and a live K8s client but does NOT start the 27-informer
// set (docs/INFORMER-LIFECYCLE-INVESTIGATION.md). Informers start the first
// time a feature actually needs informer-backed data (EnsureActive, called
// from the two choke points every consumer already goes through —
// GetInformerManager and GetOverview) and stop after IdleTTL of no such
// calls, via a single centralized sweep goroutine (not one ticker per
// cluster — avoids the ticker-leak class of bug this engagement has
// repeatedly found and fixed, e.g. VALID-04).
//
// Concurrency model: one *sync.Mutex per cluster (entry.mu), not a single
// global lock — concurrent activation of different clusters never blocks
// each other. For a single cluster, Start and Stop are both performed WHILE
// HOLDING that cluster's own entry.mu for the full operation: this is the
// entire correctness mechanism. It guarantees, by plain mutual exclusion
// (no additional coalescing primitive needed):
//   - exactly one informer generation can ever exist for a cluster at a time
//   - a caller either observes the fully-completed prior state (lock
//     acquired before a transition started) or the fully-completed new state
//     (lock acquired after it finished) — never a half-transitioned,
//     "stale generation" state
//   - 100 concurrent EnsureActive calls for the same cluster: the first to
//     acquire entry.mu performs the actual start; all others block, then
//     each sees state==StateActive once they acquire the lock and return
//     immediately without starting a second generation
//   - the TTL sweep, to stop an idle cluster, must also acquire entry.mu —
//     so it can never interleave with an in-flight EnsureActive for that
//     same cluster; whichever acquires the lock first completes atomically
//     before the other proceeds
type ClusterLifecycleManager struct {
	cache *OverviewCache

	mu      sync.RWMutex // protects entries map structure only (add/remove keys)
	entries map[string]*clusterLifecycleEntry

	idleTTL       time.Duration
	sweepInterval time.Duration
	stopSweep     chan struct{}
	sweepDone     chan struct{}
}

// LifecycleState is a cluster's informer-lifecycle state, independent of its
// registration/connectivity state (models.Cluster.Status) which is tracked
// separately and unaffected by this manager.
type LifecycleState int32

const (
	// StateIdle: no informers running. Includes "never activated" (the
	// post-registration default) and "activated once, then TTL-expired."
	StateIdle LifecycleState = iota
	// StateActive: informers running for this cluster right now.
	StateActive
)

func (s LifecycleState) String() string {
	switch s {
	case StateActive:
		return "active"
	default:
		return "idle"
	}
}

type clusterLifecycleEntry struct {
	mu         sync.Mutex
	state      LifecycleState
	generation uint64
	lastAccess time.Time
	im         *k8s.InformerManager
	removed    bool // true once RemoveCluster has run — EnsureActive must never resurrect
}

const (
	defaultIdleTTL       = 10 * time.Minute
	defaultSweepInterval = 30 * time.Second
)

// NewClusterLifecycleManager constructs a manager bound to the given
// OverviewCache (the existing, unmodified Start/StopClusterCache
// implementation — this manager decides WHEN to call them, not how they
// work). idleTTL<=0 uses defaultIdleTTL.
func NewClusterLifecycleManager(cache *OverviewCache, idleTTL time.Duration) *ClusterLifecycleManager {
	if idleTTL <= 0 {
		idleTTL = defaultIdleTTL
	}
	m := &ClusterLifecycleManager{
		cache:         cache,
		entries:       make(map[string]*clusterLifecycleEntry),
		idleTTL:       idleTTL,
		sweepInterval: defaultSweepInterval,
		stopSweep:     make(chan struct{}),
		sweepDone:     make(chan struct{}),
	}
	go m.runSweep()
	return m
}

func (m *ClusterLifecycleManager) entryFor(clusterID string) *clusterLifecycleEntry {
	m.mu.RLock()
	e, ok := m.entries[clusterID]
	m.mu.RUnlock()
	if ok {
		return e
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.entries[clusterID]; ok { // re-check under write lock
		return e
	}
	e = &clusterLifecycleEntry{state: StateIdle}
	m.entries[clusterID] = e
	return e
}

// EnsureActive is the single entry point every informer-backed-data consumer
// calls (wired into GetInformerManager and GetOverview in cluster_service.go
// — not into every individual feature handler, since both of those are
// already the choke points all such handlers go through). Idempotent, safe
// for unbounded concurrent callers on the same or different clusters. Bumps
// lastAccess on every call, including when already active, so TTL correctly
// measures "no consumer has asked for this cluster's data recently," not
// just "time since activation."
//
// Registration/reconnect/startup paths do NOT call this — they persist
// metadata and a live client only (see LoadClustersFromRepo, AddCluster).
// This is intentionally the ONLY path that starts informers, so "registered
// ≠ active" is a structural guarantee, not a convention callers must
// remember.
func (m *ClusterLifecycleManager) EnsureActive(ctx context.Context, clusterID string, client *k8s.Client) (*k8s.InformerManager, error) {
	if client == nil {
		return nil, fmt.Errorf("cluster lifecycle: no client for cluster %s", clusterID)
	}
	e := m.entryFor(clusterID)
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.removed {
		return nil, fmt.Errorf("cluster lifecycle: cluster %s was removed", clusterID)
	}
	e.lastAccess = time.Now()
	if e.state == StateActive {
		return e.im, nil
	}

	// StateIdle -> activate now, while holding this cluster's own lock (see
	// type doc comment for why this alone is the correctness mechanism).
	if err := m.cache.StartClusterCache(ctx, clusterID, client); err != nil {
		return nil, err
	}
	e.im = m.cache.GetInformerManager(clusterID)
	e.state = StateActive
	e.generation++
	lifecycleLog("activate", clusterID, e.generation, "ensure-active", 0)
	return e.im, nil
}

// Reconnected implements the VALID-04 invariant (OLD CLIENT → OLD GENERATION
// → STOP → NEW CLIENT → NEW GENERATION, never new client + old informers)
// through this manager, so the manager's own state stays authoritative —
// every start/stop goes through entry.mu, including reconnects, with no
// second, parallel bookkeeping path. Reconnect eagerly reactivates (does not
// leave the cluster Idle for the next consumer to lazily reactivate):
// a user explicitly reconnecting is, in practice, about to use the cluster,
// and the pre-existing UX contract (reconnect succeeds → cluster usable
// immediately) must not regress.
//
// Deliberately does NOT check e.removed the way EnsureActive does: the
// caller chain here (ReconnectCluster → applyAndStoreClient → Reconnected)
// already has its OWN, independently-tested, VALID-01-established
// reconnect-races-removal rollback — finishReconnect re-checks repo
// existence after this returns and calls lifecycle.Remove again if the row
// is gone, tearing back down whatever was just (re)started. That existing
// mechanism is the single source of truth for this specific race
// (TestVALID04_ReconnectRacesRemove_NoResurrectionNoStaleCache asserts
// ReconnectCluster still returns nil error in this exact scenario — adding
// a proactive block here would silently change that already-tested,
// already-correct contract for no added safety, since the end state is
// identical either way: finishReconnect's rollback runs regardless).
func (m *ClusterLifecycleManager) Reconnected(ctx context.Context, clusterID string, client *k8s.Client) error {
	e := m.entryFor(clusterID)
	e.mu.Lock()
	defer e.mu.Unlock()

	// Stop unconditionally first — StopClusterCache is a documented no-op
	// when nothing is running (VALID-04's existing guarantee), so this is
	// safe whether the prior state was Active or Idle.
	m.cache.StopClusterCache(clusterID)
	e.im = nil

	if err := m.cache.StartClusterCache(ctx, clusterID, client); err != nil {
		e.state = StateIdle
		return err
	}
	e.im = m.cache.GetInformerManager(clusterID)
	e.state = StateActive
	e.generation++
	e.lastAccess = time.Now()
	lifecycleLog("activate", clusterID, e.generation, "reconnect", 0)
	return nil
}

// Remove tears down any running informers and permanently tombstones the
// entry so a racing EnsureActive/Reconnected call (e.g. from an in-flight
// request that started before RemoveCluster ran — the exact VALID-01
// reconnect-races-removal scenario, now also possible at this layer)
// cannot resurrect it.
//
// Deliberately does NOT delete the entry from m.entries: an earlier version
// did, which left a real race — a concurrent Reconnected() call could, after
// Remove's own e.mu.Unlock() but before Remove deleted the map key, call
// entryFor(clusterID) and (finding no entry) create a BRAND NEW one with
// removed=false, resurrecting informers for a cluster whose DB row is
// already gone. Cluster IDs are server-generated UUIDs (uuid.New(), never
// reused across AddCluster calls — see addClusterWithSource), so keeping a
// tombstoned entry forever is safe and bounded (one small struct per
// ever-removed cluster, the same order of magnitude as the clusters table
// itself) — this is what makes checking e.removed under the SAME entry's
// mu, rather than relying on map-key presence, an actual guarantee instead
// of a best-effort one.
func (m *ClusterLifecycleManager) Remove(clusterID string) {
	e := m.entryFor(clusterID)
	e.mu.Lock()
	wasActive := e.state == StateActive
	m.cache.StopClusterCache(clusterID)
	e.im = nil
	e.state = StateIdle
	e.removed = true
	gen := e.generation
	e.mu.Unlock()

	if wasActive {
		lifecycleLog("deactivate", clusterID, gen, "removed", 0)
	}
}

// Shutdown stops the centralized TTL sweep goroutine. Does not stop any
// currently-active cluster caches — that remains the backend process
// lifecycle's responsibility (unchanged from today).
func (m *ClusterLifecycleManager) Shutdown() {
	close(m.stopSweep)
	<-m.sweepDone
}

// runSweep is the ONE background goroutine for the entire manager — not one
// per cluster. Explicitly required by the implementation brief ("no global
// ticker per cluster if avoidable... prefer a centralized lifecycle
// scheduler"). Each tick scans every known cluster; the actual Stop for an
// idle one happens while holding that cluster's own entry.mu, so it cannot
// race a concurrent EnsureActive for the same cluster (see type doc).
func (m *ClusterLifecycleManager) runSweep() {
	defer close(m.sweepDone)
	ticker := time.NewTicker(m.sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopSweep:
			return
		case <-ticker.C:
			m.safeSweepOnce()
		}
	}
}

// safeSweepOnce wraps sweepOnce with panic recovery per-tick (not once for the
// whole goroutine) so a single bad sweep degrades to "skip this tick, log it,
// try again next interval" instead of permanently killing the ONE centralized
// idle-cluster-cleanup goroutine for the entire fleet — or, unrecovered,
// crashing the whole backend process (docs/ai/ARCHITECTURE.md: every
// goroutine needs this).
func (m *ClusterLifecycleManager) safeSweepOnce() {
	defer func() {
		if r := recover(); r != nil {
			slog.Default().Error("panic in cluster lifecycle sweep tick — will retry next interval", "error", r)
		}
	}()
	m.sweepOnce()
}

func (m *ClusterLifecycleManager) sweepOnce() {
	m.mu.RLock()
	ids := make([]string, 0, len(m.entries))
	for id := range m.entries {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	for _, id := range ids {
		m.mu.RLock()
		e, ok := m.entries[id]
		m.mu.RUnlock()
		if !ok {
			continue // removed between the snapshot and now
		}

		e.mu.Lock()
		idle := !e.removed && e.state == StateActive && time.Since(e.lastAccess) >= m.idleTTL
		if !idle {
			e.mu.Unlock()
			continue
		}
		// Still holding e.mu for the whole stop: see type doc — this is
		// what makes the TTL-vs-active-request race impossible. A request
		// that calls EnsureActive concurrently simply blocks on this same
		// lock until the stop below finishes, then reactivates fresh.
		idleFor := time.Since(e.lastAccess)
		m.cache.StopClusterCache(id)
		e.im = nil
		e.state = StateIdle
		gen := e.generation
		e.mu.Unlock()

		lifecycleLog("deactivate", id, gen, "idle-ttl", idleFor)
	}
}

// ForceIdleAndSweepForTest is test-only instrumentation (docs/ENGINE-
// LIFECYCLE-SOAK-INVESTIGATION.md): forces every known entry's lastAccess
// into the past by more than ttl, then runs exactly one sweep pass,
// deterministically — no behavior change, no production code path calls
// this. Exists only so cross-package investigation tests (an external
// _test package importing both this package and internal/graph, to avoid
// the service->graph->otel->events->service import cycle) can drive a
// deterministic sweep without sleeping for the real TTL.
func (m *ClusterLifecycleManager) ForceIdleAndSweepForTest(ttl time.Duration) {
	m.mu.RLock()
	ids := make([]string, 0, len(m.entries))
	for id := range m.entries {
		ids = append(ids, id)
	}
	m.mu.RUnlock()
	for _, id := range ids {
		m.mu.RLock()
		e, ok := m.entries[id]
		m.mu.RUnlock()
		if !ok {
			continue
		}
		e.mu.Lock()
		e.lastAccess = time.Now().Add(-ttl - time.Second)
		e.mu.Unlock()
	}
	m.sweepOnce()
}

// lifecycleLog is the structured-ish observability hook required by
// docs/INFORMER-LIFECYCLE-IMPLEMENTATION.md §14 — cluster ID, state
// transition, generation, reason, and duration where applicable. Never logs
// credentials/kubeconfig contents (only ever passed a clusterID, which is a
// backend-generated UUID, not a secret). Uses the same fmt.Printf-based
// convention already established throughout cluster_service.go (e.g.
// "[AddCluster] ...") rather than introducing a new logging dependency for
// this one file.
func lifecycleLog(transition, clusterID string, generation uint64, reason string, idleFor time.Duration) {
	if idleFor > 0 {
		fmt.Printf("[ClusterLifecycle] %s cluster=%s generation=%d reason=%s idle_for=%s\n",
			transition, clusterID, generation, reason, idleFor.Round(time.Second))
		return
	}
	fmt.Printf("[ClusterLifecycle] %s cluster=%s generation=%d reason=%s\n",
		transition, clusterID, generation, reason)
}
