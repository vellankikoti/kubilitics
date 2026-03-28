package rest

import (
	"net/http"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kubilitics/kubilitics-backend/internal/pkg/logger"
)

// ---------------------------------------------------------------------------
// Response types
// ---------------------------------------------------------------------------

// FleetClusterInfo describes a single cluster within the fleet overview.
type FleetClusterInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	Nodes        int    `json:"nodes"`
	Pods         int    `json:"pods"`
	Deployments  int    `json:"deployments"`
	Namespaces   int    `json:"namespaces"`
	HealthStatus string `json:"healthStatus"`
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
		mu       sync.Mutex
		infos    []FleetClusterInfo
		totals   FleetTotals
	)

	g, gCtx := errgroup.WithContext(ctx)
	for _, c := range clusters {
		c := c // capture loop variable
		g.Go(func() error {
			summary, summaryErr := h.clusterService.GetClusterSummary(gCtx, c.ID)
			if summaryErr != nil {
				// Cluster unreachable — record as unhealthy but do not fail the whole request.
				mu.Lock()
				infos = append(infos, FleetClusterInfo{
					ID:           c.ID,
					Name:         c.Name,
					Status:       c.Status,
					HealthStatus: "unhealthy",
				})
				totals.Unhealthy++
				mu.Unlock()
				return nil
			}

			info := FleetClusterInfo{
				ID:           c.ID,
				Name:         c.Name,
				Status:       c.Status,
				Nodes:        summary.NodeCount,
				Pods:         summary.PodCount,
				Deployments:  summary.DeploymentCount,
				Namespaces:   summary.NamespaceCount,
				HealthStatus: summary.HealthStatus,
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
