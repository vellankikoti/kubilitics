package runtime

import (
	"context"
	"testing"

	"github.com/vellankikoti/kubilitics/brain/internal/llm/types"
)

// fakeToolExecutor records the args it was actually called with, so these
// tests can assert on what clusterIDInjectingExecutor passes downstream —
// not just that Execute returns without error.
type fakeToolExecutor struct {
	lastArgs map[string]interface{}
}

func (f *fakeToolExecutor) Execute(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	f.lastArgs = args
	return "ok", nil
}

func (f *fakeToolExecutor) WithAutonomyLevel(level int) types.ToolExecutor {
	return f
}

// TestClusterIDInjectingExecutor_InjectsWhenMissing is the Area 5
// (AI/MCP tool correctness) regression test this mechanism never had:
// clusterIDInjectingExecutor is documented as "the authoritative defense"
// against an LLM that omits cluster_id and the inner handler's
// first-registered-cluster fallback silently routing to the wrong
// cluster — but the injection itself had zero direct test coverage.
func TestClusterIDInjectingExecutor_InjectsWhenMissing(t *testing.T) {
	inner := &fakeToolExecutor{}
	e := &clusterIDInjectingExecutor{inner: inner, clusterID: "cluster-focus"}

	if _, err := e.Execute(context.Background(), "list_pods", map[string]interface{}{"namespace": "default"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got, ok := inner.lastArgs["cluster_id"].(string)
	if !ok || got != "cluster-focus" {
		t.Fatalf("cluster_id = %v, want %q — the focus cluster must be injected when the LLM omits it", inner.lastArgs["cluster_id"], "cluster-focus")
	}
}

// TestClusterIDInjectingExecutor_InjectsWhenEmptyString covers the other
// half of the doc comment's "missing (or passes an empty string)" claim.
func TestClusterIDInjectingExecutor_InjectsWhenEmptyString(t *testing.T) {
	inner := &fakeToolExecutor{}
	e := &clusterIDInjectingExecutor{inner: inner, clusterID: "cluster-focus"}

	if _, err := e.Execute(context.Background(), "list_pods", map[string]interface{}{"cluster_id": ""}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if got := inner.lastArgs["cluster_id"]; got != "cluster-focus" {
		t.Fatalf("cluster_id = %v, want %q — an explicit empty string must be treated the same as missing", got, "cluster-focus")
	}
}

// TestClusterIDInjectingExecutor_PreservesExplicitClusterID is the
// equally important other direction: if the LLM (or a deliberate
// multi-cluster tool call) already specified a real cluster_id, the
// wrapper must NOT silently overwrite it with the session's focus
// cluster — that would make cross-cluster tool calls impossible even
// when correctly specified.
func TestClusterIDInjectingExecutor_PreservesExplicitClusterID(t *testing.T) {
	inner := &fakeToolExecutor{}
	e := &clusterIDInjectingExecutor{inner: inner, clusterID: "cluster-focus"}

	if _, err := e.Execute(context.Background(), "list_pods", map[string]interface{}{"cluster_id": "cluster-other"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if got := inner.lastArgs["cluster_id"]; got != "cluster-other" {
		t.Fatalf("cluster_id = %v, want %q — an explicitly-specified cluster_id must survive untouched", got, "cluster-other")
	}
}

// TestClusterIDInjectingExecutor_NilArgsDoesNotPanic — the inner handler
// doc comment says "if args == nil, args = {}" before injection; confirm
// that path is real and doesn't panic on a nil map write.
func TestClusterIDInjectingExecutor_NilArgsDoesNotPanic(t *testing.T) {
	inner := &fakeToolExecutor{}
	e := &clusterIDInjectingExecutor{inner: inner, clusterID: "cluster-focus"}

	if _, err := e.Execute(context.Background(), "list_pods", nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if got := inner.lastArgs["cluster_id"]; got != "cluster-focus" {
		t.Fatalf("cluster_id = %v, want %q even when called with nil args", got, "cluster-focus")
	}
}
