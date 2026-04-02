package graph

import (
	"fmt"

	"github.com/kubilitics/kubilitics-backend/internal/models"
)

// ComputeRemediations returns a prioritized list of recommended actions to reduce
// the blast radius of a resource, based on its current configuration.
func ComputeRemediations(isSPOF bool, hasPDB bool, hasHPA bool, replicas int, fanIn int, crossNsCount int, isDataStore bool) []models.Remediation {
	var remediations []models.Remediation

	// Critical: data store with insufficient replicas
	if isDataStore && replicas < 3 {
		remediations = append(remediations, models.Remediation{
			Type:        "increase-replicas-datastore",
			Description: fmt.Sprintf("Data store has only %d replica(s) — increase to at least 3 for quorum-based redundancy", replicas),
			Priority:    "critical",
			Impact:      "Reduces blast radius score by ~25 points (eliminates SPOF + replica penalties)",
		})
	}

	// Critical: SPOF with high fan-in
	if isSPOF && fanIn > 5 {
		remediations = append(remediations, models.Remediation{
			Type:        "resolve-critical-spof",
			Description: fmt.Sprintf("Critical single point of failure — %d workloads depend on this resource with only 1 replica", fanIn),
			Priority:    "critical",
			Impact:      "Reduces blast radius score by ~15 points (eliminates SPOF penalty and reduces pod-crash impact to near-zero)",
		})
	}

	// High: single replica (non-datastore, or low fan-in SPOF)
	if replicas == 1 && !(isDataStore && replicas < 3) && !(isSPOF && fanIn > 5) {
		remediations = append(remediations, models.Remediation{
			Type:        "increase-replicas",
			Description: "Increase replicas to at least 3 — a single pod crash currently takes out 100% of capacity",
			Priority:    "high",
			Impact:      "Reduces pod-crash blast radius score by ~66% (from 1/1 to 1/3 replica factor)",
		})
	}

	// High: no PDB
	if !hasPDB && replicas > 0 {
		remediations = append(remediations, models.Remediation{
			Type:        "add-pdb",
			Description: "Add a PodDisruptionBudget to protect against voluntary disruptions (node drains, cluster upgrades)",
			Priority:    "high",
			Impact:      "Reduces blast radius score by ~5 points",
		})
	}

	// Medium: no HPA
	if !hasHPA && replicas > 0 {
		remediations = append(remediations, models.Remediation{
			Type:        "add-hpa",
			Description: "Add a HorizontalPodAutoscaler to enable automatic scaling under load",
			Priority:    "medium",
			Impact:      "Reduces blast radius score by ~5 points and improves resilience to traffic spikes",
		})
	}

	// Medium: cross-namespace dependency hub
	if crossNsCount > 2 {
		crossNsPoints := float64(crossNsCount) * 2.5
		if crossNsPoints > 10.0 {
			crossNsPoints = 10.0
		}
		remediations = append(remediations, models.Remediation{
			Type:        "reduce-cross-ns-coupling",
			Description: fmt.Sprintf("Cross-namespace dependency hub — %d namespaces depend on this resource; consider service mesh or API gateway isolation", crossNsCount),
			Priority:    "medium",
			Impact:      fmt.Sprintf("Reduces blast radius score by ~%.0f points (cross-namespace contribution)", crossNsPoints),
		})
	}

	return remediations
}
