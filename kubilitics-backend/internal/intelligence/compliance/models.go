package compliance

import "time"

// ComplianceResult is the top-level response for a compliance evaluation
// against a single framework (e.g. CIS Kubernetes Benchmark v1.8, SOC2).
type ComplianceResult struct {
	ClusterID   string          `json:"cluster_id"`
	Framework   string          `json:"framework"`     // "cis-1.8", "soc2", "hipaa"
	PassCount   int             `json:"pass_count"`
	FailCount   int             `json:"fail_count"`
	WarnCount   int             `json:"warn_count"`
	TotalCount  int             `json:"total_count"`
	Score       float64         `json:"score"`          // pass_count / total_count * 100
	Controls    []ControlResult `json:"controls"`
	GeneratedAt time.Time       `json:"generated_at"`
}

// ControlResult is the evaluation outcome for a single compliance control.
type ControlResult struct {
	ControlID         string        `json:"control_id"`   // e.g. "CIS-5.2.1"
	Title             string        `json:"title"`
	Description       string        `json:"description"`
	Status            string        `json:"status"`       // "pass", "fail", "warn"
	Severity          string        `json:"severity"`     // "critical", "high", "medium", "low"
	AffectedResources []ResourceRef `json:"affected_resources"`
	Remediation       string        `json:"remediation"`
	Framework         string        `json:"framework"`
}

// ResourceRef identifies a Kubernetes resource in a compliance context.
type ResourceRef struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
}
