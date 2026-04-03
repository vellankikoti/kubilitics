package compliance

import (
	"fmt"
	"time"
)

// Framework is the interface every compliance framework must implement.
type Framework interface {
	Name() string
	Evaluate(data ClusterComplianceData) []ControlResult
}

// ClusterComplianceData holds the cluster-level data required to evaluate
// compliance controls. It is assembled from the graph snapshot and Kubernetes
// resource state before being passed to each framework.
type ClusterComplianceData struct {
	Workloads         []WorkloadInfo
	CriticalityScores map[string]ScoreInfo
	NetworkPolicies   map[string]bool // namespace -> has network policy
	ResourceQuotas    map[string]bool // namespace -> has resource quota
}

// WorkloadInfo captures the properties of a workload controller that are
// relevant for compliance evaluation.
type WorkloadInfo struct {
	Name        string
	Kind        string
	Namespace   string
	Replicas    int
	HasPDB      bool
	HasHPA      bool
	HasLimits   bool // resource limits set
	HasRequests bool // resource requests set
	IsSPOF      bool
	Privileged  bool
}

// ScoreInfo holds a precomputed criticality score for a resource.
type ScoreInfo struct {
	Score float64
	Level string // "low", "medium", "high", "critical"
}

// Engine is the central compliance evaluation engine. It holds a registry
// of named frameworks and delegates evaluation to them.
type Engine struct {
	frameworks map[string]Framework
}

// NewEngine creates an Engine with all built-in frameworks registered.
func NewEngine() *Engine {
	e := &Engine{
		frameworks: make(map[string]Framework),
	}
	e.Register(&CISFramework{})
	e.Register(&SOC2Framework{})
	return e
}

// Register adds a framework to the engine.
func (e *Engine) Register(f Framework) {
	e.frameworks[f.Name()] = f
}

// Evaluate runs all controls for the named framework and returns an
// aggregated ComplianceResult. Returns an error if the framework is unknown.
func (e *Engine) Evaluate(framework, clusterID string, data ClusterComplianceData) (*ComplianceResult, error) {
	f, ok := e.frameworks[framework]
	if !ok {
		return nil, fmt.Errorf("unknown compliance framework: %q", framework)
	}

	controls := f.Evaluate(data)

	var pass, fail, warn int
	for _, c := range controls {
		switch c.Status {
		case "pass":
			pass++
		case "fail":
			fail++
		case "warn":
			warn++
		}
	}

	total := pass + fail + warn
	score := 0.0
	if total > 0 {
		score = float64(pass) / float64(total) * 100
	}

	return &ComplianceResult{
		ClusterID:   clusterID,
		Framework:   framework,
		PassCount:   pass,
		FailCount:   fail,
		WarnCount:   warn,
		TotalCount:  total,
		Score:       score,
		Controls:    controls,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

// ListFrameworks returns the names of all registered frameworks.
func (e *Engine) ListFrameworks() []string {
	names := make([]string, 0, len(e.frameworks))
	for n := range e.frameworks {
		names = append(names, n)
	}
	return names
}
