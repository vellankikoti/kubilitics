package graph

import (
	"testing"

	"github.com/kubilitics/kubilitics-backend/internal/models"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestComputeLostPods_PodCrash(t *testing.T) {
	snap := &GraphSnapshot{}
	snap.EnsureMaps()
	snap.Nodes["Pod/default/pod-1"] = models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "pod-1"}

	lost := computeLostPods(snap, models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "pod-1"}, FailureModePodCrash)
	if len(lost) != 1 || !lost["Pod/default/pod-1"] {
		t.Errorf("expected exactly pod-1 lost, got %v", lost)
	}
}

func TestComputeLostPods_WorkloadDeletion(t *testing.T) {
	snap := &GraphSnapshot{}
	snap.EnsureMaps()
	snap.Nodes["Deployment/default/app"] = models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "app"}
	snap.Nodes["Pod/default/app-abc-1"] = models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "app-abc-1"}
	snap.Nodes["Pod/default/app-abc-2"] = models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "app-abc-2"}
	snap.Nodes["Pod/default/other-pod"] = models.ResourceRef{Kind: "Pod", Namespace: "default", Name: "other-pod"}
	snap.PodOwners = map[string]string{
		"Pod/default/app-abc-1": "Deployment/default/app",
		"Pod/default/app-abc-2": "Deployment/default/app",
		"Pod/default/other-pod": "Deployment/default/other",
	}

	lost := computeLostPods(snap, models.ResourceRef{Kind: "Deployment", Namespace: "default", Name: "app"}, FailureModeWorkloadDeletion)
	if len(lost) != 2 {
		t.Errorf("expected 2 lost pods, got %d: %v", len(lost), lost)
	}
}

func TestClassifyServiceImpact_Broken(t *testing.T) {
	endpoints := map[string][]corev1.EndpointAddress{
		"Service/default/svc-a": {
			{IP: "10.0.0.1", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-1", Namespace: "default"}},
		},
	}
	lostPods := map[string]bool{"Pod/default/pod-1": true}
	impacts := classifyServiceImpact(endpoints, lostPods, nil)
	if len(impacts) != 1 {
		t.Fatalf("expected 1 impact, got %d", len(impacts))
	}
	if impacts[0].Classification != "broken" {
		t.Errorf("expected broken, got %s", impacts[0].Classification)
	}
}

func TestClassifyServiceImpact_SelfHealing(t *testing.T) {
	endpoints := map[string][]corev1.EndpointAddress{
		"Service/default/svc-a": {
			{IP: "10.0.0.1", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-1", Namespace: "default"}},
			{IP: "10.0.0.2", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-2", Namespace: "default"}},
			{IP: "10.0.0.3", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-3", Namespace: "default"}},
			{IP: "10.0.0.4", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-4", Namespace: "default"}},
		},
	}
	lostPods := map[string]bool{"Pod/default/pod-1": true}
	impacts := classifyServiceImpact(endpoints, lostPods, nil)
	if len(impacts) != 1 {
		t.Fatalf("expected 1 impact, got %d", len(impacts))
	}
	if impacts[0].Classification != "self-healing" {
		t.Errorf("expected self-healing, got %s", impacts[0].Classification)
	}
}

func TestClassifyServiceImpact_DegradedWithPDB(t *testing.T) {
	minAvail := intstr.FromInt32(2)
	pdbs := []policyv1.PodDisruptionBudget{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "my-pdb", Namespace: "default"},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MinAvailable: &minAvail,
				Selector:     &metav1.LabelSelector{MatchLabels: map[string]string{"app": "svc-a"}},
			},
		},
	}
	endpoints := map[string][]corev1.EndpointAddress{
		"Service/default/svc-a": {
			{IP: "10.0.0.1", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-1", Namespace: "default"}},
			{IP: "10.0.0.2", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-2", Namespace: "default"}},
			{IP: "10.0.0.3", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-3", Namespace: "default"}},
		},
	}
	lostPods := map[string]bool{"Pod/default/pod-1": true, "Pod/default/pod-2": true}
	impacts := classifyServiceImpact(endpoints, lostPods, pdbs)
	if len(impacts) != 1 {
		t.Fatalf("expected 1 impact, got %d", len(impacts))
	}
	if impacts[0].Classification != "degraded" {
		t.Errorf("expected degraded, got %s", impacts[0].Classification)
	}
	if impacts[0].ThresholdSource != "pdb:my-pdb" {
		t.Errorf("expected pdb source, got %s", impacts[0].ThresholdSource)
	}
}

func TestClassifyServiceImpact_NoImpact(t *testing.T) {
	endpoints := map[string][]corev1.EndpointAddress{
		"Service/default/svc-a": {
			{IP: "10.0.0.1", TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "pod-2", Namespace: "default"}},
		},
	}
	lostPods := map[string]bool{"Pod/default/pod-1": true}
	impacts := classifyServiceImpact(endpoints, lostPods, nil)
	if len(impacts) != 0 {
		t.Errorf("expected 0 impacts, got %d", len(impacts))
	}
}

func TestComputeBlastRadiusPercent(t *testing.T) {
	impacts := []models.ServiceImpact{
		{Classification: "broken", Service: models.ResourceRef{Namespace: "default"}},
		{Classification: "degraded", Service: models.ResourceRef{Namespace: "default"}},
		{Classification: "self-healing", Service: models.ResourceRef{Namespace: "default"}},
	}
	pct := computeBlastRadiusPercent(impacts, nil, nil, 10)
	expected := 15.0
	if pct != expected {
		t.Errorf("expected %.1f%%, got %.1f%%", expected, pct)
	}
}

func TestComputeBlastRadiusPercent_ZeroDenominator(t *testing.T) {
	pct := computeBlastRadiusPercent(nil, nil, nil, 0)
	if pct != 0 {
		t.Errorf("expected 0, got %f", pct)
	}
}
