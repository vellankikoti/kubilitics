package service

import (
	"context"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubilitics/kubilitics-backend/internal/config"
	"github.com/kubilitics/kubilitics-backend/internal/k8s"
	"github.com/kubilitics/kubilitics-backend/internal/models"
)

// TestGetClusterSummary_ExposesHealthTimestamps locks in that
// GetClusterSummary surfaces the same LastCheckedAt/LastSuccessAt/
// LastError freshness fields Presence already exposes via
// Client.HealthStatus()/LastCheckedAt() — needed so Fleet rows and
// Picker rows can render identical "last seen" copy.
func TestGetClusterSummary_ExposesHealthTimestamps(t *testing.T) {
	ctx := context.Background()
	repo := &mockClusterRepo{clusters: make(map[string]*models.Cluster)}
	var capturedClient *k8s.Client
	factory := func(kubeconfigPath, contextName string) (*k8s.Client, error) {
		capturedClient = k8s.NewClientForTest(fake.NewSimpleClientset())
		return capturedClient, nil
	}
	cfg := &config.Config{K8sTimeoutSec: 30}
	svc := NewClusterServiceWithClientFactory(repo, cfg, factory)

	c, err := svc.AddCluster(ctx, writeFakeKubeconfig(t), "reachability-ctx")
	if err != nil {
		t.Fatalf("AddCluster: %v", err)
	}

	before := time.Now()
	summary, err := svc.GetClusterSummary(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetClusterSummary: %v", err)
	}
	after := time.Now()

	if summary.LastCheckedAt == nil {
		t.Fatal("expected LastCheckedAt to be set after a successful summary fetch")
	}
	if summary.LastCheckedAt.Before(before) || summary.LastCheckedAt.After(after) {
		t.Errorf("LastCheckedAt %v not within [%v, %v]", *summary.LastCheckedAt, before, after)
	}
	if summary.LastSuccessAt == nil {
		t.Fatal("expected LastSuccessAt to be set on a successful check")
	}
	if summary.LastError != "" {
		t.Errorf("expected empty LastError on success, got %q", summary.LastError)
	}
	_ = capturedClient
}
