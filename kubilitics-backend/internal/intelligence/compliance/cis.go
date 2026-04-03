package compliance

import "fmt"

// CISFramework implements the CIS Kubernetes Benchmark v1.8 compliance checks.
type CISFramework struct{}

// Name returns the framework identifier.
func (f *CISFramework) Name() string { return "cis-1.8" }

// Evaluate runs all CIS v1.8 controls against the provided cluster data and
// returns a ControlResult per control.
func (f *CISFramework) Evaluate(data ClusterComplianceData) []ControlResult {
	var results []ControlResult
	results = append(results, f.checkResourceQuotas(data))
	results = append(results, f.checkPDB(data))
	results = append(results, f.checkPrivileged(data))
	results = append(results, f.checkNetworkPolicy(data))
	results = append(results, f.checkResourceLimits(data))
	results = append(results, f.checkResourceRequests(data))
	results = append(results, f.checkReplicaCount(data))
	results = append(results, f.checkHPA(data))
	results = append(results, f.checkSPOF(data))
	return results
}

// CIS-5.1.1 — Ensure resource quotas are configured.
func (f *CISFramework) checkResourceQuotas(data ClusterComplianceData) ControlResult {
	cr := ControlResult{
		ControlID:   "CIS-5.1.1",
		Title:       "Ensure resource quotas are configured",
		Description: "Every namespace should have a ResourceQuota to prevent unbounded resource consumption.",
		Severity:    "medium",
		Framework:   "cis-1.8",
		Remediation: "Create a ResourceQuota in each namespace to enforce CPU, memory, and object count limits.",
	}

	// Collect all namespaces from workloads and from the ResourceQuotas map.
	allNamespaces := collectNamespaces(data)

	var affected []ResourceRef
	for ns := range allNamespaces {
		if !data.ResourceQuotas[ns] {
			affected = append(affected, ResourceRef{Name: ns, Kind: "Namespace", Namespace: ns})
		}
	}

	if len(affected) > 0 {
		cr.Status = "fail"
		cr.AffectedResources = affected
		cr.Description = fmt.Sprintf("%d namespace(s) lack ResourceQuotas.", len(affected))
	} else {
		cr.Status = "pass"
	}
	return cr
}

// CIS-5.2.1 — Ensure PodDisruptionBudgets are configured for multi-replica workloads.
func (f *CISFramework) checkPDB(data ClusterComplianceData) ControlResult {
	cr := ControlResult{
		ControlID:   "CIS-5.2.1",
		Title:       "Ensure PodDisruptionBudgets are configured",
		Description: "Workloads with more than one replica should have a PDB to survive voluntary disruptions.",
		Severity:    "critical",
		Framework:   "cis-1.8",
		Remediation: "Create a PodDisruptionBudget for every Deployment/StatefulSet with replicas > 1.",
	}

	var affected []ResourceRef
	for _, w := range data.Workloads {
		if w.Replicas > 1 && !w.HasPDB {
			affected = append(affected, ResourceRef{Name: w.Name, Kind: w.Kind, Namespace: w.Namespace})
		}
	}

	if len(affected) > 0 {
		cr.Status = "fail"
		cr.AffectedResources = affected
		cr.Description = fmt.Sprintf("%d workload(s) with replicas > 1 have no PodDisruptionBudget.", len(affected))
	} else {
		cr.Status = "pass"
	}
	return cr
}

// CIS-5.2.5 — Minimize privileged containers.
func (f *CISFramework) checkPrivileged(data ClusterComplianceData) ControlResult {
	cr := ControlResult{
		ControlID:   "CIS-5.2.5",
		Title:       "Minimize privileged containers",
		Description: "No pod should run in privileged mode unless absolutely required.",
		Severity:    "critical",
		Framework:   "cis-1.8",
		Remediation: "Set securityContext.privileged to false on all containers, or use a Pod Security Standard that blocks privileged pods.",
	}

	var affected []ResourceRef
	for _, w := range data.Workloads {
		if w.Privileged {
			affected = append(affected, ResourceRef{Name: w.Name, Kind: w.Kind, Namespace: w.Namespace})
		}
	}

	if len(affected) > 0 {
		cr.Status = "fail"
		cr.AffectedResources = affected
		cr.Description = fmt.Sprintf("%d workload(s) run privileged containers.", len(affected))
	} else {
		cr.Status = "pass"
	}
	return cr
}

// CIS-5.3.2 — Ensure NetworkPolicy is configured per namespace.
func (f *CISFramework) checkNetworkPolicy(data ClusterComplianceData) ControlResult {
	cr := ControlResult{
		ControlID:   "CIS-5.3.2",
		Title:       "Ensure NetworkPolicy is configured",
		Description: "Every namespace hosting workloads should have at least one NetworkPolicy.",
		Severity:    "high",
		Framework:   "cis-1.8",
		Remediation: "Create a default-deny NetworkPolicy in each namespace, then add explicit allow rules.",
	}

	allNamespaces := collectNamespaces(data)

	var affected []ResourceRef
	for ns := range allNamespaces {
		if !data.NetworkPolicies[ns] {
			affected = append(affected, ResourceRef{Name: ns, Kind: "Namespace", Namespace: ns})
		}
	}

	if len(affected) > 0 {
		cr.Status = "fail"
		cr.AffectedResources = affected
		cr.Description = fmt.Sprintf("%d namespace(s) have no NetworkPolicy.", len(affected))
	} else {
		cr.Status = "pass"
	}
	return cr
}

