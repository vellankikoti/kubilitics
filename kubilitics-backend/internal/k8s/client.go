package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/kubilitics/kubilitics-backend/internal/pkg/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetKubeconfigContexts returns all context names and the current context from a kubeconfig file.
func GetKubeconfigContexts(kubeconfigPath string) ([]string, string, error) {
	if kubeconfigPath == "" {
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
		}
	}
	if kubeconfigPath == "" {
		return nil, "", nil
	}
	raw, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{},
	).RawConfig()
	if err != nil {
		return nil, "", err
	}
	names := make([]string, 0, len(raw.Contexts))
	for name := range raw.Contexts {
		names = append(names, name)
	}
	return names, raw.CurrentContext, nil
}

// Client wraps Kubernetes client-go
type Client struct {
	Clientset      kubernetes.Interface
	Dynamic        dynamic.Interface
	Config         *rest.Config
	Context        string
	kubeconfigPath string
	// Timeout for outbound K8s API calls; 0 means no timeout (use request context only).
	Timeout time.Duration
	// Limiter optionally rate-limits outbound API calls per cluster (C1.5). Nil = no limit.
	limiter *rate.Limiter
	// CircuitBreaker protects against cascading failures (BE-SCALE-001).
	circuitBreaker *CircuitBreaker
	// Health status: last successful call time, last error, etc.
	lastSuccessTime time.Time
	lastError       error
	// lastCheckedTime records when TestConnection/GetClusterInfo was last
	// attempted, success or failure — distinct from lastSuccessTime, which
	// only moves forward on success. HEALTH-2 (docs/PRODUCTION-RELIABILITY-
	// AUDIT.md): without this, nothing could distinguish "confirmed reachable
	// 2 seconds ago" from "confirmed reachable 10 minutes ago" from "never
	// actually checked."
	lastCheckedTime time.Time
	healthMu        sync.RWMutex
	// BA-8: Discovery cache for CRD/unknown resource types; TTL 5 min.
	discoveryCache     []DiscoveredResource
	discoveryCacheTime time.Time
	discoveryCacheMu   sync.Mutex
}

// defaultClientQPS/defaultClientBurst override client-go's own built-in
// default (rest.DefaultQPS=5, rest.DefaultBurst=10) on every constructed
// K8s client. Set once at startup via SetDefaultClientRateLimit (cmd/server/
// main.go, right after config loads, before any client is constructed) —
// zero value (unset) means "use client-go's default," so a backend that
// never calls SetDefaultClientRateLimit (e.g. in unit tests) is unaffected.
//
// Phase 2D (docs/TOPOLOGY-CONCURRENCY-INVESTIGATION.md): this limiter lives
// on the shared rest.Config/clientset for a cluster, applied to EVERY
// concurrent caller of that cluster's client — client-go's conservative
// default (tuned for low-frequency controller reconcile loops) throttles
// severely under Kubilitics' own concurrent-request load (10 concurrent
// topology builds: 18.2s -> 3.2s after raising this; verified the real API
// server's own CPU stayed under 20% throughout both, i.e. this was not
// simply shifting an overload onto the API server).
var defaultClientQPS float32
var defaultClientBurst int

// SetDefaultClientRateLimit configures the QPS/Burst every subsequently
// constructed K8s client uses, in place of client-go's own default.
// qps<=0 or burst<=0 leaves client-go's default (5/10) in effect for that
// value. Call once at startup, before any client is constructed.
func SetDefaultClientRateLimit(qps float32, burst int) {
	defaultClientQPS = qps
	defaultClientBurst = burst
}

func applyDefaultClientRateLimit(config *rest.Config) {
	if defaultClientQPS > 0 {
		config.QPS = defaultClientQPS
	}
	if defaultClientBurst > 0 {
		config.Burst = defaultClientBurst
	}
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfigPath, context string) (*Client, error) {
	var config *rest.Config
	var err error

	if kubeconfigPath == "" {
		// Try in-cluster config first
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to default kubeconfig
			homeDir, _ := os.UserHomeDir()
			if homeDir != "" {
				kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
			}
		}
	}

	if config == nil {
		config, err = buildConfigFromFlags(context, kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build config: %w", err)
		}
	}
	applyDefaultClientRateLimit(config)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Client{
		Clientset:      clientset,
		Dynamic:        dynamicClient,
		Config:         config,
		Context:        context,
		kubeconfigPath: kubeconfigPath,
		circuitBreaker: NewCircuitBreaker(""), // clusterID will be set via SetClusterID if available
		lastSuccessTime: time.Now(),
	}, nil
}

