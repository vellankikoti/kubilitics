package models

// BlastRadiusResult contains the full blast radius analysis for a single resource.
type BlastRadiusResult struct {
	TargetResource     ResourceRef      `json:"target_resource"`
	CriticalityScore   float64          `json:"criticality_score"`   // 0-100
	CriticalityLevel   string           `json:"criticality_level"`   // low / medium / high / critical
	BlastRadiusPercent float64          `json:"blast_radius_percent"` // percentage of namespace services affected
	AffectedResources  []ResourceRef    `json:"affected_resources"`
	FanIn              int              `json:"fan_in"`  // how many resources depend on this
	FanOut             int              `json:"fan_out"` // how many resources this depends on
	IsSPOF             bool             `json:"is_spof"` // single point of failure
	DependencyChain    []BlastDependencyEdge `json:"dependency_chain"`
}

// ResourceRef identifies a Kubernetes resource by kind, name, and namespace.
type ResourceRef struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// BlastDependencyEdge represents a directed dependency between two resources
// discovered by the blast radius engine.
type BlastDependencyEdge struct {
	Source ResourceRef `json:"source"`
	Target ResourceRef `json:"target"`
	Type   string      `json:"type"` // env_var, volume_mount, selector, ingress_route, network_policy
}