// CIS-5.4.1 — Ensure resource limits are set.
func (f *CISFramework) checkResourceLimits(data ClusterComplianceData) ControlResult {
	cr := ControlResult{
		ControlID:   "CIS-5.4.1",
		Title:       "Ensure resource limits are set",
		Description: "All containers should have CPU and memory limits to prevent noisy-neighbour issues.",
		Severity:    "medium",
		Framework:   "cis-1.8",
		Remediation: "Set resources.limits.cpu and resources.limits.memory on every container spec.",
	}

	var affected []ResourceRef
	for _, w := range data.Workloads {
		if !w.HasLimits {
			affected = append(affected, ResourceRef{Name: w.Name, Kind: w.Kind, Namespace: w.Namespace})
		}
	}

	if len(affected) > 0 {
		cr.Status = "fail"
		cr.AffectedResources = affected
		cr.Description = fmt.Sprintf("%d workload(s) are missing resource limits.", len(affected))
	} else {
		cr.Status = "pass"
	}
	return cr
}

// CIS-5.4.2 — Ensure resource requests are set.
func (f *CISFramework) checkResourceRequests(data ClusterComplianceData) ControlResult {
	cr := ControlResult{
		ControlID:   "CIS-5.4.2",
		Title:       "Ensure resource requests are set",
		Description: "All containers should have CPU and memory requests for proper scheduling.",
		Severity:    "low",
		Framework:   "cis-1.8",
		Remediation: "Set resources.requests.cpu and resources.requests.memory on every container spec.",
	}

	var affected []ResourceRef
	for _, w := range data.Workloads {
		if !w.HasRequests {
			affected = append(affected, ResourceRef{Name: w.Name, Kind: w.Kind, Namespace: w.Namespace})
		}
	}

	if len(affected) > 0 {
		cr.Status = "warn"
		cr.AffectedResources = affected
		cr.Description = fmt.Sprintf("%d workload(s) are missing resource requests.", len(affected))
	} else {
		cr.Status = "pass"
	}
	return cr
}

// CIS-5.7.1 — Ensure replica count meets availability requirements.
func (f *CISFramework) checkReplicaCount(data ClusterComplianceData) ControlResult {
	cr := ControlResult{
		ControlID:   "CIS-5.7.1",
		Title:       "Ensure replica count meets availability",
		Description: "Production workloads should run at least 2 replicas for high availability.",
		Severity:    "high",
		Framework:   "cis-1.8",
		Remediation: "Increase spec.replicas to at least 2 for production workloads.",
	}

	var affected []ResourceRef
	for _, w := range data.Workloads {
		// Only controllers that support replicas
		if w.Kind != "Deployment" && w.Kind != "StatefulSet" {
			continue
		}
		if w.Replicas < 2 {
			affected = append(affected, ResourceRef{Name: w.Name, Kind: w.Kind, Namespace: w.Namespace})
		}
	}

	if len(affected) > 0 {
		cr.Status = "fail"
		cr.AffectedResources = affected
		cr.Description = fmt.Sprintf("%d workload(s) have fewer than 2 replicas.", len(affected))
	} else {
		cr.Status = "pass"
	}
	return cr
}

// CIS-5.7.2 — Ensure HPA is configured for scalable workloads.
func (f *CISFramework) checkHPA(data ClusterComplianceData) ControlResult {
	cr := ControlResult{
		ControlID:   "CIS-5.7.2",
		Title:       "Ensure HPA is configured for scalable workloads",
		Description: "Deployments and StatefulSets should have a HorizontalPodAutoscaler for automatic scaling.",
		Severity:    "medium",
		Framework:   "cis-1.8",
		Remediation: "Create a HorizontalPodAutoscaler targeting the workload with appropriate min/max replicas and metrics.",
	}

	var affected []ResourceRef
	for _, w := range data.Workloads {
		if w.Kind != "Deployment" && w.Kind != "StatefulSet" {
			continue
		}
		if !w.HasHPA {
			affected = append(affected, ResourceRef{Name: w.Name, Kind: w.Kind, Namespace: w.Namespace})
		}
	}

	if len(affected) > 0 {
		cr.Status = "warn"
		cr.AffectedResources = affected
		cr.Description = fmt.Sprintf("%d workload(s) have no HorizontalPodAutoscaler.", len(affected))
	} else {
		cr.Status = "pass"
	}
	return cr
}

// CIS-5.7.3 — Ensure no single points of failure.
func (f *CISFramework) checkSPOF(data ClusterComplianceData) ControlResult {
	cr := ControlResult{
		ControlID:   "CIS-5.7.3",
		Title:       "Ensure no single points of failure",
		Description: "No workload should be a single point of failure in the dependency graph.",
		Severity:    "critical",
		Framework:   "cis-1.8",
		Remediation: "Increase replica count, add PodDisruptionBudgets, and eliminate single-path dependencies.",
	}

	var affected []ResourceRef
	for _, w := range data.Workloads {
		if w.IsSPOF {
			affected = append(affected, ResourceRef{Name: w.Name, Kind: w.Kind, Namespace: w.Namespace})
		}
	}

	if len(affected) > 0 {
		cr.Status = "fail"
		cr.AffectedResources = affected
		cr.Description = fmt.Sprintf("%d workload(s) are single points of failure.", len(affected))
	} else {
		cr.Status = "pass"
	}
	return cr
}

// collectNamespaces returns the set of all namespaces referenced by workloads,
// network policies, and resource quotas.
func collectNamespaces(data ClusterComplianceData) map[string]bool {
	ns := make(map[string]bool)
	for _, w := range data.Workloads {
		ns[w.Namespace] = true
	}
	for n := range data.NetworkPolicies {
		ns[n] = true
	}
	for n := range data.ResourceQuotas {
		ns[n] = true
	}
	return ns
}
