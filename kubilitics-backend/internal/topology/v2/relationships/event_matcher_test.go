package relationships

import (
	"context"
	"testing"

	v2 "github.com/kubilitics/kubilitics-backend/internal/topology/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Batch 2 / Theme 2 #11: EventMatcher previously did an
// O(events × resources-of-that-kind) linear scan (hasResource/hasDeployment/
// hasReplicaSet/...) per event, with zero prior test coverage. These tests
// cover the O(1) index replacement (buildResourceExistenceIndex), including
// the Node/Namespace namespace-is-ignored quirk the original code had that
// the index must replicate exactly.

func TestEventMatcher_Name(t *testing.T) {
	m := EventMatcher{}
	if m.Name() != "event" {
		t.Errorf("expected name 'event', got %q", m.Name())
	}
}

func TestEventMatcher_NilBundle(t *testing.T) {
	m := &EventMatcher{}
	edges, err := m.Match(context.Background(), nil)
	if err != nil || edges != nil {
		t.Errorf("expected (nil, nil) for nil bundle, got (%v, %v)", edges, err)
	}
}

func TestEventMatcher_PodEvent_Matches(t *testing.T) {
	bundle := &v2.ResourceBundle{
		Pods: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"}},
		},
		Events: []corev1.Event{
			{
				ObjectMeta:     metav1.ObjectMeta{Name: "ev-1", Namespace: "default"},
				InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "my-pod"},
				Reason:         "Scheduled",
			},
		},
	}

	m := &EventMatcher{}
	edges, err := m.Match(context.Background(), bundle)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].Target != v2.NodeID("Pod", "default", "my-pod") {
		t.Errorf("expected edge target Pod/default/my-pod, got %v", edges[0].Target)
	}
}

func TestEventMatcher_PodEvent_DifferentNamespace_NoMatch(t *testing.T) {
	bundle := &v2.ResourceBundle{
		Pods: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"}},
		},
		Events: []corev1.Event{
			{
				ObjectMeta:     metav1.ObjectMeta{Name: "ev-1", Namespace: "other"},
				InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "other", Name: "my-pod"},
				Reason:         "Scheduled",
			},
		},
	}

	m := &EventMatcher{}
	edges, err := m.Match(context.Background(), bundle)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 0 {
		t.Fatalf("expected no edge for a pod with the same name in a different namespace, got %d", len(edges))
	}
}

func TestEventMatcher_NodeEvent_NamespaceIgnored(t *testing.T) {
	// Nodes are cluster-scoped. The event's InvolvedObject.Namespace is
	// typically empty, but hasResource's original Node case matched by
	// name only regardless of what namespace was passed — the index must
	// replicate that exactly, not silently start requiring ns=="".
	bundle := &v2.ResourceBundle{
		Nodes: []corev1.Node{
			{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}},
		},
		Events: []corev1.Event{
			{
				ObjectMeta:     metav1.ObjectMeta{Name: "ev-1"},
				InvolvedObject: corev1.ObjectReference{Kind: "Node", Namespace: "some-stray-namespace", Name: "node-1"},
				Reason:         "NodeReady",
			},
		},
	}

	m := &EventMatcher{}
	edges, err := m.Match(context.Background(), bundle)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected the Node event to match by name regardless of namespace, got %d edges", len(edges))
	}
}

func TestEventMatcher_UnresolvedInvolvedObject_Skipped(t *testing.T) {
	bundle := &v2.ResourceBundle{
		Events: []corev1.Event{
			{
				ObjectMeta:     metav1.ObjectMeta{Name: "ev-1", Namespace: "default"},
				InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "ghost-pod"},
				Reason:         "Scheduled",
			},
		},
	}

	m := &EventMatcher{}
	edges, err := m.Match(context.Background(), bundle)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 0 {
		t.Fatalf("expected no edge when the involved object doesn't exist in the bundle, got %d", len(edges))
	}
}

func TestEventMatcher_DuplicateEdges_Deduplicated(t *testing.T) {
	bundle := &v2.ResourceBundle{
		Pods: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"}},
		},
		Events: []corev1.Event{
			{
				ObjectMeta:     metav1.ObjectMeta{Name: "ev-1", Namespace: "default"},
				InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "my-pod"},
				Reason:         "Scheduled",
			},
			{
				ObjectMeta:     metav1.ObjectMeta{Name: "ev-1", Namespace: "default"},
				InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "my-pod"},
				Reason:         "Scheduled",
			},
		},
	}

	m := &EventMatcher{}
	edges, err := m.Match(context.Background(), bundle)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected duplicate Event->Pod edges to be deduplicated, got %d", len(edges))
	}
}
