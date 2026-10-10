package service

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/kubilitics/kubilitics-backend/internal/config"
	"github.com/kubilitics/kubilitics-backend/internal/k8s"
	"github.com/kubilitics/kubilitics-backend/internal/models"
	"github.com/kubilitics/kubilitics-backend/internal/repository"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
	"golang.org/x/time/rate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
)

const defaultMaxClusters = 100

// cachedTypedItems reads resourceType from the informer cache and converts
// each item to T, skipping any item that fails to convert. Returns
// (nil, false) when the cache is unavailable, not yet synced, or genuinely
// has no cached items — callers fall back to a live API call in that case.
func cachedTypedItems[T any](im *k8s.InformerManager, resourceType string) ([]T, bool) {
	if im == nil || !im.HasSynced() {
		return nil, false
	}
	cached, ok := im.ListFromCache(resourceType, "", metav1.ListOptions{})
	if !ok || cached == nil {
		return nil, false
	}
	items := make([]T, 0, len(cached.Items))
	skipped := 0
	for _, u := range cached.Items {
		var item T
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &item); err == nil {
			items = append(items, item)
		} else {
			skipped++
		}
	}
	if skipped > 0 {
		log.Printf("cachedTypedItems[%T]: skipped %d/%d cached %s items due to conversion errors", *new(T), skipped, len(cached.Items), resourceType)
	}
	return items, true
}

// ErrClusterLimitReached is returned when a cluster registration would exceed the configured maxClusters limit.
// Carries the current count and limit for structured error responses.
type ErrClusterLimitReached struct {
	Current int
	Max     int
}

func (e *ErrClusterLimitReached) Error() string {
	return fmt.Sprintf("cluster limit reached (%d/%d); cannot add more clusters", e.Current, e.Max)
}

// ClusterService manages Kubernetes clusters
type ClusterService interface {
	ListClusters(ctx context.Context) ([]*models.Cluster, error)
	GetCluster(ctx context.Context, id string) (*models.Cluster, error)
	AddCluster(ctx context.Context, kubeconfigPath, contextName string) (*models.Cluster, error)
	// AddClusterFromBytes adds a cluster from raw kubeconfig content (e.g., uploaded via browser).
	// It writes the content to ~/.kubilitics/kubeconfigs/<context>.yaml and delegates to AddCluster.
	// The cluster is fully persisted and provider-detected, same as AddCluster.
	AddClusterFromBytes(ctx context.Context, kubeconfigBytes []byte, contextName string) (*models.Cluster, error)
	RemoveCluster(ctx context.Context, id string) error
	TestConnection(ctx context.Context, id string) error
	GetClusterSummary(ctx context.Context, id string) (*models.ClusterSummary, error)
	// LoadClustersFromRepo restores K8s clients from persisted clusters (call on startup).
	LoadClustersFromRepo(ctx context.Context) error
	// GetClient returns the K8s client for a cluster (for internal use by topology, resources, etc.).
	GetClient(id string) (*k8s.Client, error)
	// GetOrReconnectClient is like GetClient, but if the pool is cold for this
	// cluster it will lazily call ReconnectCluster (coalesced via singleflight,
	// bounded by a short timeout, with a negative cache for recent failures).
	// This is the client lookup that HTTP handlers should use on the fallback path.
	GetOrReconnectClient(ctx context.Context, id string) (*k8s.Client, error)
	// HasMetalLB returns true if MetalLB CRDs (ipaddresspools, bgppeers) are installed in the cluster.
	HasMetalLB(ctx context.Context, id string) (bool, error)
	// DiscoverClusters scans the configured kubeconfig for contexts not yet in the repository.
	DiscoverClusters(ctx context.Context) ([]*models.Cluster, error)
	// GetOverview returns the cached overview for a cluster if available.
	GetOverview(clusterID string) (*models.ClusterOverview, bool)
	// Subscribe returns a channel and unsubscribe function for real-time overview updates.
	// Returns ErrTooManyListeners if the per-cluster listener limit is reached.
	Subscribe(clusterID string) (chan *models.ClusterOverview, func(), error)
	// ReconnectCluster resets the circuit breaker and forces a fresh K8s client connection.
	// Call this when the user explicitly requests reconnect or the cluster status page is opened.
	ReconnectCluster(ctx context.Context, id string) (*models.Cluster, error)
	// GetInformerManager returns the InformerManager for a cluster if available.
	// Used by the REST handler to serve resource lists from in-memory cache (<1ms)
	// instead of making direct K8s API calls (~200-2000ms). Returns nil if not started.
	GetInformerManager(clusterID string) *k8s.InformerManager
}

// K8sClientFactory creates a k8s client from kubeconfig path and context. Used in tests to inject a fake client.
// When nil, AddCluster uses k8s.NewClient.
type K8sClientFactory func(kubeconfigPath, contextName string) (*k8s.Client, error)

type clusterService struct {
	mu                 sync.RWMutex
	repo               repository.ClusterRepository
	clients            map[string]*k8s.Client // id -> live K8s client
	overviewCache      *OverviewCache
	// lifecycle implements the hybrid informer lifecycle
	// (docs/INFORMER-LIFECYCLE-IMPLEMENTATION.md): decides WHEN
	// overviewCache's Start/StopClusterCache run. Registration/reconnect/
	// startup no longer call overviewCache directly — see EnsureActive's
	// doc comment for why this is the only path that starts informers.
	lifecycle          *ClusterLifecycleManager
	maxClusters        int
	k8sTimeout         time.Duration // timeout for outbound K8s API calls; 0 = use request context only
	k8sRateLimitPerSec float64
	k8sRateLimitBurst  int
	clientFactory      K8sClientFactory // optional; tests only

	// reconnectSF coalesces parallel lazy-reconnect attempts for the same cluster id.
	// Without it, a single page load (which fires ~10 parallel API requests) would
	// start ~10 parallel ReconnectCluster calls — each runs TestConnection, creates
	// a client, and starts informers. The losers of the map-write race leak their
	// client handles and informer watches against the apiserver.
	reconnectSF singleflight.Group
	// reconnectFailCache remembers recent failures so we don't retry a known-broken
	// cluster on every request; keys are cluster ids, values are failure timestamps.
	reconnectFailCache sync.Map
}

// reconnectNegativeCacheTTL is how long a failed lazy-reconnect is remembered
// before we're willing to try again. Keeps offline clusters from hanging every
// request for 8 seconds each.
const reconnectNegativeCacheTTL = 10 * time.Second

// reconnectTimeout bounds the per-attempt wall clock when lazy-reconnecting a
// cluster from the request hot path. Shorter than the startup load budget on
// purpose: a user is waiting on the other end of this call.
const reconnectTimeout = 3 * time.Second

