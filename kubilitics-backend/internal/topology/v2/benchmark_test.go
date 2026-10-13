package v2_test

import (
	"context"
	"sort"
	"testing"

	v2 "github.com/kubilitics/kubilitics-backend/internal/topology/v2"
	"github.com/kubilitics/kubilitics-backend/internal/topology/v2/builder"
	"github.com/kubilitics/kubilitics-backend/internal/topology/v2/relationships"
)

// BenchmarkTopologyBuild_100Resources benchmarks topology build for ~100 resources.
// Target: < 500ms
func BenchmarkTopologyBuild_100Resources(b *testing.B) {
	opts := v2.FixtureOptions{
		Namespaces: 2, Deployments: 10, PodsPerDeploy: 3,
		Services: 10, ConfigMaps: 5, Secrets: 5, Nodes: 3,
	}
	bundle := v2.NewLargeFixture(opts)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry := relationships.NewDefaultRegistry()
		edges, _, _ := registry.MatchAll(ctx, bundle)
		nodes := builder.NodesFromBundle(bundle)
		_ = &v2.TopologyResponse{
			Nodes: nodes,
			Edges: edges,
		}
	}
}

// BenchmarkTopologyBuild_500Resources benchmarks topology build for ~500 resources.
// Target: < 1.5s
func BenchmarkTopologyBuild_500Resources(b *testing.B) {
	opts := v2.FixtureOptions{
		Namespaces: 5, Deployments: 50, PodsPerDeploy: 3,
		Services: 50, ConfigMaps: 20, Secrets: 20, Nodes: 10,
	}
	bundle := v2.NewLargeFixture(opts)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry := relationships.NewDefaultRegistry()
		edges, _, _ := registry.MatchAll(ctx, bundle)
		nodes := builder.NodesFromBundle(bundle)
		_ = &v2.TopologyResponse{
			Nodes: nodes,
			Edges: edges,
		}
	}
}

// BenchmarkTopologyBuild_1000Resources benchmarks topology build for ~1000 resources.
// Target: < 3s
func BenchmarkTopologyBuild_1000Resources(b *testing.B) {
	opts := v2.FixtureOptions{
		Namespaces: 5, Deployments: 100, PodsPerDeploy: 3,
		Services: 100, ConfigMaps: 30, Secrets: 30, Nodes: 10,
	}
	bundle := v2.NewLargeFixture(opts)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry := relationships.NewDefaultRegistry()
		edges, _, _ := registry.MatchAll(ctx, bundle)
		nodes := builder.NodesFromBundle(bundle)
		_ = &v2.TopologyResponse{
			Nodes: nodes,
			Edges: edges,
		}
	}
}

// BenchmarkTopologyBuild_2000Resources benchmarks topology build for ~2000 resources.
// Target: < 5s
func BenchmarkTopologyBuild_2000Resources(b *testing.B) {
	opts := v2.FixtureOptions{
		Namespaces: 10, Deployments: 200, PodsPerDeploy: 3,
		Services: 200, ConfigMaps: 50, Secrets: 50, Nodes: 20,
	}
	bundle := v2.NewLargeFixture(opts)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry := relationships.NewDefaultRegistry()
		edges, _, _ := registry.MatchAll(ctx, bundle)
		nodes := builder.NodesFromBundle(bundle)
		_ = &v2.TopologyResponse{
			Nodes: nodes,
			Edges: edges,
		}
	}
}

// BenchmarkMatchAll_StandardFixture benchmarks MatchAll on the standard test fixture.
// Target: < 100ms
func BenchmarkMatchAll_StandardFixture(b *testing.B) {
	bundle := v2.NewTestFixtureBundle()
	ctx := context.Background()
	registry := relationships.NewDefaultRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = registry.MatchAll(ctx, bundle)
	}
}

