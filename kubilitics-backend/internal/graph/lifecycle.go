package graph

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
)

// EngineLifecycleManager applies the same REGISTERED != ACTIVE pattern
// already proven for the OverviewCache informer set
// (docs/INFORMER-LIFECYCLE-IMPLEMENTATION.md) to ClusterGraphEngine, the
// separate ~15-informer-type system backing Blast Radius (and opportunistically
// reused by the Topology v2 resource bundle and Fleet X-Ray — see Resources()
// and getGraphEngine call sites).
//
// Found during the 2026-10 verification pass
// (docs/INFORMER-LIFECYCLE-IMPLEMENTATION.md "Verification Pass" section):
// main.go used to construct and Start() a ClusterGraphEngine for EVERY
// reachable persisted cluster unconditionally, ~5s after backend boot,
// regardless of whether any consumer ever used Blast Radius for that
// cluster — live-measured at ~169 extra goroutines per registered cluster.
// It also wrote directly into the same map rest.Handler serves requests
// from, with no shared lock, racing the handler's own mutex-protected
// accessors (an unsynchronized concurrent map read/write — a Go runtime
// fatal error, not a recoverable panic, under realistic timing).
//
// This manager is a deliberately separate type from
// service.ClusterLifecycleManager, not a "duplicate for convenience": the
// resource it owns lives in a different package/layer (constructed from a
// raw kubernetes.Interface + an onRebuild callback, not a
// service.OverviewCache), is owned by rest.Handler rather than
// service.clusterService, and has different consumers (Blast Radius,
// Topology v2's opportunistic bundle reuse, Fleet X-Ray). It reuses the
// exact same correctness mechanism — one mutex per entry held for the
// entire start-or-stop operation, a permanent tombstone on removal so a
// racing activation can never resurrect a deleted cluster, and a single
// centralized sweep goroutine rather than one ticker per cluster.
type EngineLifecycleManager struct {
	mu      sync.RWMutex // protects entries map structure only (add/remove keys)
	entries map[string]*engineEntry

	// onRebuild is wired identically for every engine this manager creates
	// (today: invalidate the V1 + V2 topology caches — see main.go).
	onRebuild func(clusterID string)

	idleTTL       time.Duration
	sweepInterval time.Duration
	stopSweep     chan struct{}
	sweepDone     chan struct{}
}

type engineEntry struct {
	mu         sync.Mutex
	engine     *ClusterGraphEngine
	lastAccess time.Time
	removed    bool // true once OnClusterDisconnected has run — EnsureActive must never resurrect
}

const (
	defaultEngineIdleTTL       = 10 * time.Minute
	defaultEngineSweepInterval = 30 * time.Second
)

// NewEngineLifecycleManager constructs a manager. idleTTL<=0 uses
// defaultEngineIdleTTL. onRebuild may be nil.
func NewEngineLifecycleManager(idleTTL time.Duration, onRebuild func(clusterID string)) *EngineLifecycleManager {
	if idleTTL <= 0 {
		idleTTL = defaultEngineIdleTTL
	}
	m := &EngineLifecycleManager{
		entries:       make(map[string]*engineEntry),
		onRebuild:     onRebuild,
		idleTTL:       idleTTL,
		sweepInterval: defaultEngineSweepInterval,
		stopSweep:     make(chan struct{}),
		sweepDone:     make(chan struct{}),
	}
	go m.runSweep()
	return m
}

func (m *EngineLifecycleManager) entryFor(clusterID string) *engineEntry {
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
	e = &engineEntry{}
	m.entries[clusterID] = e
	return e
}

// EnsureActive is the only path that creates/starts a ClusterGraphEngine —
// called from Blast Radius request handlers on first real use. Idempotent
// and safe for unbounded concurrent callers on the same or different
// clusters: the per-entry mutex held for the whole create-or-reuse
// operation guarantees exactly one engine generation is ever created for a
// cluster, the same way service.ClusterLifecycleManager.EnsureActive does
// for informers.
func (m *EngineLifecycleManager) EnsureActive(ctx context.Context, clusterID string, clientset kubernetes.Interface, log *slog.Logger) (*ClusterGraphEngine, error) {
	if clientset == nil {
		return nil, fmt.Errorf("graph engine lifecycle: no clientset for cluster %s", clusterID)
	}
	e := m.entryFor(clusterID)
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.removed {
		return nil, fmt.Errorf("graph engine lifecycle: cluster %s was removed", clusterID)
	}
	e.lastAccess = time.Now()
	if e.engine != nil {
		return e.engine, nil
	}

	if log == nil {
		log = slog.Default()
	}
	engine := NewClusterGraphEngine(clusterID, clientset, log)
	if m.onRebuild != nil {
		engine.SetOnRebuild(m.onRebuild)
	}
	engine.Start(ctx)
	e.engine = engine
	return e.engine, nil
}

