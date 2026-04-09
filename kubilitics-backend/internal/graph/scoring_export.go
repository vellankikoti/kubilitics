package graph

import (
	"math"

	"github.com/kubilitics/kubilitics-backend/internal/models"
)

// ScoringParams is kept for backward compatibility with the simulation package.
// Full composite scoring now happens at query time in scoring_v2.go.
type ScoringParams struct {
	PageRank         float64
	FanIn            int
	CrossNsCount     int
	IsDataStore      bool
	IsIngressExposed bool
	IsSPOF           bool
	HasHPA           bool
	HasPDB           bool
}

// ComputeCriticalityScore returns a lightweight structural importance score
// (PageRank + fan-in) used by the simulation package when rescoring a mutated
// snapshot. Full composite scoring happens at query time via scoring_v2.go.
func ComputeCriticalityScore(p ScoringParams) float64 {
	return math.Min(p.PageRank*30.0, 30.0) + math.Min(float64(p.FanIn)*3.0, 20.0)
}

// SimplePageRank exports the internal simplePageRank function
// so the simulation package can recompute PageRank on a mutated graph.
func SimplePageRank(
	nodes map[string]models.ResourceRef,
	forward, reverse map[string]map[string]bool,
) map[string]float64 {
	return simplePageRank(nodes, forward, reverse)
}

// BfsWalk exports the internal bfsWalk function for cross-namespace counting.
func BfsWalk(adj map[string]map[string]bool, startKey string) map[string]bool {
	return bfsWalk(adj, startKey)
}

// RefKey exports the internal refKey helper.
func RefKey(r models.ResourceRef) string {
	return refKey(r)
}