// SetTimeout sets the timeout for outbound K8s API calls. Call after NewClient when config is available.
func (c *Client) SetTimeout(d time.Duration) {
	c.Timeout = d
}

// KubeconfigPath returns the kubeconfig file path used by this client.
func (c *Client) KubeconfigPath() string {
	return c.kubeconfigPath
}

// ClusterID returns the cluster ID this client's health-check metrics are
// currently labeled with (empty if SetClusterID was never called).
func (c *Client) ClusterID() string {
	if c.circuitBreaker != nil {
		return c.circuitBreaker.clusterID
	}
	return ""
}

// SetClusterID sets the cluster ID for circuit breaker metrics labeling.
func (c *Client) SetClusterID(clusterID string) {
	if c.circuitBreaker != nil {
		c.circuitBreaker.clusterID = clusterID
	}
}

// ResetCircuitBreaker forcibly closes the circuit breaker so the next call is attempted.
// Call this before building a fresh client (reconnect) so any open state from a prior client is cleared.
func (c *Client) ResetCircuitBreaker() {
	if c.circuitBreaker != nil {
		c.circuitBreaker.Reset()
	}
}

// SetLimiter sets a token-bucket rate limiter for outbound K8s API calls (C1.5). Call after NewClient when config is available.
func (c *Client) SetLimiter(l *rate.Limiter) {
	c.limiter = l
}

func (c *Client) waitRateLimit(ctx context.Context) error {
	if c.limiter == nil {
		return nil
	}
	return c.limiter.Wait(ctx)
}

// withTimeout returns ctx with timeout applied if c.Timeout > 0; otherwise returns ctx and a no-op cancel.
func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.Timeout > 0 {
		return context.WithTimeout(ctx, c.Timeout)
	}
	return ctx, func() {}
}

// WithTimeout is the exported form of withTimeout, for callers outside this
// package that make raw Clientset calls directly (bypassing the wrapped
// TestConnection/GetClusterInfo/ListResources helpers, which already apply
// this internally). LOADING-3/LOADING-4 (docs/PRODUCTION-RELIABILITY-AUDIT.md):
// every K8s API call must be bounded by the client's configured timeout, not
// left to rely on the caller's own context alone.
func (c *Client) WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return c.withTimeout(ctx)
}

func buildConfigFromFlags(context, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}

// NewClientFromBytes creates a Kubernetes client from kubeconfig bytes (Headlamp/Lens model).
// This allows client-side cluster management without storing kubeconfig on backend.
func NewClientFromBytes(kubeconfigBytes []byte, context string) (*Client, error) {
	// Load kubeconfig from bytes
	rawConfig, err := clientcmd.Load(kubeconfigBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Determine context to use
	contextToUse := context
	if contextToUse == "" {
		contextToUse = rawConfig.CurrentContext
	}
	if contextToUse == "" {
		return nil, fmt.Errorf("no context specified and no current context in kubeconfig")
	}

	// Verify context exists
	if _, exists := rawConfig.Contexts[contextToUse]; !exists {
		return nil, fmt.Errorf("context %s not found in kubeconfig", contextToUse)
	}

	// Create config with context override
	config, err := clientcmd.NewNonInteractiveClientConfig(
		*rawConfig,
		contextToUse,
		&clientcmd.ConfigOverrides{},
		&clientcmd.ClientConfigLoadingRules{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build config for context %s: %w", contextToUse, err)
	}
	applyDefaultClientRateLimit(config)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Client{
		Clientset:      clientset,
		Dynamic:        dynamicClient,
		Config:         config,
		Context:        contextToUse,
		kubeconfigPath: "", // Not stored when using bytes
		circuitBreaker: NewCircuitBreaker(""), // clusterID will be set via SetClusterID if available
		lastSuccessTime: time.Now(),
	}, nil
}

// GetServerVersion returns Kubernetes server version
func (c *Client) GetServerVersion(ctx context.Context) (string, error) {
	version, err := c.Clientset.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return version.GitVersion, nil
}

// TestConnection verifies connectivity to the cluster (with timeout, retry, and circuit breaker).
// BE-SCALE-001: Uses circuit breaker to prevent cascading failures.
func (c *Client) TestConnection(ctx context.Context) error {
	start := time.Now()
	if err := c.waitRateLimit(ctx); err != nil {
		return err
	}

	// Use circuit breaker
	err := c.circuitBreaker.Execute(ctx, func() error {
		ctx, cancel := c.withTimeout(ctx)
		defer cancel()
		return doWithRetry(ctx, defaultRetryAttempts, func() error {
			_, err := c.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
			return err
		})
	})

	c.updateHealth(err, time.Since(start))
	return err
}

// GetClusterInfo returns basic cluster information (with timeout, retry, and circuit breaker).
// BE-SCALE-001: Uses circuit breaker to prevent cascading failures.
func (c *Client) GetClusterInfo(ctx context.Context) (map[string]interface{}, error) {
	start := time.Now()
	if err := c.waitRateLimit(ctx); err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err := c.circuitBreaker.Execute(ctx, func() error {
		ctx, cancel := c.withTimeout(ctx)
		defer cancel()
		var fnErr error
		result, fnErr = doWithRetryValue(ctx, defaultRetryAttempts, func() (map[string]interface{}, error) {
			version, err := c.GetServerVersion(ctx)
			if err != nil {
				return nil, err
			}
			nodes, err := c.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, err
			}
			namespaces, err := c.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, err
			}
			serverURL := ""
			if c.Config != nil {
				serverURL = c.Config.Host
			}
			return map[string]interface{}{
				"version":         version,
				"node_count":      len(nodes.Items),
				"namespace_count": len(namespaces.Items),
				"server_url":      serverURL,
			}, nil
		})
		return fnErr
	})

	c.updateHealth(err, time.Since(start))
	if err != nil {
		return nil, err
	}
	return result, nil
}

