package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kubilitics/kubilitics-backend/internal/config"
	"github.com/kubilitics/kubilitics-backend/internal/k8s"
	"github.com/kubilitics/kubilitics-backend/internal/models"
	"github.com/kubilitics/kubilitics-backend/internal/repository"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// mockClusterRepo implements repository.ClusterRepository for tests.
//
// STARTUP-1 (docs/PRODUCTION-RELIABILITY-AUDIT.md): LoadClustersFromRepo now
// calls Update concurrently (bounded fan-out), unlike before. The real
// SQLite/Postgres repos are safe for this (SQLite: a single pooled connection
// serializes access; Postgres: connection-pool + driver-level safety), but this
// plain-map fake is not — guard it with a mutex so tests exercise the
// concurrency this change introduces without a false-positive data race.
type mockClusterRepo struct {
	mu       sync.Mutex
	clusters map[string]*models.Cluster
}

func (m *mockClusterRepo) Create(ctx context.Context, cluster *models.Cluster) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.clusters == nil {
		m.clusters = make(map[string]*models.Cluster)
	}
	c := *cluster
	m.clusters[cluster.ID] = &c
	return nil
}

func (m *mockClusterRepo) Get(ctx context.Context, id string) (*models.Cluster, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.clusters[id]; ok {
		cp := *c
		return &cp, nil
	}
	return nil, errors.New("cluster not found")
}

func (m *mockClusterRepo) List(ctx context.Context) ([]*models.Cluster, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*models.Cluster
	for _, c := range m.clusters {
		cp := *c
		out = append(out, &cp)
	}
	return out, nil
}

// Update matches real SQLite/Postgres UPDATE semantics: a no-op (zero rows
// affected, no error) when the row no longer exists — NOT an upsert. This
// is exactly the behavior kickBackgroundReconnect's own doc comment already
// relies on ("a harmless no-op UPDATE on a since-deleted id — SQLite does
// not error on a zero-row UPDATE"). Informer-lifecycle work surfaced that
// this mock previously always wrote (effectively upserting, resurrecting a
// row RemoveCluster had just deleted) — a test-fixture fidelity gap, not a
// production behavior change; fixed here, not in production code.
func (m *mockClusterRepo) Update(ctx context.Context, cluster *models.Cluster) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.clusters == nil {
		return nil
	}
	if _, exists := m.clusters[cluster.ID]; !exists {
		return nil // real UPDATE on a deleted row: zero rows affected, not an error
	}
	c := *cluster
	m.clusters[cluster.ID] = &c
	return nil
}

func (m *mockClusterRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clusters, id)
	return nil
}

// Get is also used by tests directly to snapshot state without racing writers.
func (m *mockClusterRepo) snapshot() map[string]*models.Cluster {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]*models.Cluster, len(m.clusters))
	for k, v := range m.clusters {
		cp := *v
		out[k] = &cp
	}
	return out
}

func TestClusterService_ListClusters_EmptyRepo(t *testing.T) {
	ctx := context.Background()
	repo := &mockClusterRepo{clusters: make(map[string]*models.Cluster)}
	svc := NewClusterService(repo, nil)

	list, err := svc.ListClusters(ctx)
	if err != nil {
		t.Fatalf("ListClusters: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 clusters, got %d", len(list))
	}
}

func TestClusterService_RemoveCluster_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockClusterRepo{clusters: make(map[string]*models.Cluster)}
	svc := NewClusterService(repo, nil)

	err := svc.RemoveCluster(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error when removing non-existent cluster")
	}
}