// GetOrReconnectClient returns a live k8s client for id, building one on demand
// if the cluster has no client in the pool (e.g. it was persisted but offline
// when the backend started, or became reachable after startup).
//
// Safe to call from many request goroutines concurrently — singleflight ensures
// only one reconnect runs at a time per cluster, and a short negative cache
// prevents hammering a known-broken cluster.
func (s *clusterService) GetOrReconnectClient(ctx context.Context, id string) (*k8s.Client, error) {
	if client, err := s.GetClient(id); err == nil {
		return client, nil
	}

	// Honor the negative cache: if a recent reconnect failed, fail fast with the
	// cached error instead of paying the timeout cost again.
	if v, ok := s.reconnectFailCache.Load(id); ok {
		if entry, ok := v.(reconnectFailure); ok {
			if time.Since(entry.at) < reconnectNegativeCacheTTL {
				return nil, entry.err
			}
			s.reconnectFailCache.Delete(id)
		}
	}

	// Coalesce parallel callers so the expensive reconnect only runs once per id.
	_, err, _ := s.reconnectSF.Do(id, func() (interface{}, error) {
		// Double-check under singleflight: another caller may have populated the
		// pool while we were queued. Avoid re-reconnecting.
		if client, err := s.GetClient(id); err == nil {
			return client, nil
		}
		rctx, cancel := context.WithTimeout(ctx, reconnectTimeout)
		defer cancel()
		if _, rerr := s.ReconnectCluster(rctx, id); rerr != nil {
			s.reconnectFailCache.Store(id, reconnectFailure{at: time.Now(), err: rerr})
			return nil, rerr
		}
		return s.GetClient(id)
	})
	if err != nil {
		return nil, err
	}
	return s.GetClient(id)
}

type reconnectFailure struct {
	at  time.Time
	err error
}

// clusterInfoString safely extracts a string value from a GetClusterInfo result map.
// Returns "" if the key is absent or the value is not a string.
func clusterInfoString(info map[string]interface{}, key string) string {
	v, _ := info[key].(string)
	return v
}

// serverURLFromKubeconfig reads the API server URL a context declares,
// straight off disk — no network call, so it's available even when the
// cluster turns out to be unreachable. Mirrors
// internal/cluster/discovery/kubeconfig_source.go's Enumerate(), which reads
// the identical field for the same purpose (discovery identity); kept here
// rather than imported to avoid a cross-package dependency for one field
// read. Returns "" on any parse failure or missing context/cluster — callers
// already treat "" as "unknown," the same as before this helper existed.
func serverURLFromKubeconfig(kubeconfigPath, contextName string) string {
	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return ""
	}
	kctx, ok := cfg.Contexts[contextName]
	if !ok || kctx == nil {
		return ""
	}
	cluster, ok := cfg.Clusters[kctx.Cluster]
	if !ok || cluster == nil {
		return ""
	}
	return cluster.Server
}

// clusterInfoInt safely extracts an int value from a GetClusterInfo result map.
// Returns 0 if the key is absent or the value is not an int.
func clusterInfoInt(info map[string]interface{}, key string) int {
	v, _ := info[key].(int)
	return v
}

func NewClusterService(repo repository.ClusterRepository, cfg *config.Config) ClusterService {
	return newClusterService(repo, cfg, nil)
}

// NewClusterServiceWithClientFactory is for tests: injects a client factory so AddCluster does not call real k8s.NewClient.
func NewClusterServiceWithClientFactory(repo repository.ClusterRepository, cfg *config.Config, factory K8sClientFactory) ClusterService {
	return newClusterService(repo, cfg, factory)
}

func newClusterService(repo repository.ClusterRepository, cfg *config.Config, factory K8sClientFactory) ClusterService {
	maxClusters := defaultMaxClusters
	var k8sTimeout time.Duration
	var k8sRatePerSec float64
	var k8sRateBurst int
	var idleTTL time.Duration
	if cfg != nil {
		if cfg.MaxClusters > 0 {
			maxClusters = cfg.MaxClusters
		}
		if cfg.K8sTimeoutSec > 0 {
			k8sTimeout = time.Duration(cfg.K8sTimeoutSec) * time.Second
		}
		if cfg.K8sRateLimitPerSec > 0 && cfg.K8sRateLimitBurst > 0 {
			k8sRatePerSec = cfg.K8sRateLimitPerSec
			k8sRateBurst = cfg.K8sRateLimitBurst
		}
		if cfg.ClusterIdleTTLSec > 0 {
			idleTTL = time.Duration(cfg.ClusterIdleTTLSec) * time.Second
		}
	}
	overviewCache := NewOverviewCache()
	return &clusterService{
		repo:               repo,
		clients:            make(map[string]*k8s.Client),
		overviewCache:      overviewCache,
		lifecycle:          NewClusterLifecycleManager(overviewCache, idleTTL),
		maxClusters:        maxClusters,
		k8sTimeout:         k8sTimeout,
		k8sRateLimitPerSec: k8sRatePerSec,
		k8sRateLimitBurst:  k8sRateBurst,
		clientFactory:      factory,
	}
}

// VALID-01 (docs/VALID-01-INVESTIGATION.md): backgroundReconnectBudget bounds
// the non-blocking reconnect ListClusters kicks off for a cluster with no
// live client — generous enough to let GetOrReconnectClient's own 3s
// reconnectTimeout plus a real GetClusterInfo call complete, but this budget
// is never on the critical path of any HTTP response (see
// kickBackgroundReconnect).
const backgroundReconnectBudget = 8 * time.Second

