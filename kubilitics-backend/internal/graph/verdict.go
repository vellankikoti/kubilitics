package graph

import (
	"fmt"
	"strings"

	"github.com/kubilitics/kubilitics-backend/internal/models"
)

// generateVerdict produces a deterministic natural language explanation of the blast radius result.
func generateVerdict(result *models.BlastRadiusResult) string {
	var parts []string

	parts = append(parts, fmt.Sprintf(
		"This %s has %s criticality (score: %.0f).",
		result.TargetResource.Kind,
		strings.ToUpper(result.CriticalityLevel),
		result.CriticalityScore,
	))

	protections := []string{}
	if result.HasHPA {
		protections = append(protections, "HPA")
	}
	if result.HasPDB {
		protections = append(protections, "PDB")
	}

	if result.ReplicaCount > 0 {
		if len(protections) > 0 {
			parts = append(parts, fmt.Sprintf(
				"It has %d replica(s) with %s.",
				result.ReplicaCount, strings.Join(protections, " and "),
			))
		} else {
			parts = append(parts, fmt.Sprintf(
				"It has %d replica(s), no HPA, no PDB.",
				result.ReplicaCount,
			))
		}
	}

	if result.IsIngressExposed {
		parts = append(parts, "It is internet-facing via Ingress.")
	}

	broken := result.ImpactSummary.BrokenCount
	degraded := result.ImpactSummary.DegradedCount

	if broken == 0 && degraded == 0 {
		parts = append(parts, "Under this failure mode, no services lose functionality.")
	} else {
		if broken > 0 {
			parts = append(parts, fmt.Sprintf("%d service(s) would become unreachable.", broken))
		}
		if degraded > 0 {
			parts = append(parts, fmt.Sprintf("%d service(s) would operate at reduced capacity.", degraded))
		}
	}

	if result.CoverageLevel == "partial" {
		parts = append(parts, "Note: Consumer dependencies not available — enable distributed tracing for full analysis.")
	}

	return strings.Join(parts, " ")
}
