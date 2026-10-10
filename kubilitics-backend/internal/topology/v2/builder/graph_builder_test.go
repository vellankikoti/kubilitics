package builder

import (
	"context"
	"testing"

	"github.com/kubilitics/kubilitics-backend/internal/topology/v2"
)

func TestBuildGraph_WithFixture(t *testing.T) {
	bundle := v2.NewTestFixtureBundle()
	opts := v2.Options{ClusterID: "test", ClusterName: "test-cluster", Mode: v2.ViewModeNamespace}
	resp, err := BuildGraph(context.Background(), opts, bundle)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Nodes) < 20 {
		t.Errorf("expected at least 20 nodes, got %d", len(resp.Nodes))
	}
	if len(resp.Edges) < 30 {
		t.Errorf("expected at least 30 edges, got %d", len(resp.Edges))
	}
	if resp.Metadata.ClusterID != "test" || resp.Metadata.ClusterName != "test-cluster" {
		t.Errorf("metadata: got clusterId=%q clusterName=%q", resp.Metadata.ClusterID, resp.Metadata.ClusterName)
	}
}

// TestBuildGraph_AggregatesPodsOnBroadViews is the regression seal for wiring
// AggregatePods (pod_aggregation.go) into BuildGraph. It existed, was fully
// tested in isolation, and had zero call sites — every pod in the cluster was
// built and returned as its own node regardless of cluster size, which is the
// root cause behind Topology freezing/slowing down as pod count grows (see
// docs/ai/KNOWN-ISSUES.md). A broad view (Cluster/Namespace/Workload) must
// collapse pods under a shared owner into one PodGroup node once the group
// exceeds podGroupThreshold.
func TestBuildGraph_AggregatesPodsOnBroadViews(t *testing.T) {
	bundle := v2.NewLargeFixture(v2.FixtureOptions{
		Namespaces: 1, Deployments: 1, PodsPerDeploy: 10, Nodes: 1, Services: 1, ConfigMaps: 1, Secrets: 1,
	})
	opts := v2.Options{ClusterID: "test", Mode: v2.ViewModeNamespace, Namespace: "ns-0"}
	resp, err := BuildGraph(context.Background(), opts, bundle)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	podCount, groupCount := 0, 0
	for _, n := range resp.Nodes {
		switch n.Kind {
		case "Pod":
			podCount++
		case "PodGroup":
			groupCount++
		}
	}
	if podCount != 0 {
		t.Errorf("expected individual Pod nodes to be collapsed on a Namespace view, found %d", podCount)
	}
	if groupCount != 1 {
		t.Errorf("expected exactly 1 PodGroup node for the 10-pod deployment, got %d", groupCount)
	}
}

// TestBuildGraph_DoesNotAggregatePodsOnResourceFocus verifies a focused
// single-resource drill-down (ViewModeResource) keeps individual pods — a
// user inspecting one Deployment wants to see which specific pod is
// unhealthy, not a collapsed summary.
func TestBuildGraph_DoesNotAggregatePodsOnResourceFocus(t *testing.T) {
	bundle := v2.NewLargeFixture(v2.FixtureOptions{
		Namespaces: 1, Deployments: 1, PodsPerDeploy: 10, Nodes: 1, Services: 1, ConfigMaps: 1, Secrets: 1,
	})
	opts := v2.Options{ClusterID: "test", Mode: v2.ViewModeResource, Namespace: "ns-0", Resource: "Deployment/ns-0/deploy-0"}
	resp, err := BuildGraph(context.Background(), opts, bundle)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	podCount := 0
	for _, n := range resp.Nodes {
		if n.Kind == "Pod" {
			podCount++
		}
	}
	if podCount != 10 {
		t.Errorf("expected all 10 individual Pod nodes on a resource-focused view, got %d", podCount)
	}
}
