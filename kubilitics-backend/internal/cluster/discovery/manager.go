package discovery

import (
	"context"
	"sync"
	"time"

	"github.com/kubilitics/kubilitics-backend/internal/cluster/presence"
)

// ReachabilityStatus is a point-in-time reachability check result for one
// registered cluster, keyed by its backend session/cluster ID.
type ReachabilityStatus struct {
	Reachable     bool
	LastCheckedAt time.Time // zero if never checked
	LastSuccessAt time.Time // zero if never successful
	LastError     string    // empty if the last check succeeded or none has run
}

// ReachabilityChecker reports live reachability for a registered cluster by
// its backend session ID. HEALTH-1/HEALTH-2 (docs/PRODUCTION-RELIABILITY-
// AUDIT.md): wired to ClusterService's live client registry (populated only
// on a successful connection test, cleared on disconnect/removal) and its
// per-client health tracking — not a new, independent health concept.
type ReachabilityChecker func(sessionID string) ReachabilityStatus

// Manager composes multiple DiscoverySources into a single deduplicated
// PresenceSnapshot. First-wins dedup by identity key — earlier sources
// in the slice take precedence.
type Manager struct {
	sources     []DiscoverySource
	mu          sync.RWMutex
	discovered  []DiscoveredCluster
	byKey       map[string]int // key → index in discovered
	isReachable ReachabilityChecker
}

func NewManager(sources []DiscoverySource) *Manager {
	return &Manager{sources: sources, byKey: map[string]int{}}
}

// SetReachabilityChecker wires a live-reachability source. Call this once
// after both the Manager and the cluster service exist (main.go constructs
// them in a fixed order; this setter avoids a circular import between the
// discovery and service packages). Safe to call concurrently with Snapshot().
// Until called, Snapshot() reports every registered cluster as NOT reachable
// — HEALTH-1's whole point is that an unverified cluster must never be
// presented as reachable, so "no checker wired yet" must fail closed, not
// default to the old hardcoded-true behavior.
func (m *Manager) SetReachabilityChecker(checker ReachabilityChecker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isReachable = checker
}

// Refresh enumerates every source and rebuilds the snapshot. Called on
// startup and whenever a quorum of events warrants a full re-sync.
func (m *Manager) Refresh(ctx context.Context) error {
	merged := []DiscoveredCluster{}
	byKey := map[string]int{}
	for _, s := range m.sources {
		enum, err := s.Enumerate(ctx)
		if err != nil {
			// Do NOT abort — one broken source should not blank out others.
			continue
		}
		for _, c := range enum {
			k := c.Identity.Key()
			if idx, seen := byKey[k]; seen {
				// A later source can enrich an entry an earlier source
				// provided — most importantly, ManualSource carries the
				// SessionID + Provider for clusters that are ALSO in a
				// kubeconfig file. Without this merge, `kind-kubilitics-test`
				// (seen first by KubeconfigFileSource → no SessionID) would
				// mask the registered state from ManualSource and stay
				// stuck in Discovered-only forever.
				existing := merged[idx]
				if existing.SessionID == "" && c.SessionID != "" {
					existing.SessionID = c.SessionID
				}
				if existing.Provider == "" && c.Provider != "" {
					existing.Provider = c.Provider
				}
				if existing.ContextName == "" && c.ContextName != "" {
					existing.ContextName = c.ContextName
				}
				if existing.KubeconfigPath == "" && c.KubeconfigPath != "" {
					existing.KubeconfigPath = c.KubeconfigPath
				}
				merged[idx] = existing
				continue
			}
			byKey[k] = len(merged)
			merged = append(merged, c)
		}
	}
	m.mu.Lock()
	m.discovered = merged
	m.byKey = byKey
	m.mu.Unlock()
	return nil
}