func (s *clusterService) ListClusters(ctx context.Context) ([]*models.Cluster, error) {
	clusters, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	// Identify active context from local kubeconfig to highlight in UI
	home, _ := os.UserHomeDir()
	currentContext := ""
	if home != "" {
		_, currentContext, _ = k8s.GetKubeconfigContexts(filepath.Join(home, ".kube", "config"))
	}

	// VALID-01 (docs/VALID-01-INVESTIGATION.md, Phase: post-Release-Gate
	// validation): previously every cluster — including ones with no live
	// client at all — was enriched inside this function's wg.Wait() group,
	// with a reconnect attempt (tryReconnectCluster) bounded only by a 10s
	// per-call timeout and NO caching between calls. One unreachable cluster
	// therefore delayed this entire response — including already-known-good
	// data for every OTHER, healthy cluster — by up to 10 seconds, on EVERY
	// single call (live-reproduced: GET /api/v1/clusters took 10.015s with
	// 1 healthy + 1 unreachable cluster).
	//
	// Fix: only clusters that already have a live client are refreshed
	// synchronously inside the wg.Wait() group below — this is the
	// already-fast case (live-measured ~70-300ms) and is unchanged from
	// before. Clusters with NO live client are never added to the wait
	// group at all: this call returns their persisted, truthfully-labeled
	// last-known Status/NodeCount/NamespaceCount/LastConnected immediately
	// (never fabricated as "connected"), while a reconnect attempt is fired
	// through the existing, purpose-built GetOrReconnectClient — which
	// already provides singleflight coalescing (no duplicate reconnect
	// storms across concurrent/repeated ListClusters calls) and a 10s
	// negative-failure cache (a known-broken cluster is not re-dialed on
	// every call) — on a background context, so it is not cancelled when
	// this HTTP response is written, and updates the persisted row for the
	// *next* ListClusters call. No new reconnection/cache/timeout mechanism
	// is introduced; this reuses GetOrReconnectClient exactly as the
	// investigation recommended (Option E) instead of ListClusters'
	// previous, separate, uncached tryReconnectCluster call.
	var wg sync.WaitGroup
	for _, c := range clusters {
		c.IsCurrent = (c.Context == currentContext)

		s.mu.RLock()
		client, hasClient := s.clients[c.ID]
		s.mu.RUnlock()

		if !hasClient {
			// Return this cluster's persisted last-known state as-is (already
			// truthfully labeled connected/disconnected/error by whatever
			// previously wrote it — AddCluster, a prior successful enrichment,
			// or a prior background reconnect's correction below) and kick off
			// a non-blocking refresh for next time. Does not join wg — must
			// never be waited on by this response.
			s.kickBackgroundReconnect(c.ID)
			continue
		}

		wg.Add(1)
		go func(c *models.Cluster) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("[ListClusters] Panic in enrichment goroutine for cluster %s: %v\n", c.ID, r)
				}
				wg.Done()
			}()

			// Per-cluster timeout for background enrichment to ensure responsiveness.
			clusterCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			info, err := client.GetClusterInfo(clusterCtx)
			if err != nil {
				c.Status = clusterStatusFromError(err)
				_ = s.repo.Update(ctx, c)
				return
			}
			c.ServerURL = clusterInfoString(info, "server_url")
			c.Version = clusterInfoString(info, "version")
			c.NodeCount = clusterInfoInt(info, "node_count")
			c.NamespaceCount = clusterInfoInt(info, "namespace_count")
			c.Status = "connected"
			c.LastConnected = time.Now()
			if p, err := client.DetectProvider(clusterCtx); err == nil && p != "" {
				c.Provider = p
			}
			_ = s.repo.Update(ctx, c)

			// Informer lifecycle (docs/INFORMER-LIFECYCLE-IMPLEMENTATION.md):
			// a live client is established here, but informers are NOT
			// started eagerly — EnsureActive (called from GetInformerManager/
			// GetOverview, the choke points every consumer already goes
			// through) starts them lazily on first actual use. Registered ≠
			// Active.
		}(c)
	}
	wg.Wait()
	return clusters, nil
}

// kickBackgroundReconnect fires a non-blocking reconnect attempt for a
// cluster with no live client, reusing GetOrReconnectClient's existing
// singleflight coalescing and negative-failure cache (see its doc comment)
// rather than introducing a second, parallel reconnect implementation.
// Intentionally fire-and-forget: runs on context.Background() (not the
// triggering HTTP request's context), matching the same pattern
// cmd/server/main.go's existing 60s presence-refresh ticker already uses,
// so the attempt is not cancelled the moment the HTTP response that
// triggered it is written. Reads a FRESH copy of the cluster row via
// s.repo.Get rather than mutating the *models.Cluster instance already
// returned to (and potentially already serialized by) the caller of
// ListClusters — this avoids ANY possibility of mutating response data
// after it may have already been written to the wire.
//
// Known, pre-existing, NOT newly introduced race: if RemoveCluster runs
// concurrently with this goroutine, s.repo.Get/Update may race the
// row's deletion (a harmless no-op UPDATE on a since-deleted id — SQLite
// does not error on a zero-row UPDATE), and GetOrReconnectClient could in
// principle populate s.clients[clusterID] for an id that RemoveCluster
// already deleted. This exact race already exists for every other
// concurrent caller of GetOrReconnectClient (e.g. an in-flight resource
// request during a removal) — this fix does not widen it beyond invoking
// an already-present, already-accepted code path from one additional call
// site; closing it fully is a broader concurrency change out of VALID-01's
// scope.
func (s *clusterService) kickBackgroundReconnect(clusterID string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[ListClusters] Panic in background reconnect for cluster %s: %v\n", clusterID, r)
			}
		}()

		bgCtx, cancel := context.WithTimeout(context.Background(), backgroundReconnectBudget)
		defer cancel()

		client, err := s.GetOrReconnectClient(bgCtx, clusterID)
		if err != nil {
			// GetOrReconnectClient has already recorded this failure in its own
			// negative-failure cache (bounding future reconnect attempts, not
			// just future DB writes). Correct the persisted row only if it
			// doesn't already reflect a non-connected state, so a cluster that
			// was last known "connected" (e.g. before a backend restart) does
			// not keep showing a stale, now-proven-false "connected" status
			// indefinitely.
			c, gerr := s.repo.Get(context.Background(), clusterID)
			if gerr != nil || c == nil {
				return // removed or lookup failed — nothing to correct
			}
			newStatus := clusterStatusFromError(err)
			if c.Status != newStatus {
				c.Status = newStatus
				_ = s.repo.Update(context.Background(), c)
			}
			return
		}

		c, gerr := s.repo.Get(context.Background(), clusterID)
		if gerr != nil || c == nil {
			return // cluster was removed while reconnecting; nothing to update
		}

		infoCtx, infoCancel := context.WithTimeout(context.Background(), reconnectTimeout)
		defer infoCancel()
		if info, ierr := client.GetClusterInfo(infoCtx); ierr == nil {
			c.ServerURL = clusterInfoString(info, "server_url")
			c.Version = clusterInfoString(info, "version")
			c.NodeCount = clusterInfoInt(info, "node_count")
			c.NamespaceCount = clusterInfoInt(info, "namespace_count")
		}
		c.Status = "connected"
		c.LastConnected = time.Now()
		if p, perr := client.DetectProvider(infoCtx); perr == nil && p != "" {
			c.Provider = p
		}
		_ = s.repo.Update(context.Background(), c)
		// Informer lifecycle: client re-established, informers start lazily
		// on first use via EnsureActive — see the registration-path comment
		// above for the same rationale.
	}()
}

func (s *clusterService) GetCluster(ctx context.Context, id string) (*models.Cluster, error) {
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}

	s.mu.RLock()
	client, ok := s.clients[id]
	s.mu.RUnlock()

	if ok {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if info, err := client.GetClusterInfo(ctx); err == nil {
			c.ServerURL = clusterInfoString(info, "server_url")
			c.Version = clusterInfoString(info, "version")
			c.NodeCount = clusterInfoInt(info, "node_count")
			c.NamespaceCount = clusterInfoInt(info, "namespace_count")
			c.Status = "connected"
			c.LastConnected = time.Now()
			if p, err := client.DetectProvider(ctx); err == nil && p != "" {
				c.Provider = p
			}
			_ = s.repo.Update(ctx, c)
		} else {
			c.Status = clusterStatusFromError(err)
			_ = s.repo.Update(ctx, c)
		}
	} else {
		if s.tryReconnectCluster(ctx, c) {
			s.mu.RLock()
			client, ok = s.clients[id]
			s.mu.RUnlock()
			if ok {
				info, _ := client.GetClusterInfo(ctx)
				if info != nil {
					c.ServerURL = clusterInfoString(info, "server_url")
					c.Version = clusterInfoString(info, "version")
					c.NodeCount = clusterInfoInt(info, "node_count")
					c.NamespaceCount = clusterInfoInt(info, "namespace_count")
				}
				c.Status = "connected"
				c.LastConnected = time.Now()
				if p, err := client.DetectProvider(ctx); err == nil && p != "" {
					c.Provider = p
				}
				_ = s.repo.Update(ctx, c)
			}
		} else {
			c.Status = "disconnected"
		}
	}
	return c, nil
}

