package rest

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kubilitics/kubilitics-backend/internal/pkg/logger"
)

// ---------------------------------------------------------------------------
// Response types
// ---------------------------------------------------------------------------

// FleetClusterInfo describes a single cluster within the fleet overview.
//
// FLEET-N1 (docs/FLEET-N1-IMPLEMENTATION.md, docs/ENTERPRISE-SCALE-RELIABILITY-REPORT.md §10): originally a trimmed subset of
// what ClusterService.GetClusterSummary already returns per cluster — this
// handler had the full models.Cluster + models.ClusterSummary in scope the
// whole time, it just wasn't copying several fields the frontend actually
// needs (reachable/stale/errorMessage especially — HEALTH-1/HEALTH-2's
// "never fabricate a healthy state" guarantee depends on these reaching the
// UI). That gap is why the frontend built its own N-request client-side
// aggregation instead of using this endpoint. Extended here, additively,
// rather than inventing a second response shape.
type FleetClusterInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Context       string `json:"context"`
	Status        string `json:"status"`
	Provider      string `json:"provider,omitempty"`
	Version       string `json:"version,omitempty"`
	LastConnected string `json:"last_connected,omitempty"`
	Nodes         int    `json:"nodes"`
	Pods          int    `json:"pods"`
	Deployments   int    `json:"deployments"`
	Services      int    `json:"services"`
	Namespaces    int    `json:"namespaces"`
	HealthStatus  string `json:"healthStatus"`
	HealthReason  string `json:"healthReason,omitempty"`
	// Reachable/Stale/StaleAsOf/ErrorMessage mirror models.ClusterSummary's
	// own fields exactly (HEALTH-1/HEALTH-2) — the frontend must be able to
	// tell "confirmed healthy" apart from "unknown/last-known-good cache"
	// apart from "confirmed unreachable," never collapsing any of the three.
	Reachable    bool   `json:"reachable"`
	Stale        bool   `json:"stale,omitempty"`
	StaleAsOf    string `json:"stale_as_of,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	// SummaryUnavailable mirrors the frontend's own existing distinction
	// (useFleetOverview.ts) between "the /summary call itself failed" and
	// "it succeeded but reported bad health" — true only in the former case.
	SummaryUnavailable bool `json:"summary_unavailable,omitempty"`
}

// FleetTotals contains aggregate counts across all clusters.
type FleetTotals struct {
	Nodes       int `json:"nodes"`
	Pods        int `json:"pods"`
	Deployments int `json:"deployments"`
	Namespaces  int `json:"namespaces"`
	Healthy     int `json:"healthy"`
	Degraded    int `json:"degraded"`
	Unhealthy   int `json:"unhealthy"`
}

// FleetOverviewResponse is the response body for GET /fleet/overview.
type FleetOverviewResponse struct {
	Clusters []FleetClusterInfo `json:"clusters"`
	Totals   FleetTotals        `json:"totals"`
}

// FleetSearchResultItem is a single resource match from cross-cluster search.
type FleetSearchResultItem struct {
	ClusterID   string `json:"clusterId"`
	ClusterName string `json:"clusterName"`
	Kind        string `json:"kind"`
	Namespace   string `json:"namespace,omitempty"`
	Name        string `json:"name"`
	Status      string `json:"status,omitempty"`
}

// FleetSearchResponse is the response body for GET /fleet/search.
type FleetSearchResponse struct {
	Results []FleetSearchResultItem `json:"results"`
	Total   int                     `json:"total"`
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

const fleetSearchMaxResults = 100

// maxConcurrentFleetClusterSummaries bounds GetFleetOverview's per-cluster
// fan-out (Phase I-B). An evidence-informed starting bound, not empirically
// load-tested at 50-100 cluster scale in this pass — see the implementation
// doc for what remains UNVERIFIED and why.
const maxConcurrentFleetClusterSummaries = 10

// GetFleetOverview handles GET /fleet/overview.
// It iterates all registered clusters, fetches summary data for each using
// the existing ClusterService.GetClusterSummary logic, and returns aggregated
// fleet health information.
func (h *Handler) GetFleetOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clusters, err := h.clusterService.ListClusters(ctx)
	if err != nil {
		requestID := logger.FromContext(ctx)
		respondErrorWithCode(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error(), requestID)
		return
	}

	var (
		mu     sync.Mutex
		infos  []FleetClusterInfo
		totals FleetTotals
	)

	// Phase I-B (docs/FLEET-PERFORMANCE-IMPLEMENTATION.md): this fan-out was
	// previously fully unbounded — one goroutine per cluster, each of which
	// itself fans out to up to 30 concurrent K8s calls (buildClusterSummary).
	// At N clusters that was N×30 simultaneous outbound K8s API calls with no
	// ceiling in either dimension. Bounded here to maxConcurrentFleetClusterSummaries
	// at a time; combined with maxConcurrentSummaryListCalls's own per-cluster
	// bound, worst-case concurrent K8s calls is now a fixed product of the two
	// constants rather than growing unboundedly with fleet size.
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrentFleetClusterSummaries)
	for _, c := range clusters {
		c := c // capture loop variable
		g.Go(func() error {
			summary, summaryErr := h.clusterService.GetClusterSummary(gCtx, c.ID)
			if summaryErr != nil {
				// Cluster unreachable — record as unhealthy but do not fail the whole request.
				// FLEET-N1: SummaryUnavailable=true + ErrorMessage set, so the frontend
				// can distinguish "the /summary call itself failed" from "it succeeded
				// but reported bad health" — the same distinction useFleetOverview.ts
				// already made for its own per-cluster requests.
				mu.Lock()
				infos = append(infos, FleetClusterInfo{
					ID:                 c.ID,
					Name:               c.Name,
					Context:            c.Context,
					Status:             c.Status,
					Provider:           c.Provider,
					Version:            c.Version,
					HealthStatus:       "unhealthy",
					Reachable:          false,
					ErrorMessage:       summaryErr.Error(),
					SummaryUnavailable: true,
				})
				totals.Unhealthy++
				mu.Unlock()
				return nil
			}

			info := FleetClusterInfo{
				ID:           c.ID,
				Name:         c.Name,
				Context:      c.Context,
				Status:       c.Status,
				Provider:     c.Provider,
				Version:      c.Version,
				Nodes:        summary.NodeCount,
				Pods:         summary.PodCount,
				Deployments:  summary.DeploymentCount,
				Services:     summary.ServiceCount,
				Namespaces:   summary.NamespaceCount,
				HealthStatus: summary.HealthStatus,
				HealthReason: summary.HealthReason,
				Reachable:    summary.Reachable,
				Stale:        summary.Stale,
				ErrorMessage: summary.ErrorMessage,
			}
			if !c.LastConnected.IsZero() {
				info.LastConnected = c.LastConnected.Format(time.RFC3339)
			}
			if summary.StaleAsOf != nil {
				info.StaleAsOf = summary.StaleAsOf.Format(time.RFC3339)
			}

			mu.Lock()
			infos = append(infos, info)
			totals.Nodes += summary.NodeCount
			totals.Pods += summary.PodCount
			totals.Deployments += summary.DeploymentCount
			totals.Namespaces += summary.NamespaceCount
			switch summary.HealthStatus {
			case "healthy":
				totals.Healthy++
			case "degraded":
				totals.Degraded++
			default:
				totals.Unhealthy++
			}
			mu.Unlock()
			return nil
		})
	}

	// errgroup goroutines never return non-nil errors, but handle defensively.
	if waitErr := g.Wait(); waitErr != nil {
		requestID := logger.FromContext(ctx)
		respondErrorWithCode(w, http.StatusInternalServerError, ErrCodeInternalError, waitErr.Error(), requestID)
		return
	}

	if infos == nil {
		infos = []FleetClusterInfo{}
	}
	respondJSON(w, http.StatusOK, FleetOverviewResponse{Clusters: infos, Totals: totals})
}

// fleetSearchKinds are the resource kinds queried for fleet-wide search.
var fleetSearchKinds = []string{
	"pods", "deployments", "services", "nodes", "namespaces",
	"configmaps", "secrets", "ingresses", "statefulsets", "daemonsets",
	"jobs", "cronjobs",
}

// GetFleetSearch handles GET /fleet/search?q=...&kind=...
// It searches across ALL registered clusters in parallel and returns up to
// 100 matching results.
func (h *Handler) GetFleetSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		respondError(w, http.StatusBadRequest, "Missing or empty query parameter: q")
		return
	}
	qLower := strings.ToLower(q)

	kindFilter := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("kind")))

	// Determine which kinds to search.
	kinds := fleetSearchKinds
	if kindFilter != "" {
		kinds = []string{kindFilter}
	}

	clusters, err := h.clusterService.ListClusters(ctx)
	if err != nil {
		requestID := logger.FromContext(ctx)
		respondErrorWithCode(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error(), requestID)
		return
	}

	var (
		mu      sync.Mutex
		results []FleetSearchResultItem
		done    bool // true once we have enough results
	)

	g, gCtx := errgroup.WithContext(ctx)
	for _, c := range clusters {
		c := c
		g.Go(func() error {
			client, clientErr := h.clusterService.GetClient(c.ID)
			if clientErr != nil {
				return nil // skip unreachable clusters
			}

			var wg sync.WaitGroup
			for _, kind := range kinds {
				kind := kind
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() {
						if r := recover(); r != nil {
							slog.Default().Error("panic in fleet per-kind search goroutine", "error", r)
						}
					}()

					mu.Lock()
					if done {
						mu.Unlock()
						return
					}
					mu.Unlock()

					opts := metav1.ListOptions{Limit: int64(fleetSearchMaxResults)}
					list, listErr := client.ListResources(gCtx, kind, "", opts)
					if listErr != nil {
						return
					}
					for i := range list.Items {
						item := &list.Items[i]
						name, _, _ := unstructured.NestedString(item.Object, "metadata", "name")
						namespace, _, _ := unstructured.NestedString(item.Object, "metadata", "namespace")
						if name == "" {
							continue
						}
						matches := strings.Contains(strings.ToLower(name), qLower) ||
							strings.Contains(strings.ToLower(namespace), qLower)
						if !matches {
							continue
						}

						status := extractResourceStatus(item)

						mu.Lock()
						if done {
							mu.Unlock()
							return
						}
						results = append(results, FleetSearchResultItem{
							ClusterID:   c.ID,
							ClusterName: c.Name,
							Kind:        kind,
							Namespace:   namespace,
							Name:        name,
							Status:      status,
						})
						if len(results) >= fleetSearchMaxResults {
							done = true
						}
						mu.Unlock()
					}
				}()
			}
			wg.Wait()
			return nil
		})
	}

	_ = g.Wait()

	if results == nil {
		results = []FleetSearchResultItem{}
	}
	if len(results) > fleetSearchMaxResults {
		results = results[:fleetSearchMaxResults]
	}
	respondJSON(w, http.StatusOK, FleetSearchResponse{Results: results, Total: len(results)})
}

// extractResourceStatus attempts to derive a simple status string from an
// unstructured Kubernetes resource (e.g. pod phase, deployment available
// replicas). Returns empty string if status cannot be determined.
func extractResourceStatus(item *unstructured.Unstructured) string {
	// Pod phase
	if phase, ok, _ := unstructured.NestedString(item.Object, "status", "phase"); ok && phase != "" {
		return phase
	}
	// Deployment/StatefulSet/DaemonSet: check availableReplicas vs replicas
	if replicas, ok, _ := unstructured.NestedInt64(item.Object, "status", "availableReplicas"); ok {
		desired, dOk, _ := unstructured.NestedInt64(item.Object, "spec", "replicas")
		if dOk && replicas >= desired {
			return "Available"
		}
		return "Progressing"
	}
	// Job: check succeeded
	if succeeded, ok, _ := unstructured.NestedInt64(item.Object, "status", "succeeded"); ok && succeeded > 0 {
		return "Complete"
	}
	return ""
}