// Snapshot returns a copy-safe view. Entries that came from a source
// that carries a SessionID (today: ManualSource — the cluster is already
// in the backend DB with an assigned UUID) are promoted into
// Registered so the frontend can use session_id for cluster-scoped API
// calls. Entries without a SessionID stay in Discovered only.
//
// Connected is the subset of Registered the reachability checker reports
// as currently healthy. This is not a separate session-tracking concept:
// SetReachabilityChecker's doc comment is explicit that `Reachable` already
// comes from ClusterService's live client registry (populated on a
// successful connection, cleared on disconnect/removal) — i.e. "reachable"
// and "has an active backend session" are the same underlying fact. A
// prior version of this method left Connected hardcoded empty pending a
// separate ConnectionManager; that was unnecessary since the data already
// exists here.
func (m *Manager) Snapshot() presence.Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	now := time.Now().Format(time.RFC3339)
	disc := make([]presence.DiscoveredCluster, 0, len(m.discovered))
	reg := make([]presence.RegisteredCluster, 0, len(m.discovered))
	conn := make([]presence.ConnectedCluster, 0, len(m.discovered))
	for _, c := range m.discovered {
		pd := presence.DiscoveredCluster{
			Identity:   c.Identity,
			Source:     c.Source,
			LastSeenAt: now,
		}
		disc = append(disc, pd)
		if c.SessionID != "" {
			rc := presence.RegisteredCluster{
				DiscoveredCluster: pd,
				RegisteredAt:      now,
				Reachable:         false, // fail closed until a checker says otherwise — see SetReachabilityChecker
				SessionID:         c.SessionID,
				Provider:          c.Provider,
				KubeconfigPath:    c.KubeconfigPath, // VALID-05
				ContextName:       c.ContextName,    // VALID-05
			}
			if m.isReachable != nil {
				status := m.isReachable(c.SessionID)
				rc.Reachable = status.Reachable
				if !status.LastCheckedAt.IsZero() {
					rc.LastCheckedAt = status.LastCheckedAt.Format(time.RFC3339)
				}
				if !status.LastSuccessAt.IsZero() {
					rc.LastSuccessAt = status.LastSuccessAt.Format(time.RFC3339)
				}
				rc.LastError = status.LastError
			}
			reg = append(reg, rc)
			if rc.Reachable {
				connectedAt := rc.LastSuccessAt
				if connectedAt == "" {
					connectedAt = now
				}
				conn = append(conn, presence.ConnectedCluster{
					RegisteredCluster: rc,
					ConnectedAt:       connectedAt,
				})
			}
		}
	}
	return presence.Snapshot{
		Discovered: disc,
		Registered: reg,
		Connected:  conn,
	}
}

// Events fans in all sources' Watch() channels. The manager filters out
// events that would be duplicates of already-known identities.
//
// Each source's own Watch() loop (e.g. KubeconfigFileSource's fsnotify
// handler) maintains its OWN prev/curr diff entirely separately from
// m.discovered, which is only rebuilt by Refresh() — previously called
// only on an explicit cluster mutation or the 60s defensive tick, never
// in response to a raw watch event. Callers (the SSE handler, and the
// frontend's es.onmessage) treat "an event arrived" as "go re-fetch
// Snapshot() now" — so without refreshing here first, that immediate
// re-fetch silently returned stale pre-change data for up to 60s. This
// is the concrete mechanism behind "cluster recreated outside the app
// doesn't show up." Refreshing re-enumerates every source (cheap,
// already the steady-state cost of the periodic tick) before the event
// is forwarded, so by the time a subscriber reacts, Snapshot() already
// reflects it.
func (m *Manager) Events(ctx context.Context) <-chan DiscoveryEvent {
	out := make(chan DiscoveryEvent, 32)
	var wg sync.WaitGroup
	for _, s := range m.sources {
		ch, err := s.Watch(ctx)
		if err != nil {
			continue // source doesn't support watch
		}
		wg.Add(1)
		go func(c <-chan DiscoveryEvent) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case ev, ok := <-c:
					if !ok {
						return
					}
					_ = m.Refresh(ctx) // see doc comment above; Refresh never actually errors today
					select {
					case out <- ev:
					case <-ctx.Done():
						return
					}
				}
			}
		}(ch)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