// AddCluster registers a cluster loaded from the user's kubeconfig at the
// given path and context. Convenience wrapper around addClusterWithSource
// that forces source="kubeconfig" — the right value for every code path
// that reaches here (picker, API, auto-connect). For the upload flow,
// AddClusterFromBytes calls addClusterWithSource directly with source="upload".
func (s *clusterService) AddCluster(ctx context.Context, kubeconfigPath, contextName string) (*models.Cluster, error) {
	return s.addClusterWithSource(ctx, kubeconfigPath, contextName, "kubeconfig")
}

func (s *clusterService) addClusterWithSource(ctx context.Context, kubeconfigPath, contextName, source string) (*models.Cluster, error) {
	fmt.Printf("[AddCluster] Starting for context: %s, path: %s\n", contextName, kubeconfigPath)

	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			home, _ := os.UserHomeDir()
			if home != "" {
				kubeconfigPath = filepath.Join(home, ".kube", "config")
			}
		}
	}

	if kubeconfigPath == "" {
		return nil, fmt.Errorf("could not determine kubeconfig path")
	}

	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing clusters: %w", err)
	}

	if len(list) >= s.maxClusters {
		return nil, &ErrClusterLimitReached{Current: len(list), Max: s.maxClusters}
	}

	if _, err := os.Stat(kubeconfigPath); err != nil {
		return nil, fmt.Errorf("kubeconfig not found: %w", err)
	}

	fmt.Printf("[AddCluster] Initializing K8s client for %s\n", contextName)
	var client *k8s.Client
	if s.clientFactory != nil {
		client, err = s.clientFactory(kubeconfigPath, contextName)
	} else {
		client, err = k8s.NewClient(kubeconfigPath, contextName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to initialize k8s client: %w", err)
	}
	if s.k8sTimeout > 0 {
		client.SetTimeout(s.k8sTimeout)
	}
	if s.k8sRateLimitPerSec > 0 && s.k8sRateLimitBurst > 0 {
		client.SetLimiter(rate.NewLimiter(rate.Limit(s.k8sRateLimitPerSec), s.k8sRateLimitBurst))
	}

	// P0-B: For new registrations, cap connection test to 5s to avoid blocking the UI forever.
	fmt.Printf("[AddCluster] Testing connection for %s (5s timeout)\n", contextName)
	regCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	status := "connected"
	// Seed from the kubeconfig's own declared `server:` field — a local,
	// network-free read — so ServerURL is always populated, not only when
	// the live connection test below happens to succeed. Without this, an
	// unreachable cluster persists ServerURL="", which diverges from
	// KubeconfigFileSource's discovery identity (it reads the same field
	// directly off disk) for the exact same cluster. Two different
	// LogicalIdentity keys for one cluster means the presence Manager's
	// dedup never merges them, so the registered/session_id/kubeconfig_path
	// data never reaches the entry the frontend actually renders and lets
	// the user click — see TestClusterService_AddCluster_
	// UnreachableClusterKeepsDeclaredServerURL for the full chain.
	serverURL := serverURLFromKubeconfig(kubeconfigPath, contextName)
	version := ""
	provider := k8s.ProviderOnPrem

	if err := client.TestConnection(regCtx); err != nil {
		fmt.Printf("[AddCluster] Connection test failed for %s: %v\n", contextName, err)
		status = clusterStatusFromError(err)
	} else {
		fmt.Printf("[AddCluster] Connection test successful for %s\n", contextName)
		if info, err := client.GetClusterInfo(regCtx); err == nil {
			// Prefer the live value (it's the source of truth when available),
			// but never regress to empty — keep the kubeconfig-declared seed
			// above if GetClusterInfo didn't actually return one.
			if live := clusterInfoString(info, "server_url"); live != "" {
				serverURL = live
			}
			version = clusterInfoString(info, "version")
			if p, err := client.DetectProvider(regCtx); err == nil && p != "" {
				provider = p
			}
		} else {
			status = "error"
		}
	}

	// P2-10: Idempotent add — return existing cluster (same ID) when (context, kubeconfig_path) or (context, server_url) matches.
	normPath := filepath.Clean(kubeconfigPath)
	for _, c := range list {
		if c.Context != contextName {
			continue
		}
		// Match by path or server URL (if we were able to get it)
		pathMatch := filepath.Clean(c.KubeconfigPath) == normPath
		urlMatch := serverURL != "" && c.ServerURL == serverURL

		if pathMatch || urlMatch {
			c.Status = status
			c.LastConnected = time.Now()
			if serverURL != "" {
				c.ServerURL = serverURL
			}
			if version != "" {
				c.Version = version
			}
			if provider != k8s.ProviderOnPrem {
				c.Provider = provider
			}
			c.UpdatedAt = time.Now()
			fmt.Printf("[AddCluster] Idempotent match found, updating cluster %s\n", c.ID)
			if err := s.repo.Update(ctx, c); err != nil {
				return nil, fmt.Errorf("failed to update existing cluster: %w", err)
			}
			if status == "connected" {
				s.mu.Lock()
				s.clients[c.ID] = client
				s.mu.Unlock()
				// Informer lifecycle: client stored; informers start lazily
				// on first use (EnsureActive), not here.
			}
			return c, nil
		}
	}

	cluster := &models.Cluster{
		ID:             uuid.New().String(),
		Name:           contextName, // Default name to context
		Context:        contextName,
		KubeconfigPath: normPath,
		ServerURL:      serverURL,
		Version:        version,
		Status:         status,
		Provider:       provider,
		Source:         source,
		LastConnected:  time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	fmt.Printf("[AddCluster] Persisting new cluster %s in repo\n", cluster.ID)
	if err := s.repo.Create(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to persist cluster: %w", err)
	}

	if status == "connected" {
		s.mu.Lock()
		s.clients[cluster.ID] = client
		s.mu.Unlock()
		// Informer lifecycle: client stored; informers start lazily on
		// first use (EnsureActive), not here.
	}

	fmt.Printf("[AddCluster] Successfully registered %s\n", cluster.ID)
	return cluster, nil
}

// AddClusterFromBytes adds a cluster from raw kubeconfig bytes (browser upload / paste).
// It resolves the context name, writes the kubeconfig to ~/.kubilitics/kubeconfigs/<context>.yaml
// with 0600 permissions, then delegates fully to AddCluster for persistence and provider detection.
func (s *clusterService) AddClusterFromBytes(ctx context.Context, kubeconfigBytes []byte, contextName string) (*models.Cluster, error) {
	// Parse kubeconfig to resolve context name and validate structure.
	rawConfig, err := clientcmd.Load(kubeconfigBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid kubeconfig: %w", err)
	}

	if contextName == "" {
		contextName = rawConfig.CurrentContext
	}
	if contextName == "" {
		// Pick first available context when current-context is not set.
		for name := range rawConfig.Contexts {
			contextName = name
			break
		}
	}
	if contextName == "" {
		return nil, fmt.Errorf("kubeconfig contains no contexts")
	}
	if _, exists := rawConfig.Contexts[contextName]; !exists {
		available := make([]string, 0, len(rawConfig.Contexts))
		for n := range rawConfig.Contexts {
			available = append(available, n)
		}
		return nil, fmt.Errorf("context %q not found in kubeconfig (available: %s)", contextName, strings.Join(available, ", "))
	}

	// Persist to ~/.kubilitics/kubeconfigs/<sanitized-context>.yaml
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}
	kubeDir := filepath.Join(home, ".kubilitics", "kubeconfigs")
	if err := os.MkdirAll(kubeDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create kubeconfigs directory: %w", err)
	}

	safeName := sanitizeContextForFilename(contextName)
	kubeconfigPath := filepath.Join(kubeDir, safeName+".yaml")

	if err := os.WriteFile(kubeconfigPath, kubeconfigBytes, 0600); err != nil {
		return nil, fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	fmt.Printf("[AddClusterFromBytes] Written kubeconfig to %s for context %s\n", kubeconfigPath, contextName)
	return s.addClusterWithSource(ctx, kubeconfigPath, contextName, "upload")
}

// sanitizeContextForFilename maps a Kubernetes context name to a safe filesystem name.
// Characters outside [a-zA-Z0-9._-] are replaced with '-'. Max length 200.
func sanitizeContextForFilename(name string) string {
	safe := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '-'
	}, name)
	if len(safe) > 200 {
		safe = safe[:200]
	}
	if safe == "" {
		safe = "default"
	}
	return safe
}