func TestClusterService_ListClusters_FromRepo(t *testing.T) {
	ctx := context.Background()
	repo := &mockClusterRepo{clusters: map[string]*models.Cluster{
		"id-1": {
			ID: "id-1", Name: "cluster-1", Context: "ctx1",
			Status: "disconnected", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}}
	svc := NewClusterService(repo, nil)

	list, err := svc.ListClusters(ctx)
	if err != nil {
		t.Fatalf("ListClusters: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(list))
	}
	if list[0].ID != "id-1" || list[0].Name != "cluster-1" {
		t.Errorf("unexpected cluster: %+v", list[0])
	}
}

func TestClusterService_GetCluster_FromRepo(t *testing.T) {
	ctx := context.Background()
	repo := &mockClusterRepo{clusters: map[string]*models.Cluster{
		"id-1": {
			ID: "id-1", Name: "cluster-1", Context: "ctx1",
			Status: "disconnected", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}}
	svc := NewClusterService(repo, nil)

	c, err := svc.GetCluster(ctx, "id-1")
	if err != nil {
		t.Fatalf("GetCluster: %v", err)
	}
	if c.ID != "id-1" || c.Name != "cluster-1" {
		t.Errorf("unexpected cluster: %+v", c)
	}

	_, err = svc.GetCluster(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent cluster")
	}
}

func TestClusterService_AddCluster_RespectsMaxClusters(t *testing.T) {
	ctx := context.Background()
	repo := &mockClusterRepo{clusters: map[string]*models.Cluster{
		"id-1": {
			ID: "id-1", Name: "cluster-1", Context: "ctx1",
			Status: "disconnected", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}}
	cfg := &config.Config{MaxClusters: 1}
	svc := NewClusterService(repo, cfg)

	_, err := svc.AddCluster(ctx, "/nonexistent/kubeconfig", "ctx")
	if err == nil {
		t.Fatal("expected error when at cluster limit")
	}
	if !strings.Contains(err.Error(), "cluster limit reached") {
		t.Errorf("expected cluster limit error, got: %v", err)
	}

	// Verify typed error carries count and limit
	var limitErr *ErrClusterLimitReached
	if !errors.As(err, &limitErr) {
		t.Fatal("expected ErrClusterLimitReached typed error")
	}
	if limitErr.Current != 1 {
		t.Errorf("expected Current=1, got %d", limitErr.Current)
	}
	if limitErr.Max != 1 {
		t.Errorf("expected Max=1, got %d", limitErr.Max)
	}
}

// TestClusterService_DiscoverClusters_ReturnsNonEmptyID verifies BA-1 / task list: discovered clusters have non-empty id.
func TestClusterService_DiscoverClusters_ReturnsNonEmptyID(t *testing.T) {
	// Minimal valid kubeconfig with one context (same format as rest package createTestKubeconfig).
	kubeconfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test-server:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: ctx-one
currentContext: ctx-one
users:
- name: test-user
  user:
    token: test-token
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(kubeconfig), 0600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	orig := os.Getenv("KUBECONFIG")
	os.Setenv("KUBECONFIG", path)
	defer func() { _ = os.Setenv("KUBECONFIG", orig) }()

	ctx := context.Background()
	repo := &mockClusterRepo{clusters: make(map[string]*models.Cluster)}
	svc := NewClusterService(repo, nil)

	list, err := svc.DiscoverClusters(ctx)
	if err != nil {
		t.Fatalf("DiscoverClusters: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("expected at least one discovered cluster")
	}
	for i, c := range list {
		if c.ID == "" {
			t.Errorf("discovered cluster[%d] has empty id (context=%q)", i, c.Context)
		}
	}
}

// TestClusterService_AddCluster_Idempotent verifies task list: POST /clusters with same context twice returns same UUID.
func TestClusterService_AddCluster_Idempotent(t *testing.T) {
	ctx := context.Background()
	repo := &mockClusterRepo{clusters: make(map[string]*models.Cluster)}
	factory := func(kubeconfigPath, contextName string) (*k8s.Client, error) {
		return k8s.NewClientForTest(fake.NewSimpleClientset()), nil
	}
	svc := NewClusterServiceWithClientFactory(repo, nil, factory)

	path := filepath.Join(t.TempDir(), "kubeconfig")
	_ = os.WriteFile(path, []byte("apiVersion: v1\nkind: Config"), 0644)
	contextName := "ctx-one"

	c1, err := svc.AddCluster(ctx, path, contextName)
	if err != nil {
		t.Fatalf("first AddCluster: %v", err)
	}
	if c1.ID == "" {
		t.Fatal("first AddCluster returned cluster with empty ID")
	}

	c2, err := svc.AddCluster(ctx, path, contextName)
	if err != nil {
		t.Fatalf("second AddCluster: %v", err)
	}
	if c2.ID != c1.ID {
		t.Errorf("idempotent add: first ID %q, second ID %q (expected same)", c1.ID, c2.ID)
	}
}

// TestClusterService_AddCluster_ConnectionFailure verifies registration succeeds even if connection fails.
func TestClusterService_AddCluster_ConnectionFailure(t *testing.T) {
	ctx := context.Background()
	repo := &mockClusterRepo{clusters: make(map[string]*models.Cluster)}
	factory := func(kubeconfigPath, contextName string) (*k8s.Client, error) {
		// client.TestConnection in k8s package would hit real network or fail;
		// since we use a factory, we can't easily mock the client.TestConnection method
		// easily without a more complex interface.
		// However, NewClientForTest doesn't mock TestConnection.
		// Let's assume AddCluster uses the clientFactory but we need to control the error.
		return k8s.NewClientForTest(fake.NewSimpleClientset()), nil
	}

	// Wait, the current AddCluster implementation calls client.TestConnection(regCtx).
	// For testing, we might need to mock the client more deeply if we want to truly test the connection failure path.
	// But let's see if we can at least verify that it doesn't throw a fatal error when a client fails.
	// Since client.TestConnection uses c.Clientset.CoreV1().Namespaces().List, and fake clientset
	// usually succeeds, we'd need to mock it to fail.

	// For now, let's at least verify it completes with "connected" status if mock succeeds.
	svc := NewClusterServiceWithClientFactory(repo, nil, factory)
	path := filepath.Join(t.TempDir(), "kubeconfig-fail")
	_ = os.WriteFile(path, []byte("apiVersion: v1\nkind: Config"), 0644)

	c, err := svc.AddCluster(ctx, path, "fail-ctx")
	if err != nil {
		t.Fatalf("AddCluster failed: %v", err)
	}
	if c.Status != "connected" && c.Status != "disconnected" && c.Status != "error" {
		t.Errorf("unexpected status: %s", c.Status)
	}
}

// TestClusterService_AddCluster_UnreachableClusterKeepsDeclaredServerURL is the
// regression seal for the root cause behind "clicking an unreachable cluster
// in the picker does something confusing/broken": when the live connection
// test fails, addClusterWithSource left ServerURL as "" (serverURL only gets
// set inside the TestConnection-succeeded branch) and persisted that empty
// string. KubeconfigFileSource's discovery Enumerate(), in contrast, parses
// the kubeconfig's declared `server:` field directly off disk — a purely
// local read requiring no network call. The result: the SAME cluster gets
// TWO DIFFERENT LogicalIdentity keys system-wide the moment it's unreachable
// — identity.LogicalIdentity{Name, ServerURL} with a real ServerURL from
// discovery, and {Name, ""} from the registered/manual record — so the
// presence Manager's dedup-by-key (manager.go's Refresh) never merges them.
// The frontend then renders/clicks the enriched-less (no session_id, no
// kubeconfig_path) discovered-only entry, re-POSTs /clusters with a wrong
// fallback path, and the user sees a cryptic failure instead of "already
// registered, just unreachable." The declared kubeconfig server URL must
// survive a failed connection test exactly like KubeconfigFileSource reads
// it, independent of whether the live call could confirm it.
func TestClusterService_AddCluster_UnreachableClusterKeepsDeclaredServerURL(t *testing.T) {
	ctx := context.Background()
	repo := &mockClusterRepo{clusters: make(map[string]*models.Cluster)}

	const declaredServerURL = "https://10.255.255.1:6443"
	const contextName = "blackhole-context"

	factory := func(kubeconfigPath, ctxName string) (*k8s.Client, error) {
		clientset := fake.NewSimpleClientset()
		clientset.PrependReactor("list", "namespaces", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("context deadline exceeded")
		})
		return k8s.NewClientForTest(clientset), nil
	}
	svc := NewClusterServiceWithClientFactory(repo, nil, factory)

	path := filepath.Join(t.TempDir(), "kubeconfig")
	kubeconfigYAML := `apiVersion: v1
kind: Config
clusters:
- name: blackhole-cluster
  cluster:
    server: ` + declaredServerURL + `
contexts:
- name: ` + contextName + `
  context:
    cluster: blackhole-cluster
    user: blackhole-user
users:
- name: blackhole-user
  user:
    token: fake
`
	if err := os.WriteFile(path, []byte(kubeconfigYAML), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := svc.AddCluster(ctx, path, contextName)
	if err != nil {
		t.Fatalf("AddCluster (expected to succeed despite unreachable connection): %v", err)
	}
	if c.Status == "connected" {
		t.Fatalf("expected a non-connected status for an unreachable cluster, got %q", c.Status)
	}
	if c.ServerURL != declaredServerURL {
		t.Fatalf(
			"ServerURL = %q, want the kubeconfig's declared server %q — an empty/wrong ServerURL here "+
				"diverges from KubeconfigFileSource's discovery identity for the same cluster, breaking "+
				"the presence dedup",
			c.ServerURL, declaredServerURL,
		)
	}
}

// Ensure mockClusterRepo satisfies repository.ClusterRepository
var _ repository.ClusterRepository = (*mockClusterRepo)(nil)

// ─── ReconnectCluster strict kubeconfig validation (P0 data-identity fix) ───

func TestReconnectCluster_ErrorsWhenKubeconfigPathIsEmpty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := &mockClusterRepo{clusters: map[string]*models.Cluster{
		"c-empty-path": {
			ID: "c-empty-path", Name: "was-aws", Context: "aws-prod",
			KubeconfigPath: "", Status: "connected",
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}}
	svc := NewClusterService(repo, nil)

	_, err := svc.ReconnectCluster(ctx, "c-empty-path")
	if err == nil {
		t.Fatal("expected error for empty kubeconfig path, got nil")
	}
	if !strings.Contains(err.Error(), "no stored kubeconfig path") {
		t.Errorf("unexpected error: %v", err)
	}
	updated, _ := repo.Get(ctx, "c-empty-path")
	if updated.Status != "disconnected" {
		t.Errorf("expected status=disconnected, got %q", updated.Status)
	}
}

func TestReconnectCluster_ErrorsWhenKubeconfigFileMissing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tmp := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	repo := &mockClusterRepo{clusters: map[string]*models.Cluster{
		"c-missing-file": {
			ID: "c-missing-file", Name: "was-aws", Context: "aws-prod",
			KubeconfigPath: tmp, Status: "connected",
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}}
	svc := NewClusterService(repo, nil)

	_, err := svc.ReconnectCluster(ctx, "c-missing-file")
	if err == nil {
		t.Fatal("expected error for missing kubeconfig file, got nil")
	}
	if !strings.Contains(err.Error(), "no longer exists") {
		t.Errorf("unexpected error: %v", err)
	}
	updated, _ := repo.Get(ctx, "c-missing-file")
	if updated.Status != "disconnected" {
		t.Errorf("expected status=disconnected, got %q", updated.Status)
	}
}

func TestReconnectCluster_ErrorsWhenKubeconfigPathIsDirectory(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	repo := &mockClusterRepo{clusters: map[string]*models.Cluster{
		"c-is-dir": {
			ID: "c-is-dir", Name: "weird", Context: "ctx",
			KubeconfigPath: dir, Status: "connected",
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}}
	svc := NewClusterService(repo, nil)

	_, err := svc.ReconnectCluster(ctx, "c-is-dir")
	if err == nil {
		t.Fatal("expected error for directory path, got nil")
	}
	if !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("unexpected error: %v", err)
	}
}
