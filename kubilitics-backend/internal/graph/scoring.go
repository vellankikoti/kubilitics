package graph

import (
	"math"

	"github.com/kubilitics/kubilitics-backend/internal/models"
)

// FailureMode describes what type of failure is being simulated.
const (
	FailureModePodCrash           = "pod-crash"
	FailureModeWorkloadDeletion   = "workload-deletion"
	FailureModeNamespaceDeletion  = "namespace-deletion"
)

// ValidFailureMode returns true if mode is a recognized failure mode string.
func ValidFailureMode(mode string) bool {
	switch mode {
	case FailureModePodCrash, FailureModeWorkloadDeletion, FailureModeNamespaceDeletion:
		return true
	default:
		return false
	}
}

// scoringParams holds the inputs required to compute a criticality score.
type scoringParams struct {
	pageRank         float64
	fanIn            int
	crossNsCount     int
	isDataStore      bool
	isIngressExposed bool
	isSPOF           bool
	hasHPA           bool
	hasPDB           bool
}

// computeBaseScore returns the raw 0-100 criticality score from the given params,
// before any failure-mode replica adjustment is applied.
func computeBaseScore(p scoringParams) float64 {
	score := 0.0

	// PageRank contribution: max 30
	score += math.Min(p.pageRank*30.0, 30.0)

	// Fan-in contribution: max 20
	score += math.Min(float64(p.fanIn)*3.0, 20.0)

	// Cross-namespace contribution: max 10, only if >1
	if p.crossNsCount > 1 {
		score += math.Min(float64(p.crossNsCount)*2.5, 10.0)
	}

	// Data store bonus
	if p.isDataStore {
		score += 15.0
	}

	// Ingress exposed bonus
	if p.isIngressExposed {
		score += 10.0
	}

	// SPOF bonus
	if p.isSPOF {
		score += 10.0
	}

	// No HPA penalty
	if !p.hasHPA {
		score += 5.0
	}

	// No PDB penalty
	if !p.hasPDB {
		score += 5.0
	}

	// Cap at 100
	if score > 100.0 {
		score = 100.0
	}
	return score
}

// computeCriticalityScore returns a 0-100 criticality score from the given params.
// This is the backward-compatible entry point that computes the base score
// (equivalent to workload-deletion mode with no replica attenuation).
func computeCriticalityScore(p scoringParams) float64 {
	return computeBaseScore(p)
}

// applyFailureMode adjusts the base score according to the failure mode and replica count.
//   - pod-crash:           score * (1 / replicas) — near-zero if replicas > 1
//   - workload-deletion:   score * 1.0 — full impact (default)
//   - namespace-deletion:  handled externally (sum of workload scores, capped at 100)
func applyFailureMode(baseScore float64, failureMode string, replicas int) float64 {
	switch failureMode {
	case FailureModePodCrash:
		if replicas > 1 {
			return baseScore * (1.0 / float64(replicas))
		}
		return baseScore // replicas <= 1 means full impact
	case FailureModeWorkloadDeletion, FailureModeNamespaceDeletion:
		return baseScore
	default:
		return baseScore
	}
}

// simplePageRank computes an iterative PageRank over the graph and returns
// a map of nodeKey -> normalized score in the [0, 1] range.
//
// Parameters:
//
//	nodes   – refKey -> ResourceRef mapping (keys are the node identifiers)
//	forward – adjacency map: source -> set of targets (what source depends on)
//	reverse – adjacency map: target -> set of sources (what depends on target)
func simplePageRank(
	nodes map[string]models.ResourceRef,
	forward map[string]map[string]bool,
	_ map[string]map[string]bool, // reverse — accepted for API symmetry; PageRank only needs forward
) map[string]float64 {
	nodeList := make([]string, 0, len(nodes))
	for k := range nodes {
		nodeList = append(nodeList, k)
	}
	return pageRankOnKeys(nodeList, forward)
}

// pageRankOnKeys is the core PageRank implementation operating on a plain
// slice of node keys.
func pageRankOnKeys(
	nodeList []string,
	forward map[string]map[string]bool,
) map[string]float64 {
	const damping = 0.85
	const maxIter = 50
	const convergenceThreshold = 0.0001

	n := len(nodeList)
	if n == 0 {
		return map[string]float64{}
	}

	rank := make(map[string]float64, n)
	initial := 1.0 / float64(n)
	for _, k := range nodeList {
		rank[k] = initial
	}

	for iter := 0; iter < maxIter; iter++ {
		newRank := make(map[string]float64, n)
		for _, k := range nodeList {
			newRank[k] = (1.0 - damping) / float64(n)
		}

		for _, k := range nodeList {
			// outDegree = number of outgoing edges (forward links)
			out := len(forward[k])
			if out == 0 {
				// Dangling node: distribute rank evenly to all nodes
				share := rank[k] / float64(n)
				for _, dest := range nodeList {
					newRank[dest] += damping * share
				}
			} else {
				share := rank[k] / float64(out)
				for dest := range forward[k] {
					newRank[dest] += damping * share
				}
			}
		}

		// Check convergence
		delta := 0.0
		for _, k := range nodeList {
			d := newRank[k] - rank[k]
			if d < 0 {
				d = -d
			}
			delta += d
		}
		rank = newRank
		if delta < convergenceThreshold {
			break
		}
	}

	// Normalize to [0, 1]
	maxVal := 0.0
	for _, v := range rank {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal > 0 {
		for k, v := range rank {
			rank[k] = v / maxVal
		}
	}

	return rank
}