func (s *clusterService) RemoveCluster(ctx context.Context, id string) error {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return fmt.Errorf("cluster not found: %s", id)
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.clients, id)
	s.mu.Unlock()
	s.lifecycle.Remove(id)
	return nil
}

func (s *clusterService) TestConnection(ctx context.Context, id string) error {
	s.mu.RLock()
	client, exists := s.clients[id]
	s.mu.RUnlock()
	if !exists {
		return fmt.Errorf("cluster not found: %s", id)
	}
	return client.TestConnection(ctx)
}

func (s *clusterService) GetClusterSummary(ctx context.Context, id string) (*models.ClusterSummary, error) {
	s.mu.RLock()
	client, exists := s.clients[id]
	s.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("cluster not found: %s", id)
	}

	info, err := client.GetClusterInfo(ctx)
	if err != nil {
		return nil, err
	}

	// FLEET-1 UNVERIFIED caveat, resolved (docs/PRODUCTION-RELIABILITY-AUDIT.md,
	// Phase 4): client.Timeout IS always set in production (NewClusterService is
	// always constructed with a real, viper-loaded cfg — K8sTimeoutSec defaults
	// to 30s — confirmed by reading cmd/server/main.go and config.go), but
	// these four raw Clientset calls never applied it: they ran on ctx alone,
	// like LOADING-3's Overview-handler bug. GetFleetOverview calls this method
	// per cluster concurrently (fleet.go); a single registered-but-newly-slow
	// cluster (e.g. VPN flapping) had no effective deadline here and could
	// still slow the whole Fleet aggregate response. Bound with the same
	// client.WithTimeout() wrapper LOADING-3 already established.
	listCtx, listCancel := client.WithTimeout(ctx)
	defer listCancel()

	// Perf: informer cache first (Headlamp/Lens model) for all 4 counts —
	// this method is GetFleetOverview's per-cluster fan-out (fleet.go), so at
	// N clusters these were N×4 live API calls with no caching; a warm cache
	// now answers in <1ms instead. Falls back to the original live Clientset
	// calls on a cache miss (informer not synced yet), unchanged from before.
	im := s.GetInformerManager(id)
	nodes := &corev1.NodeList{}
	if items, ok := cachedTypedItems[corev1.Node](im, "nodes"); ok {
		nodes.Items = items
	} else {
		nodes, _ = client.Clientset.CoreV1().Nodes().List(listCtx, metav1.ListOptions{})
	}
	pods := &corev1.PodList{}
	if items, ok := cachedTypedItems[corev1.Pod](im, "pods"); ok {
		pods.Items = items
	} else {
		pods, _ = client.Clientset.CoreV1().Pods("").List(listCtx, metav1.ListOptions{})
	}
	deployments := &appsv1.DeploymentList{}
	if items, ok := cachedTypedItems[appsv1.Deployment](im, "deployments"); ok {
		deployments.Items = items
	} else {
		deployments, _ = client.Clientset.AppsV1().Deployments("").List(listCtx, metav1.ListOptions{})
	}
	services := &corev1.ServiceList{}
	if items, ok := cachedTypedItems[corev1.Service](im, "services"); ok {
		services.Items = items
	} else {
		services, _ = client.Clientset.CoreV1().Services("").List(listCtx, metav1.ListOptions{})
	}

	// Compute health from actual resource state
	healthStatus := computeClusterHealthStatus(nodes, pods, deployments)

	// FLEET-N1 (docs/FLEET-N1-IMPLEMENTATION.md): Reachable was never set here
	// — it silently stayed at Go's zero-value (false) on every call, success
	// or not. That went unnoticed while this method's only caller
	// (GetFleetOverview) was discarding Reachable anyway; caught live when
	// GetFleetOverview was fixed to actually surface it to the frontend —
	// every cluster would have shown as "unreachable" in Fleet regardless of
	// true state, a regression versus the per-cluster /summary endpoint
	// (buildClusterSummary, a separate implementation) which does set it
	// correctly. Reaching this line means every K8s call above succeeded
	// enough to compute real counts, so Reachable=true is correct here.
	return &models.ClusterSummary{
		ID:              id,
		Name:            id,
		NodeCount:       clusterInfoInt(info, "node_count"),
		NamespaceCount:  clusterInfoInt(info, "namespace_count"),
		PodCount:        len(pods.Items),
		DeploymentCount: len(deployments.Items),
		ServiceCount:    len(services.Items),
		HealthStatus:    healthStatus,
		Reachable:       true,
	}, nil
}

// GetClient returns K8s client for internal use
func (s *clusterService) GetClient(id string) (*k8s.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client, ok := s.clients[id]
	if !ok {
		return nil, fmt.Errorf("client not found for cluster %s", id)
	}
	return client, nil
}