// updateHealth updates the health status of the client. lastCheckedTime moves
// forward on every call (success or failure); lastSuccessTime only on success.
// duration is the time the check itself took — OBS-2
// (docs/PRODUCTION-HARDENING-ROADMAP.md, Phase 8): this is the single choke
// point every TestConnection/GetClusterInfo call passes through on its way to
// updating LastCheckedAt/LastSuccessAt/LastError (HEALTH-2's freshness
// fields), so it is also the correct place to feed the per-cluster
// health-check latency/failure metrics from the same data, not a second,
// independently-tracked signal.
func (c *Client) updateHealth(err error, duration time.Duration) {
	clusterID := ""
	if c.circuitBreaker != nil {
		clusterID = c.circuitBreaker.clusterID
	}
	metrics.ClusterHealthCheckDurationSeconds.WithLabelValues(clusterID).Observe(duration.Seconds())

	c.healthMu.Lock()
	defer c.healthMu.Unlock()
	c.lastCheckedTime = time.Now()
	if err == nil {
		c.lastSuccessTime = time.Now()
		c.lastError = nil
	} else {
		c.lastError = err
		metrics.ClusterHealthCheckFailuresTotal.WithLabelValues(clusterID).Inc()
	}
}

// HealthStatus returns the health status of the cluster connection.
// BE-SCALE-001: Per-cluster connection health monitoring.
func (c *Client) HealthStatus() (isHealthy bool, lastSuccess time.Time, lastErr error, circuitState CircuitBreakerState) {
	c.healthMu.RLock()
	defer c.healthMu.RUnlock()

	state := c.circuitBreaker.State()
	isHealthy = state == StateClosed && c.lastError == nil
	return isHealthy, c.lastSuccessTime, c.lastError, state
}

// LastCheckedAt returns when TestConnection/GetClusterInfo was last attempted
// (success or failure), or the zero time if never checked. Added for
// HEALTH-2 without changing HealthStatus()'s existing signature/callers.
func (c *Client) LastCheckedAt() time.Time {
	c.healthMu.RLock()
	defer c.healthMu.RUnlock()
	return c.lastCheckedTime
}

// NewClientForTest creates a Client that uses the given Clientset. Used by tests (e.g. topology)
// that only need Clientset. Config and Dynamic are nil; callers must not use client methods that need them.
func NewClientForTest(clientset kubernetes.Interface) *Client {
	client := &Client{
		Clientset:      clientset,
		circuitBreaker: NewCircuitBreaker(""), // clusterID will be set via SetClusterID if available
		lastSuccessTime: time.Now(),
	}
	return client
}
