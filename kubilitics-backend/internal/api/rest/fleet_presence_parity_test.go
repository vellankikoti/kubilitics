package rest

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubilitics/kubilitics-backend/internal/cluster/discovery"
	"github.com/kubilitics/kubilitics-backend/internal/config"
	"github.com/kubilitics/kubilitics-backend/internal/k8s"
	"github.com/kubilitics/kubilitics-backend/internal/models"
	"github.com/kubilitics/kubilitics-backend/internal/service"
)

// inMemoryClusterRepo is a minimal, test-only implementation of
// repository.ClusterRepository, scoped to this file. It cannot be shared
// with internal/service's own test-only mockClusterRepo (unexported,
// different package) or cmd/server's clusterRepoAdapter (package main,
// not importable) — both exist for the same reason this one does.
type inMemoryClusterRepo struct {
	mu       sync.Mutex
	clusters map[string]*models.Cluster
}

func newInMemoryClusterRepo() *inMemoryClusterRepo {
	return &inMemoryClusterRepo{clusters: make(map[string]*models.Cluster)}
}

func (r *inMemoryClusterRepo) Create(ctx context.Context, c *models.Cluster) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clusters[c.ID] = c
	return nil
}

func (r *inMemoryClusterRepo) Get(ctx context.Context, id string) (*models.Cluster, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.clusters[id]
	if !ok {
		return nil, errClusterNotFound
	}
	return c, nil
}

func (r *inMemoryClusterRepo) List(ctx context.Context) ([]*models.Cluster, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*models.Cluster, 0, len(r.clusters))
	for _, c := range r.clusters {
		out = append(out, c)
	}
	return out, nil
}

func (r *inMemoryClusterRepo) Update(ctx context.Context, c *models.Cluster) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clusters[c.ID] = c
	return nil
}

func (r *inMemoryClusterRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clusters, id)
	return nil
}

// discoveryRepoAdapter adapts inMemoryClusterRepo to discovery.ClusterRepository
// (ListAll() []discovery.StoredCluster), mirroring cmd/server/main.go's
// clusterRepoAdapter exactly (SessionID: r.ID) so this test exercises the
// same mapping production wiring uses, not a reinvented one.
type discoveryRepoAdapter struct {
	repo *inMemoryClusterRepo
}

func (a *discoveryRepoAdapter) ListAll() ([]discovery.StoredCluster, error) {
	rows, err := a.repo.List(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]discovery.StoredCluster, 0, len(rows))
	for _, r := range rows {
		out = append(out, discovery.StoredCluster{
			Name:           r.Name,
			ServerURL:      r.ServerURL,
			SessionID:      r.ID,
			Provider:       r.Provider,
			KubeconfigPath: r.KubeconfigPath,
			ContextName:    r.Context,
		})
	}
	return out, nil
}

func writeFakeKubeconfigForParityTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "kubeconfig")
	if err := os.WriteFile(path, []byte("apiVersion: v1\nkind: Config"), 0o600); err != nil {
		t.Fatalf("write fake kubeconfig: %v", err)
	}
	return path
}

// newFleetParityTestFixture wires a real service.ClusterService and a real
// discovery.Manager against the SAME inMemoryClusterRepo, then registers n
// fake clusters through AddCluster (so both Fleet's ListClusters and
// Presence's Snapshot read the same persisted rows) and returns a *Handler
// plus the discovery.Manager for the test to compare against.
func newFleetParityTestFixture(t *testing.T, n int) (*Handler, *discovery.Manager) {
	t.Helper()
	ctx := context.Background()
	repo := newInMemoryClusterRepo()

	factory := func(kubeconfigPath, contextName string) (*k8s.Client, error) {
		return k8s.NewClientForTest(fake.NewSimpleClientset()), nil
	}
	cfg := &config.Config{K8sTimeoutSec: 30}
	clusterService := service.NewClusterServiceWithClientFactory(repo, cfg, factory)

	for i := 0; i < n; i++ {
		contextName := "parity-ctx-" + string(rune('a'+i))
		if _, err := clusterService.AddCluster(ctx, writeFakeKubeconfigForParityTest(t), contextName); err != nil {
			t.Fatalf("AddCluster(%s): %v", contextName, err)
		}
	}

	discoveryMgr := discovery.NewManager([]discovery.DiscoverySource{
		discovery.NewManualSource(&discoveryRepoAdapter{repo: repo}),
	})
	if err := discoveryMgr.Refresh(ctx); err != nil {
		t.Fatalf("discoveryMgr.Refresh: %v", err)
	}

	h := &Handler{clusterService: clusterService}
	return h, discoveryMgr
}

// TestFleetClusterListMatchesPresenceRegisteredList locks Fleet's and
// Presence's idea of "which clusters exist" to the same set. Reported bug
// (user screenshot, 2026-10-12): Picker showed 4 registered clusters,
// Fleet showed 3. If ListClusters and the presence Snapshot's Registered
// list ever diverge in membership, this test must fail.
func TestFleetClusterListMatchesPresenceRegisteredList(t *testing.T) {
	ctx := context.Background()
	h, discoveryMgr := newFleetParityTestFixture(t, 4)

	clusters, err := h.clusterService.ListClusters(ctx)
	if err != nil {
		t.Fatalf("ListClusters: %v", err)
	}
	snap := discoveryMgr.Snapshot()

	fleetIDs := make(map[string]bool, len(clusters))
	for _, c := range clusters {
		fleetIDs[c.ID] = true
	}
	presenceIDs := make(map[string]bool, len(snap.Registered))
	for _, c := range snap.Registered {
		presenceIDs[c.SessionID] = true
	}

	if len(fleetIDs) != len(presenceIDs) {
		t.Fatalf("membership count mismatch: Fleet has %d, Presence has %d (fleet=%v presence=%v)",
			len(fleetIDs), len(presenceIDs), fleetIDs, presenceIDs)
	}
	for id := range fleetIDs {
		if !presenceIDs[id] {
			t.Errorf("cluster %s present in Fleet's ListClusters but missing from Presence's registered list", id)
		}
	}
	for id := range presenceIDs {
		if !fleetIDs[id] {
			t.Errorf("cluster %s present in Presence's registered list but missing from Fleet's ListClusters", id)
		}
	}
}