// HasMetalLB returns true if MetalLB CRDs (ipaddresspools.metallb.io, bgppeers.metallb.io) are installed.
// Tries to list ipaddresspools with limit=1; 404 means MetalLB is not installed.
func (s *clusterService) HasMetalLB(ctx context.Context, id string) (bool, error) {
	client, err := s.GetClient(id)
	if err != nil {
		return false, err
	}
	opts := metav1.ListOptions{Limit: 1}
	_, err = client.ListResources(ctx, "ipaddresspools", "", opts)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// buildClientForCluster returns a fresh K8s client for the given cluster row.
// Three modes:
//   - Source == "in-cluster": use rest.InClusterConfig (k8s.NewClient with empty path).
//     This handles the seamless `helm install kubilitics` flow where the hub auto-
//     registers itself with no kubeconfig.
//   - KubeconfigPath != "": load that kubeconfig file with the stored Context.
//   - KubeconfigPath == "" (legacy): caller decides fallback (e.g. ~/.kube/config).
//
// Single source of truth so all three connect/reconnect/load paths stay in lockstep.
func (s *clusterService) buildClientForCluster(c *models.Cluster) (*k8s.Client, error) {
	if c.Source == "in-cluster" {
		if s.clientFactory != nil {
			return s.clientFactory("", "")
		}
		return k8s.NewClient("", "")
	}
	if c.KubeconfigPath == "" {
		return nil, fmt.Errorf("cluster %s has no stored kubeconfig path — reconnect or remove it", c.ID)
	}
	if s.clientFactory != nil {
		return s.clientFactory(c.KubeconfigPath, c.Context)
	}
	return k8s.NewClient(c.KubeconfigPath, c.Context)
}

// loadStartupTimeout is the per-cluster timeout for connection tests during startup.
// Keep it short so the backend starts promptly even when clusters are offline or require
// slow exec-based auth (aws eks get-token, gke-gcloud-auth-plugin, etc.).
const loadStartupTimeout = 8 * time.Second

// loadClustersConcurrency bounds how many clusters LoadClustersFromRepo connects
// to at once. STARTUP-1 (docs/PRODUCTION-RELIABILITY-AUDIT.md): the previous
// sequential loop cost N x loadStartupTimeout in the worst case (measured in
// docs/PRODUCTION-BASELINE.md: 3 unreachable clusters = 24s). Fanning out fully
// unbounded isn't safe either — each connection attempt can spawn an exec-based
// auth plugin subprocess (aws eks get-token, gke-gcloud-auth-plugin), and at the
// "100+ cluster enterprise architect" scale this product targets, launching all
// of them simultaneously would spike CPU/process count and cloud-provider auth
// API load for no benefit once far more than CPU-core-count are in flight. 10
// mirrors common bounded-worker-pool defaults and keeps the worst case at
// roughly (N/10) x loadStartupTimeout instead of N x loadStartupTimeout.
const loadClustersConcurrency = 10

// LoadClustersFromRepo restores K8s clients from persisted clusters (call on startup).
// Per-cluster failures do not abort the process; each cluster gets status disconnected/error.
// Connection tests run with a hard per-cluster timeout so unreachable or exec-auth clusters
// (EKS, GKE, AKS) never block the server from starting. Clusters are connected concurrently
// (bounded by loadClustersConcurrency), reusing the same bounded-fan-out pattern already
// proven correct in GetFleetOverview (internal/api/rest/fleet.go) — one slow/unreachable
// cluster no longer multiplies startup latency for every other cluster.
func (s *clusterService) LoadClustersFromRepo(ctx context.Context) error {
	clusters, err := s.repo.List(ctx)
	if err != nil {
		return err
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(loadClustersConcurrency)

	for _, c := range clusters {
		c := c // capture loop variable
		g.Go(func() error {
			s.loadOneClusterFromRepo(gCtx, c)
			return nil // per-cluster failures never abort the group — see loadOneClusterFromRepo
		})
	}

	// g.Wait() only returns an error if a Go func returned one; ours never do,
	// so this is defensive, matching GetFleetOverview's own comment.
	return g.Wait()
}

// loadOneClusterFromRepo performs the full connect sequence for a single
// persisted cluster row: build client, apply timeout/rate-limit, test
// connection, and — only if reachable — register the client and start its
// informer cache. Errors are recorded on the cluster's persisted status, never
// returned, so one cluster's failure can never affect another's (LoadClustersFromRepo
// runs this concurrently, bounded by loadClustersConcurrency).
func (s *clusterService) loadOneClusterFromRepo(ctx context.Context, c *models.Cluster) {
	// in-cluster rows have empty KubeconfigPath but build a client via
	// rest.InClusterConfig — handled inside buildClientForCluster.
	if c.KubeconfigPath == "" && c.Source != "in-cluster" {
		c.Status = "disconnected"
		_ = s.repo.Update(ctx, c)
		return
	}
	client, clientErr := s.buildClientForCluster(c)

	if clientErr != nil {
		fmt.Printf("[LoadClustersFromRepo] Skipping cluster %s (%s): failed to create client: %v\n", c.ID, c.Context, clientErr)
		c.Status = "error"
		_ = s.repo.Update(ctx, c)
		return
	}

	if s.k8sTimeout > 0 {
		client.SetTimeout(s.k8sTimeout)
	}
	if s.k8sRateLimitPerSec > 0 && s.k8sRateLimitBurst > 0 {
		client.SetLimiter(rate.NewLimiter(rate.Limit(s.k8sRateLimitPerSec), s.k8sRateLimitBurst))
	}

	// Test connection with a hard per-cluster deadline so exec-based auth plugins
	// (aws eks get-token, gke-gcloud-auth-plugin) and offline clusters don't block startup.
	testCtx, testCancel := context.WithTimeout(ctx, loadStartupTimeout)
	connErr := client.TestConnection(testCtx)
	testCancel()
	if connErr != nil {
		fmt.Printf("[LoadClustersFromRepo] Cluster %s (%s): connection test failed (%v) — marking %s\n",
			c.ID, c.Context, connErr, clusterStatusFromError(connErr))
		c.Status = clusterStatusFromError(connErr)
	} else {
		c.Status = "connected"
	}

	if connErr == nil {
		// Only register the live client when the cluster is reachable.
		// Informer lifecycle (docs/INFORMER-LIFECYCLE-IMPLEMENTATION.md):
		// this is the single highest-impact fix identified by the
		// investigation — backend startup previously started the full
		// 27-informer set for EVERY persisted, reachable cluster from any
		// past session, before any UI connected. Now it only establishes
		// the client; EnsureActive starts informers lazily on first real
		// use. A user with 50 previously-registered clusters no longer
		// pays ~297 goroutines × 50 on every backend restart regardless of
		// which (if any) they intend to use this session.
		s.mu.Lock()
		s.clients[c.ID] = client
		s.mu.Unlock()

		c.LastConnected = time.Now()

		// Detect provider with the same short timeout so it never blocks startup.
		provCtx, provCancel := context.WithTimeout(ctx, loadStartupTimeout)
		if p, err := client.DetectProvider(provCtx); err == nil && p != "" {
			c.Provider = p
		}
		provCancel()
	} else {
		fmt.Printf("[LoadClustersFromRepo] Cluster %s (%s): skipping informer cache (cluster is %s)\n",
			c.ID, c.Context, c.Status)
	}

	_ = s.repo.Update(ctx, c)
}

// tryReconnectCluster builds a K8s client for a cluster when none is in memory (e.g. after restart).
// Uses stored KubeconfigPath if the file exists; otherwise falls back to default kubeconfig (~/.kube/config)
// so clusters like docker-desktop work when kubectl works on the same machine.
// Returns true if a client was created and stored.
func (s *clusterService) tryReconnectCluster(ctx context.Context, c *models.Cluster) bool {
	// In-cluster rows take the dedicated InClusterConfig path; no kubeconfig file ever.
	if c.Source == "in-cluster" {
		client, err := s.buildClientForCluster(c)
		if err != nil { return false }
		return s.applyAndStoreClient(ctx, c, client)
	}
	path := c.KubeconfigPath
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			path = "" // stored path missing (e.g. temp upload file gone); try default
		}
	}
	if path == "" {
		home, _ := os.UserHomeDir()
		if home != "" {
			path = filepath.Join(home, ".kube", "config")
		}
		if path == "" {
			return false
		}
	}
	client, err := k8s.NewClient(path, c.Context)
	if err != nil {
		return false
	}
	return s.applyAndStoreClient(ctx, c, client)
}

