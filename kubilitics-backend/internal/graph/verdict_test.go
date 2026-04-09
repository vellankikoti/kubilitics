package graph

import (
	"strings"
	"testing"

	"github.com/kubilitics/kubilitics-backend/internal/models"
)

func TestGenerateVerdict_ZeroImpact(t *testing.T) {
	result := &models.BlastRadiusResult{
		TargetResource:   models.ResourceRef{Kind: "Pod", Name: "my-pod"},
		CriticalityLevel: "low",
		CriticalityScore: 15,
		ReplicaCount:     3,
		HasHPA:           true,
		HasPDB:           true,
		ImpactSummary: models.ImpactSummary{
			BrokenCount: 0, DegradedCount: 0, SelfHealingCount: 1,
		},
		CoverageLevel: "high",
	}
	verdict := generateVerdict(result)
	if !strings.Contains(verdict, "LOW") {
		t.Errorf("expected LOW in verdict, got: %s", verdict)
	}
	if !strings.Contains(verdict, "no services lose functionality") {
		t.Errorf("expected 'no services lose functionality', got: %s", verdict)
	}
}

func TestGenerateVerdict_BrokenServices(t *testing.T) {
	result := &models.BlastRadiusResult{
		TargetResource:   models.ResourceRef{Kind: "Deployment", Name: "api"},
		CriticalityLevel: "critical",
		CriticalityScore: 85,
		ReplicaCount:     1,
		HasHPA:           false,
		HasPDB:           false,
		IsIngressExposed: true,
		ImpactSummary: models.ImpactSummary{
			BrokenCount: 2, DegradedCount: 1,
		},
		CoverageLevel: "high",
	}
	verdict := generateVerdict(result)
	if !strings.Contains(verdict, "CRITICAL") {
		t.Errorf("expected CRITICAL, got: %s", verdict)
	}
	if !strings.Contains(verdict, "2 service(s) would become unreachable") {
		t.Errorf("expected broken count, got: %s", verdict)
	}
	if !strings.Contains(verdict, "internet-facing") {
		t.Errorf("expected internet-facing, got: %s", verdict)
	}
}

func TestGenerateVerdict_PartialCoverage(t *testing.T) {
	result := &models.BlastRadiusResult{
		TargetResource:   models.ResourceRef{Kind: "Service", Name: "svc"},
		CriticalityLevel: "medium",
		CriticalityScore: 30,
		CoverageLevel:    "partial",
	}
	verdict := generateVerdict(result)
	if !strings.Contains(verdict, "tracing") {
		t.Errorf("expected tracing note, got: %s", verdict)
	}
}
