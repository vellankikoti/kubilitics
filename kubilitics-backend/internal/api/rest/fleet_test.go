package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/kubilitics/kubilitics-backend/internal/k8s"
	"github.com/kubilitics/kubilitics-backend/internal/models"
)

// fleetMockClusterService is deliberately separate from the shared
// mockClusterService in cluster_handler_test.go — that mock returns one
// shared *models.ClusterSummary for every cluster ID, which can't express
// "cluster B is unreachable while cluster A is healthy," the exact
// per-cluster isolation FLEET-N1's tests need.
type fleetMockClusterService struct {
	mockClusterService
	summaries   map[string]*models.ClusterSummary
	summaryErrs map[string]error
}

func (m *fleetMockClusterService) GetClusterSummary(ctx context.Context, id string) (*models.ClusterSummary, error) {
	if err, ok := m.summaryErrs[id]; ok {
		return nil, err
	}
	if s, ok := m.summaries[id]; ok {
		return s, nil
	}
	return nil, errClusterNotFound
}

func newFleetHandler(cs *fleetMockClusterService) *Handler {
	return &Handler{clusterService: cs}
}

func doFleetOverview(t *testing.T, h *Handler) FleetOverviewResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/fleet/overview", nil)
	rec := httptest.NewRecorder()
	h.GetFleetOverview(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GetFleetOverview status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp FleetOverviewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func findFleetCluster(t *testing.T, resp FleetOverviewResponse, id string) FleetClusterInfo {
	t.Helper()
	for _, c := range resp.Clusters {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("cluster %q not found in response: %+v", id, resp.Clusters)
	return FleetClusterInfo{}
}

// FLEET-N1 (docs/FLEET-N1-IMPLEMENTATION.md): the response must carry every
// field useFleetOverview.ts previously had to make N separate per-cluster
// requests to obtain — reachable/stale/errorMessage especially, since
// HEALTH-1/HEALTH-2 require the frontend be able to tell "confirmed
// healthy" apart from "unknown" apart from "confirmed unreachable."
func TestGetFleetOverview_ReturnsFullFieldsNeededByFrontend_NoSecondRequestRequired(t *testing.T) {
	staleAsOf := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cs := &fleetMockClusterService{
		mockClusterService: mockClusterService{
			clusters: []*models.Cluster{
				{ID: "c1", Name: "prod", Context: "prod-ctx", Provider: "eks", Version: "1.29", Status: "connected", LastConnected: staleAsOf},
			},
		},
		summaries: map[string]*models.ClusterSummary{
			"c1": {
				NodeCount: 5, PodCount: 120, DeploymentCount: 30, ServiceCount: 12, NamespaceCount: 4,
				HealthStatus: "healthy", HealthReason: "all good",
				Reachable: true, Stale: true, StaleAsOf: &staleAsOf,
			},
		},
	}
	resp := doFleetOverview(t, newFleetHandler(cs))
	c := findFleetCluster(t, resp, "c1")

	if c.Context != "prod-ctx" || c.Provider != "eks" || c.Version != "1.29" {
		t.Fatalf("identity fields not populated: %+v", c)
	}
	if c.Nodes != 5 || c.Pods != 120 || c.Deployments != 30 || c.Services != 12 || c.Namespaces != 4 {
		t.Fatalf("count fields not populated: %+v", c)
	}
	if !c.Reachable {
		t.Fatalf("Reachable not propagated: %+v", c)
	}
	if !c.Stale || c.StaleAsOf == "" {
		t.Fatalf("Stale/StaleAsOf not propagated (HEALTH-2 freshness signal lost): %+v", c)
	}
	if c.HealthReason != "all good" {
		t.Fatalf("HealthReason not propagated: %+v", c)
	}
	if c.LastConnected == "" {
		t.Fatalf("LastConnected not propagated: %+v", c)
	}
}

// TestGetFleetOverview_IncludesHealthTimestamps locks in that Fleet rows
// carry the same LastCheckedAt/LastSuccessAt/LastError freshness fields
// Presence already exposes, so Fleet and Picker rows can share identical
// "last seen" copy (Task 2, cluster-connection-state plan).
func TestGetFleetOverview_IncludesHealthTimestamps(t *testing.T) {
	lastChecked := time.Date(2026, 10, 12, 0, 10, 0, 0, time.UTC)
	lastSuccess := time.Date(2026, 10, 12, 0, 10, 0, 0, time.UTC)
	cs := &fleetMockClusterService{
		mockClusterService: mockClusterService{
			clusters: []*models.Cluster{
				{ID: "c1", Name: "prod", Context: "prod-ctx", Status: "connected"},
			},
		},
		summaries: map[string]*models.ClusterSummary{
			"c1": {
				NodeCount: 1, PodCount: 1, HealthStatus: "healthy",
				Reachable:     true,
				LastCheckedAt: &lastChecked,
				LastSuccessAt: &lastSuccess,
			},
		},
	}
	resp := doFleetOverview(t, newFleetHandler(cs))
	c := findFleetCluster(t, resp, "c1")

	if c.LastCheckedAt == "" {
		t.Error("expected LastCheckedAt to be populated for a reachable cluster")
	}
	if c.LastSuccessAt == "" {
		t.Error("expected LastSuccessAt to be populated for a reachable cluster")
	}
}

// TestGetFleetOverview_UnreachableClusterIncludesHealthTimestamps locks in
// that an unreachable cluster's row still carries real LastCheckedAt/
// LastSuccessAt/LastError when a client exists for it (it was reachable
// before and has since gone down) — Final review, Important #1: this
// branch previously never set these fields at all, so Task 7's "Last
// checked" Fleet copy could never render with real backend data, since
// Reachable is always true on the success path where they WERE set.
func TestGetFleetOverview_UnreachableClusterIncludesHealthTimestamps(t *testing.T) {
	client := k8s.NewClientForTest(k8sfake.NewSimpleClientset())
	_ = client.TestConnection(context.Background()) // seeds LastCheckedAt/LastSuccessAt via updateHealth

	cs := &fleetMockClusterService{
		mockClusterService: mockClusterService{
			clusters:   []*models.Cluster{{ID: "down-1", Name: "down"}},
			clusterMap: map[string]*models.Cluster{"down-1": {ID: "down-1", Name: "down"}},
			clientMap:  map[string]*k8s.Client{"down-1": client},
		},
		summaryErrs: map[string]error{
			"down-1": errors.New("dial tcp: connection refused"),
		},
	}
	resp := doFleetOverview(t, newFleetHandler(cs))
	c := findFleetCluster(t, resp, "down-1")

	if c.Reachable {
		t.Fatalf("expected Reachable=false, got true: %+v", c)
	}
	if c.LastCheckedAt == "" {
		t.Error("expected LastCheckedAt to be populated from the existing client's cached health")
	}
	if c.LastSuccessAt == "" {
		t.Error("expected LastSuccessAt to be populated from the existing client's cached health")
	}
}

// One unreachable cluster must not block or corrupt the others' data, and
// must be clearly distinguishable (Reachable=false, SummaryUnavailable=true,
// ErrorMessage set) — never silently rendered as healthy.
func TestGetFleetOverview_UnreachableClusterIsolated_HealthyClustersUnaffected(t *testing.T) {
	cs := &fleetMockClusterService{
		mockClusterService: mockClusterService{
			clusters: []*models.Cluster{
				{ID: "healthy-1", Name: "a"},
				{ID: "broken-1", Name: "b"},
				{ID: "healthy-2", Name: "c"},
			},
		},
		summaries: map[string]*models.ClusterSummary{
			"healthy-1": {NodeCount: 3, PodCount: 50, HealthStatus: "healthy", Reachable: true},
			"healthy-2": {NodeCount: 2, PodCount: 20, HealthStatus: "healthy", Reachable: true},
		},
		summaryErrs: map[string]error{
			"broken-1": errors.New("dial tcp: connection refused"),
		},
	}
	resp := doFleetOverview(t, newFleetHandler(cs))

	if len(resp.Clusters) != 3 {
		t.Fatalf("want 3 clusters in response despite one failure: %+v", resp.Clusters)
	}
	h1 := findFleetCluster(t, resp, "healthy-1")
	if h1.HealthStatus != "healthy" || !h1.Reachable || h1.Pods != 50 {
		t.Fatalf("healthy-1 corrupted by broken-1's failure: %+v", h1)
	}
	h2 := findFleetCluster(t, resp, "healthy-2")
	if h2.HealthStatus != "healthy" || !h2.Reachable || h2.Pods != 20 {
		t.Fatalf("healthy-2 corrupted by broken-1's failure: %+v", h2)
	}
	broken := findFleetCluster(t, resp, "broken-1")
	if broken.Reachable {
		t.Fatalf("broken-1 must not be reported reachable: %+v", broken)
	}
	if !broken.SummaryUnavailable {
		t.Fatalf("broken-1 must be flagged SummaryUnavailable (distinct from 'reported unhealthy'): %+v", broken)
	}
	if broken.ErrorMessage == "" {
		t.Fatalf("broken-1 must carry a non-empty ErrorMessage: %+v", broken)
	}
	if broken.HealthStatus == "healthy" {
		t.Fatalf("broken-1 must never be reported healthy (no fabricated healthy state): %+v", broken)
	}

	// Totals must reflect exactly 2 healthy + 1 unhealthy, not corrupted counts.
	if resp.Totals.Healthy != 2 || resp.Totals.Unhealthy != 1 {
		t.Fatalf("totals incorrect: %+v", resp.Totals)
	}
	if resp.Totals.Pods != 70 {
		t.Fatalf("totals.Pods should only sum the 2 reachable clusters' pods: %+v", resp.Totals)
	}
}

// Zero clusters must produce a valid, empty response — not nil/null arrays
// that would force extra frontend null-checks, and not an error.
func TestGetFleetOverview_NoClusters_ReturnsEmptyNotNull(t *testing.T) {
	cs := &fleetMockClusterService{mockClusterService: mockClusterService{clusters: []*models.Cluster{}}}
	resp := doFleetOverview(t, newFleetHandler(cs))
	if resp.Clusters == nil {
		t.Fatalf("Clusters must be an empty slice, not nil/null")
	}
	if len(resp.Clusters) != 0 {
		t.Fatalf("want 0 clusters: %+v", resp.Clusters)
	}
}

// FLEET-N1's whole point: ONE backend request serves the whole fleet,
// regardless of cluster count — the client-visible request count must not
// scale with N. This proves the handler itself is a single request/response
// cycle (no internal per-cluster HTTP round trip back out to the caller);
// the N-way fan-out to Kubernetes API servers happens entirely server-side,
// invisible to and uncounted by the browser.
func TestGetFleetOverview_SingleRequestServesArbitraryClusterCount(t *testing.T) {
	const n = 25
	clusters := make([]*models.Cluster, 0, n)
	summaries := make(map[string]*models.ClusterSummary, n)
	for i := 0; i < n; i++ {
		id := "cluster-" + string(rune('a'+i))
		clusters = append(clusters, &models.Cluster{ID: id, Name: id})
		summaries[id] = &models.ClusterSummary{NodeCount: 1, PodCount: 10, HealthStatus: "healthy", Reachable: true}
	}
	cs := &fleetMockClusterService{mockClusterService: mockClusterService{clusters: clusters}, summaries: summaries}

	req := httptest.NewRequest(http.MethodGet, "/fleet/overview", nil)
	rec := httptest.NewRecorder()
	newFleetHandler(cs).GetFleetOverview(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp FleetOverviewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Clusters) != n {
		t.Fatalf("want %d clusters from a single request, got %d", n, len(resp.Clusters))
	}
	if resp.Totals.Pods != n*10 {
		t.Fatalf("totals.Pods = %d, want %d", resp.Totals.Pods, n*10)
	}
}