// applyAndStoreClient applies the configured timeout + rate limiter to a freshly
// built client, runs a connection test, and stores it in the in-memory client map.
// Returns false if the connection test fails — the cluster row is left unchanged.
func (s *clusterService) applyAndStoreClient(ctx context.Context, c *models.Cluster, client *k8s.Client) bool {
	if s.k8sTimeout > 0 {
		client.SetTimeout(s.k8sTimeout)
	}
	if s.k8sRateLimitPerSec > 0 && s.k8sRateLimitBurst > 0 {
		client.SetLimiter(rate.NewLimiter(rate.Limit(s.k8sRateLimitPerSec), s.k8sRateLimitBurst))
	}
	if err := client.TestConnection(ctx); err != nil {
		return false
	}
	// Map write must hold the lock — GetClient reads under RLock concurrently.
	s.mu.Lock()
	s.clients[c.ID] = client
	s.mu.Unlock()
	// VALID-04 (docs/VALID-04-INVESTIGATION.md), now mediated through
	// ClusterLifecycleManager.Reconnected (docs/INFORMER-LIFECYCLE-
	// IMPLEMENTATION.md): stop-before-start still applies (old client's
	// informers must never keep running against a new client), but the
	// manager's own entry.mu now owns this transition so its bookkeeping
	// (state/generation) never drifts out of sync with what OverviewCache
	// actually has running — only reached after TestConnection has already
	// succeeded, so a failed reconnect attempt never tears down a
	// still-working cache.
	if err := s.lifecycle.Reconnected(ctx, c.ID, client); err != nil {
		return false
	}
	return true
}

// finishReconnect persists the post-reconnect cluster row, but only if the row still exists.
// ReconnectCluster's success paths run a slow network call (TestConnection/GetClusterInfo) between
// reading c and writing it back; if RemoveCluster deletes the row during that window, an unconditional
// Update would resurrect it (see docs/VALID-01-INVESTIGATION.md). If the row is gone, this also rolls
// back the client/cache entries ReconnectCluster already installed for it, so a removed cluster is
// never left with a live client.
func (s *clusterService) finishReconnect(ctx context.Context, c *models.Cluster) {
	if _, err := s.repo.Get(ctx, c.ID); err != nil {
		// Route through lifecycle.Remove (not overviewCache.StopClusterCache
		// directly) so the lifecycle entry's own bookkeeping is marked
		// removed/deleted too — otherwise Reconnected() having just set
		// state=Active moments ago would leave the manager believing this
		// cluster is still active after this rollback actually stopped it,
		// a real state-vs-reality drift this fix closes.
		s.lifecycle.Remove(c.ID)
		s.mu.Lock()
		delete(s.clients, c.ID)
		s.mu.Unlock()
		return
	}
	_ = s.repo.Update(ctx, c)
}

// ReconnectCluster resets the circuit breaker for an existing client (if any) and builds a fresh
// K8s client from the stored kubeconfig. Updates the cluster status in the DB.
func (s *clusterService) ReconnectCluster(ctx context.Context, id string) (*models.Cluster, error) {
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %s", id)
	}

	// Reset the circuit breaker on any existing client so it doesn't block the TestConnection below.
	s.mu.RLock()
	existing, existed := s.clients[id]
	s.mu.RUnlock()
	if existed {
		existing.ResetCircuitBreaker()
	}

	// In-cluster rows go through buildClientForCluster (rest.InClusterConfig).
	// They have no kubeconfig file by design and must skip path validation entirely.
	if c.Source == "in-cluster" {
		client, err := s.buildClientForCluster(c)
		if err != nil {
			c.Status = "error"
			_ = s.repo.Update(ctx, c)
			return c, fmt.Errorf("in-cluster reconnect failed: %w", err)
		}
		if !s.applyAndStoreClient(ctx, c, client) {
			c.Status = "error"
			_ = s.repo.Update(ctx, c)
			return c, fmt.Errorf("in-cluster connection test failed for %s", id)
		}
		c.Status = "connected"
		c.LastConnected = time.Now()
		s.finishReconnect(ctx, c)
		return c, nil
	}

	// Build a fresh client (re-reads kubeconfig, fresh TLS handshake, new circuit breaker).
	//
	// Strict path validation: a missing or unreadable stored kubeconfig means
	// the cluster is gone. We must NEVER fall back to the user's system
	// ~/.kube/config — that would silently build a client against whatever
	// context is currently active (e.g. docker-desktop) while the DB row and
	// the frontend still identify this cluster as the original (e.g. AWS).
	// Silent identity substitution is a P0 data-integrity bug.
	path := c.KubeconfigPath
	if path == "" {
		c.Status = "disconnected"
		_ = s.repo.Update(ctx, c)
		return c, fmt.Errorf("cluster %s has no stored kubeconfig path — reconnect or remove it", id)
	}
	info, statErr := os.Stat(path)
	if statErr != nil {
		if errors.Is(statErr, fs.ErrNotExist) {
			c.Status = "disconnected"
			_ = s.repo.Update(ctx, c)
			return c, fmt.Errorf("cluster %s kubeconfig file no longer exists at %s — reconnect or remove it", id, path)
		}
		// Transient I/O error (permission, network-mount unreachable). Report but do
		// not mark disconnected — the next retry may succeed.
		return c, fmt.Errorf("cluster %s kubeconfig file unreadable at %s: %w", id, path, statErr)
	}
	if info.IsDir() {
		c.Status = "disconnected"
		_ = s.repo.Update(ctx, c)
		return c, fmt.Errorf("cluster %s kubeconfig path is a directory, not a file: %s", id, path)
	}

	client, err := k8s.NewClient(path, c.Context)
	if err != nil {
		c.Status = "error"
		_ = s.repo.Update(ctx, c)
		return c, fmt.Errorf("failed to create client: %w", err)
	}
	if s.k8sTimeout > 0 {
		client.SetTimeout(s.k8sTimeout)
	}
	if s.k8sRateLimitPerSec > 0 && s.k8sRateLimitBurst > 0 {
		client.SetLimiter(rate.NewLimiter(rate.Limit(s.k8sRateLimitPerSec), s.k8sRateLimitBurst))
	}
	client.SetClusterID(id)

	if err := client.TestConnection(ctx); err != nil {
		c.Status = clusterStatusFromError(err)
		_ = s.repo.Update(ctx, c)
		return c, fmt.Errorf("connection test failed: %w", err)
	}

	// Success: replace client, then replace the informer generation through
	// the lifecycle manager (keeps its state/generation bookkeeping
	// authoritative — see Reconnected's doc comment).
	// Map write must hold the lock — GetClient reads it under RLock from other
	// goroutines; unlocked writes race the runtime map rehash and can segfault.
	s.mu.Lock()
	s.clients[id] = client
	s.mu.Unlock()
	_ = s.lifecycle.Reconnected(ctx, id, client)
	// Clear any negative-cache entry now that we have a live client again.
	s.reconnectFailCache.Delete(id)

	if info, err := client.GetClusterInfo(ctx); err == nil {
		c.ServerURL = clusterInfoString(info, "server_url")
		c.Version = clusterInfoString(info, "version")
		c.NodeCount = clusterInfoInt(info, "node_count")
		c.NamespaceCount = clusterInfoInt(info, "namespace_count")
	}
	if p, err := client.DetectProvider(ctx); err == nil && p != "" {
		c.Provider = p
	}
	c.Status = "connected"
	c.LastConnected = time.Now()
	s.finishReconnect(ctx, c)
	return c, nil
}