// Get returns the engine for a cluster if one is already active, without
// creating one — for opportunistic consumers (Topology v2's resource
// bundle, Fleet X-Ray) that reuse the engine's cache when available but
// must never themselves trigger the ~15-informer-type startup cost. A hit
// still counts as real use and refreshes lastAccess.
func (m *EngineLifecycleManager) Get(clusterID string) *ClusterGraphEngine {
	e := m.entryFor(clusterID)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.engine != nil {
		e.lastAccess = time.Now()
	}
	return e.engine
}

// ForceIdleAndSweepForTest is test-only instrumentation (docs/ENGINE-
// LIFECYCLE-SOAK-INVESTIGATION.md): forces every known entry's lastAccess
// into the past by more than ttl, then runs exactly one sweep pass,
// deterministically — no behavior change, no new goroutines, no
// production code path calls this. Exists only so cross-package
// investigation tests (internal/service) can drive a deterministic sweep
// without sleeping for the real TTL.
func (m *EngineLifecycleManager) ForceIdleAndSweepForTest(ttl time.Duration) {
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

// ActiveClusterIDs returns the cluster IDs that currently have a running
// engine (not merely registered/tombstoned). For background consumers like
// autopilot.Scheduler.runAll that used to iterate every reachable
// registered cluster (when engines were eager-started for all of them) and
// must now only iterate clusters someone has actually activated — the
// whole point of removing the eager start.
func (m *EngineLifecycleManager) ActiveClusterIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.entries))
	for id, e := range m.entries {
		e.mu.Lock()
		active := e.engine != nil
		e.mu.Unlock()
		if active {
			ids = append(ids, id)
		}
	}
	return ids
}

// ActiveEngines returns a snapshot (clusterID -> engine) of every
// currently-active engine, for bulk read-only consumers (Fleet X-Ray
// dashboard/template-scoring) that previously iterated the whole raw map
// under its own lock. Safe to iterate without holding any manager lock —
// each entry's Snapshot()/Status() call is already lock-free (atomic.Value).
func (m *EngineLifecycleManager) ActiveEngines() map[string]*ClusterGraphEngine {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]*ClusterGraphEngine, len(m.entries))
	for id, e := range m.entries {
		e.mu.Lock()
		engine := e.engine
		e.mu.Unlock()
		if engine != nil {
			out[id] = engine
		}
	}
	return out
}

// OnClusterConnected implements rest.ClusterLifecycleHook. Deliberately
// does NOT start an engine (that would reintroduce the eager-start defect
// this manager exists to fix). On reconnect, a previously-active engine is
// bound to the OLD client/clientset — this stops and clears it so the next
// Blast Radius request lazily creates a fresh engine against the NEW
// client, rather than silently serving a stale graph from an abandoned
// connection forever (the same "old client -> old generation" hazard
// VALID-04 already fixed for informers).
func (m *EngineLifecycleManager) OnClusterConnected(_ kubernetes.Interface, clusterID string) error {
	e := m.entryFor(clusterID)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.engine != nil {
		e.engine.Stop()
		e.engine = nil
	}
	return nil
}

// OnClusterDisconnected implements rest.ClusterLifecycleHook. Stops any
// running engine and permanently tombstones the entry so a racing
// EnsureActive call (e.g. an in-flight Blast Radius request that started
// before removal) cannot resurrect it — same reasoning as
// service.ClusterLifecycleManager.Remove.
func (m *EngineLifecycleManager) OnClusterDisconnected(clusterID string) {
	e := m.entryFor(clusterID)
	e.mu.Lock()
	if e.engine != nil {
		e.engine.Stop()
		e.engine = nil
	}
	e.removed = true
	e.mu.Unlock()
}

// Shutdown stops the centralized TTL sweep goroutine. Does not stop any
// currently-active engine — that remains the backend process lifecycle's
// responsibility (unchanged from today's behavior for OverviewCache).
func (m *EngineLifecycleManager) Shutdown() {
	close(m.stopSweep)
	<-m.sweepDone
}

func (m *EngineLifecycleManager) runSweep() {
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

// safeSweepOnce wraps sweepOnce with panic recovery per-tick (not once for
// the whole goroutine) so a single bad sweep degrades to "skip this tick, log
// it, try again next interval" instead of permanently killing the ONE
// centralized idle-engine-cleanup goroutine for the entire fleet — or,
// unrecovered, crashing the whole backend process (docs/ai/ARCHITECTURE.md:
// every goroutine needs this).
func (m *EngineLifecycleManager) safeSweepOnce() {
	defer func() {
		if r := recover(); r != nil {
			slog.Default().Error("panic in blast-radius engine lifecycle sweep tick — will retry next interval", "error", r)
		}
	}()
	m.sweepOnce()
}

func (m *EngineLifecycleManager) sweepOnce() {
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
		idle := !e.removed && e.engine != nil && time.Since(e.lastAccess) >= m.idleTTL
		if !idle {
			e.mu.Unlock()
			continue
		}
		// Still holding e.mu for the whole stop — a concurrent EnsureActive
		// for the same cluster simply blocks on this lock until the stop
		// below finishes, then creates a fresh engine. Same race-proofing
		// as service.ClusterLifecycleManager.sweepOnce.
		e.engine.Stop()
		e.engine = nil
		e.mu.Unlock()
	}
}
