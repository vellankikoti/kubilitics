package builder

import (
	"sort"
	"testing"

	v2 "github.com/kubilitics/kubilitics-backend/internal/topology/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Batch 2 / Theme 2 #12: groupsFromBundle previously re-scanned every pod/
// deployment/service in the WHOLE bundle for each namespace instead of
// building a namespace index once. No test exercised this function
// directly or indirectly (no test asserted on BuildGraph's Groups field)
// before this change.

func groupMembers(t *testing.T, groups []v2.TopologyGroup, groupID string) []string {
	t.Helper()
	for _, g := range groups {
		if g.ID == groupID {
			return g.Members
		}
	}
	t.Fatalf("no group with ID %q found", groupID)
	return nil
}

func TestGroupsFromBundle_MembersScopedToOwnNamespace(t *testing.T) {
	bundle := &v2.ResourceBundle{
		Namespaces: []corev1.Namespace{
			{ObjectMeta: metav1.ObjectMeta{Name: "ns-a"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "ns-b"}},
		},
		Pods: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "pod-a", Namespace: "ns-a"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "pod-b", Namespace: "ns-b"}},
		},
		Deployments: []appsv1.Deployment{
			{ObjectMeta: metav1.ObjectMeta{Name: "dep-a", Namespace: "ns-a"}},
		},
		Services: []corev1.Service{
			{ObjectMeta: metav1.ObjectMeta{Name: "svc-b", Namespace: "ns-b"}},
		},
	}

	groups := groupsFromBundle(bundle)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	membersA := groupMembers(t, groups, "group-ns-ns-a")
	wantA := []string{
		v2.NodeID("Pod", "ns-a", "pod-a"),
		v2.NodeID("Deployment", "ns-a", "dep-a"),
	}
	sort.Strings(membersA)
	sort.Strings(wantA)
	if len(membersA) != len(wantA) {
		t.Fatalf("ns-a: expected members %v, got %v", wantA, membersA)
	}
	for i := range wantA {
		if membersA[i] != wantA[i] {
			t.Errorf("ns-a: expected members %v, got %v", wantA, membersA)
			break
		}
	}

	membersB := groupMembers(t, groups, "group-ns-ns-b")
	wantB := []string{
		v2.NodeID("Pod", "ns-b", "pod-b"),
		v2.NodeID("Service", "ns-b", "svc-b"),
	}
	sort.Strings(membersB)
	sort.Strings(wantB)
	if len(membersB) != len(wantB) {
		t.Fatalf("ns-b: expected members %v, got %v", wantB, membersB)
	}
	for i := range wantB {
		if membersB[i] != wantB[i] {
			t.Errorf("ns-b: expected members %v, got %v", wantB, membersB)
			break
		}
	}
}

func TestGroupsFromBundle_EmptyNamespace_NoMembers(t *testing.T) {
	bundle := &v2.ResourceBundle{
		Namespaces: []corev1.Namespace{
			{ObjectMeta: metav1.ObjectMeta{Name: "empty-ns"}},
		},
		Pods: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "pod-a", Namespace: "other-ns"}},
		},
	}

	groups := groupsFromBundle(bundle)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if members := groupMembers(t, groups, "group-ns-empty-ns"); len(members) != 0 {
		t.Errorf("expected no members for empty-ns, got %v", members)
	}
}

func TestGroupsFromBundle_NoNamespaces_NoGroups(t *testing.T) {
	bundle := &v2.ResourceBundle{}
	groups := groupsFromBundle(bundle)
	if len(groups) != 0 {
		t.Fatalf("expected no groups for an empty bundle, got %d", len(groups))
	}
}