// GetOverview is one of the two choke points (with GetInformerManager) every
// informer-backed-data consumer goes through — see EnsureActive's doc
// comment. Triggers lazy activation as a side effect (ignoring the returned
// *InformerManager; callers here want the cached overview, not the raw
// manager) so a cluster that has never been viewed warms up on its first
// Dashboard request. The first call after a cold activation still returns
// (ov, false) — informer events haven't populated the overview yet — and
// GetClusterOverview's existing direct-API fallback (internal/api/rest/
// overview.go) serves that one request; subsequent calls return real
// cached data once the informers have synced.
func (s *clusterService) GetOverview(clusterID string) (*models.ClusterOverview, bool) {
	s.ensureActiveBestEffort(clusterID)
	return s.overviewCache.GetOverview(clusterID)
}

// Subscribe is the WebSocket real-time-overview entry point. Also triggers
// activation — a subscriber opening a live feed for a never-viewed cluster
// must not sit on a channel that never receives anything because nothing
// ever started generating events for it.
func (s *clusterService) Subscribe(clusterID string) (chan *models.ClusterOverview, func(), error) {
	s.ensureActiveBestEffort(clusterID)
	return s.overviewCache.Subscribe(clusterID)
}

// ensureActiveBestEffort triggers lazy informer activation for clusterID if
// a live client exists, swallowing any error — these call sites (GetOverview,
// Subscribe) already have their own established "cache miss -> fall back"
// behavior (GetClusterOverview's direct-API path; an empty WS feed that
// fills in once synced) and must not fail the caller's request just because
// activation itself failed (e.g. a transient issue) — EnsureActive's own
// error return exists for callers that DO need to know synchronously
// (GetInformerManager's callers already tolerate a nil manager the same way).
func (s *clusterService) ensureActiveBestEffort(clusterID string) {
	s.mu.RLock()
	client, ok := s.clients[clusterID]
	s.mu.RUnlock()
	if !ok {
		return
	}
	_, _ = s.lifecycle.EnsureActive(context.Background(), clusterID, client)
}

// GetInformerManager is the other of the two choke points (with GetOverview)
// every informer-backed-data consumer goes through (resources.go,
// workloads.go, events.go — all already written to tolerate a nil return by
// falling back to a direct API call, confirmed by reading those call sites
// before this change). Triggers lazy activation; on a cold cluster this
// call itself returns the manager EnsureActive just started (informers
// running, not yet synced) rather than nil, so HasSynced() gating downstream
// behaves exactly as it already does today in the brief window right after
// an eager start — no new caller-visible state was introduced.
func (s *clusterService) GetInformerManager(clusterID string) *k8s.InformerManager {
	s.mu.RLock()
	client, ok := s.clients[clusterID]
	s.mu.RUnlock()
	if !ok {
		return s.overviewCache.GetInformerManager(clusterID) // no client — nothing to activate; preserves today's nil-on-unregistered behavior
	}
	im, err := s.lifecycle.EnsureActive(context.Background(), clusterID, client)
	if err != nil {
		return nil
	}
	return im
}

// DiscoverClusters scans the configured kubeconfig (or default ~/.kube/config) for contexts not yet in the repository.
func (s *clusterService) DiscoverClusters(ctx context.Context) ([]*models.Cluster, error) {
	kubeconfigPath := ""
	// Try to get path from environment or default
	kubeconfigPath = os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		home, _ := os.UserHomeDir()
		if home != "" {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	if kubeconfigPath == "" {
		return nil, fmt.Errorf("could not determine kubeconfig path")
	}

	if _, err := os.Stat(kubeconfigPath); err != nil {
		return nil, fmt.Errorf("kubeconfig not found at %s", kubeconfigPath)
	}

	contexts, currentContext, err := k8s.GetKubeconfigContexts(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list kubeconfig contexts: %w", err)
	}

	existingClusters, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing clusters: %w", err)
	}

	existingContexts := make(map[string]bool)
	for _, c := range existingClusters {
		existingContexts[c.Context] = true
	}

	var discovered []*models.Cluster
	for _, contextName := range contexts {
		if !existingContexts[contextName] {
			// BA-1: Ephemeral UUID so frontend has a stable handle before registration (Connect works; no clusters// in URLs).
			discovered = append(discovered, &models.Cluster{
				ID:             uuid.New().String(),
				Name:           contextName,
				Context:        contextName,
				KubeconfigPath: kubeconfigPath,
				Status:         "detected",
				IsCurrent:      contextName == currentContext,
			})
		}
	}

	return discovered, nil
}

// computeClusterHealthStatus derives an overall cluster health status from live
// node, pod, and deployment state. Returns "healthy", "degraded", or "unhealthy".
func computeClusterHealthStatus(nodes *corev1.NodeList, pods *corev1.PodList, deployments *appsv1.DeploymentList) string {
	health := "healthy"

	// Any node not Ready → degraded
	if nodes != nil {
		for i := range nodes.Items {
			ready := false
			for _, cond := range nodes.Items[i].Status.Conditions {
				if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
					ready = true
					break
				}
			}
			if !ready {
				health = "degraded"
				break
			}
		}
	}

	// Pod failure ratio >30% → unhealthy, >10% → degraded
	if pods != nil && len(pods.Items) > 0 {
		total := len(pods.Items)
		failing := 0
		for i := range pods.Items {
			phase := pods.Items[i].Status.Phase
			if phase == corev1.PodFailed || phase == corev1.PodPending {
				failing++
			}
		}
		ratio := float64(failing) / float64(total)
		if ratio > 0.3 {
			return "unhealthy"
		}
		if ratio > 0.1 {
			health = "degraded"
		}
	}

	// Any deployment with unavailable replicas → degraded
	if deployments != nil && health == "healthy" {
		for i := range deployments.Items {
			if deployments.Items[i].Status.UnavailableReplicas > 0 {
				health = "degraded"
				break
			}
		}
	}

	return health
}

// clusterStatusFromError maps K8s/context errors to status: "disconnected" for connection/context errors, "error" for 403/5xx etc.
func clusterStatusFromError(err error) string {
	if err == nil {
		return "connected"
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "disconnected"
	}
	return "error"
}