// BenchmarkViewFilter_ResourceCentric_Depth3 benchmarks resource-centric BFS filtering.
// Target: < 50ms
func BenchmarkViewFilter_ResourceCentric_Depth3(b *testing.B) {
	bundle := v2.NewTestFixtureBundle()
	ctx := context.Background()
	registry := relationships.NewDefaultRegistry()
	edges, _, _ := registry.MatchAll(ctx, bundle)
	nodes := builder.NodesFromBundle(bundle)

	response := &v2.TopologyResponse{
		Nodes: nodes,
		Edges: edges,
	}

	filter := &v2.ViewFilter{}
	opts := v2.Options{
		Mode:     v2.ViewModeResource,
		Resource: "Pod/production/payment-api-0",
		Depth:    3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filter.Filter(response, opts)
	}
}

// BenchmarkViewFilter_FullMode_2000Resources benchmarks FULL-mode (depth>=3,
// unbounded-traversal-by-design per filterResource) BFS filtering against a
// genuinely large graph. This is the missing data point flagged by
// docs/PRODUCTION-RELIABILITY-AUDIT.md's Phase 5 "large-cluster topology
// performance... explicitly left UNVERIFIED" — every prior benchmark either
// measured the BUILD step (MatchAll) at scale, or measured FULL-mode FILTER
// but only against the small NewTestFixtureBundle. This measures the actual
// unbounded-traversal cost at scale: 2000 resources, same shape as
// BenchmarkTopologyBuild_2000Resources, with Depth=3 (which per filter.go's
// own `isFullModeDepth` logic makes this an unbounded-by-design traversal,
// not a 3-hop-capped one).
// Target: stay well under the 500ms LOADING-1-class "infinite spinner" risk
// threshold referenced elsewhere in this audit — this benchmark exists to
// catch a regression toward that, not to assert a specific number.
func BenchmarkViewFilter_FullMode_2000Resources(b *testing.B) {
	opts := v2.FixtureOptions{
		Namespaces: 10, Deployments: 200, PodsPerDeploy: 3,
		Services: 200, ConfigMaps: 50, Secrets: 50, Nodes: 20,
	}
	bundle := v2.NewLargeFixture(opts)
	ctx := context.Background()
	registry := relationships.NewDefaultRegistry()
	edges, _, _ := registry.MatchAll(ctx, bundle)
	nodes := builder.NodesFromBundle(bundle)

	response := &v2.TopologyResponse{
		Nodes: nodes,
		Edges: edges,
	}

	filter := &v2.ViewFilter{}
	fullOpts := v2.Options{
		Mode:     v2.ViewModeResource,
		Resource: "Deployment/ns-0/deploy-0",
		Depth:    3, // unbounded Full mode by design, see filter.go
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := filter.Filter(response, fullOpts)
		if result == nil || len(result.Nodes) == 0 {
			b.Fatalf("expected a non-empty result for Deployment/ns-0/deploy-0 at Depth=3 against the 2000-resource fixture — filter returned nothing, which would make this benchmark falsely report as fast")
		}
	}
}

// TestDeterminism verifies that two identical builds produce identical output.
func TestDeterminism(t *testing.T) {
	bundle := v2.NewTestFixtureBundle()
	ctx := context.Background()
	registry := relationships.NewDefaultRegistry()

	edges1, _, _ := registry.MatchAll(ctx, bundle)
	edges2, _, _ := registry.MatchAll(ctx, bundle)

	if len(edges1) != len(edges2) {
		t.Fatalf("edge count mismatch: %d vs %d", len(edges1), len(edges2))
	}

	// Sort both slices by ID for deterministic comparison
	sort.Slice(edges1, func(i, j int) bool { return edges1[i].ID < edges1[j].ID })
	sort.Slice(edges2, func(i, j int) bool { return edges2[i].ID < edges2[j].ID })

	for i := range edges1 {
		if edges1[i].ID != edges2[i].ID {
			t.Fatalf("edge %d: ID mismatch %q vs %q", i, edges1[i].ID, edges2[i].ID)
		}
	}
}
