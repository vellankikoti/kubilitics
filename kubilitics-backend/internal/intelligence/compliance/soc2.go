package compliance

// SOC2Framework maps structural findings to SOC2 Type II trust service criteria.
// Since SOC2 is an audit framework (not a prescriptive benchmark), many controls
// map to underlying CIS checks rather than defining novel detection logic.
type SOC2Framework struct{}

// Name returns the framework identifier.
func (f *SOC2Framework) Name() string { return "soc2" }

// Evaluate runs all SOC2 controls by delegating to the underlying CIS checks
// where applicable and returning SOC2-scoped ControlResults.
func (f *SOC2Framework) Evaluate(data ClusterComplianceData) []ControlResult {
	cis := &CISFramework{}
	cisResults := cis.Evaluate(data)

	// Index CIS results by control ID for easy lookup.
	cisMap := make(map[string]ControlResult, len(cisResults))
	for _, cr := range cisResults {
		cisMap[cr.ControlID] = cr
	}

	var results []ControlResult
	results = append(results, f.checkCC6_1(cisMap))
	results = append(results, f.checkCC7_2(cisMap))
	results = append(results, f.checkCC8_1())
	results = append(results, f.checkA1_2(cisMap))
	return results
}

// SOC2-CC6.1 — Logical access controls (maps to CIS-5.1.3 RBAC check).
// Since we do not yet have RBAC audit data, we derive the status from the
// presence of permissive cluster role bindings. For now this always warns.
func (f *SOC2Framework) checkCC6_1(cisMap map[string]ControlResult) ControlResult {
	cr := ControlResult{
		ControlID:   "SOC2-CC6.1",
		Title:       "Logical access controls",
		Description: "The entity uses logical access security measures to restrict access to information assets.",
		Severity:    "high",
		Framework:   "soc2",
		Remediation: "Review ClusterRoleBindings and remove overly permissive bindings. Implement least-privilege RBAC.",
	}

	// We don't have RBAC data yet, so always warn.
	cr.Status = "warn"
	cr.Description = "RBAC audit not yet integrated. Manual review of ClusterRoleBindings recommended."

	return cr
}

// SOC2-CC7.2 — System availability (maps to CIS-5.7.1 replica count).
func (f *SOC2Framework) checkCC7_2(cisMap map[string]ControlResult) ControlResult {
	cr := ControlResult{
		ControlID:   "SOC2-CC7.2",
		Title:       "System availability",
		Description: "The entity monitors system availability and takes action to maintain processing capacity.",
		Severity:    "high",
		Framework:   "soc2",
		Remediation: "Ensure all production workloads run at least 2 replicas. Configure HPA for automatic scaling.",
	}

	if cis, ok := cisMap["CIS-5.7.1"]; ok {
		cr.Status = cis.Status
		cr.AffectedResources = cis.AffectedResources
		if cis.Status == "fail" {
			cr.Description = "Workloads with insufficient replicas detected (maps to CIS-5.7.1)."
		}
	} else {
		cr.Status = "pass"
	}

	return cr
}

// SOC2-CC8.1 — Change management.
// This control requires external CI/CD integration data that is not available
// from the cluster graph, so it always returns a warning.
func (f *SOC2Framework) checkCC8_1() ControlResult {
	return ControlResult{
		ControlID:   "SOC2-CC8.1",
		Title:       "Change management",
		Description: "The entity authorizes, designs, develops, configures, documents, tests, approves, and implements changes to infrastructure and software.",
		Status:      "warn",
		Severity:    "medium",
		Framework:   "soc2",
		Remediation: "Integrate CI/CD pipeline audit data to verify that all deployments follow a change management process.",
	}
}

// SOC2-A1.2 — Recovery mechanisms (maps to CIS-5.2.1 PDB coverage).
func (f *SOC2Framework) checkA1_2(cisMap map[string]ControlResult) ControlResult {
	cr := ControlResult{
		ControlID:   "SOC2-A1.2",
		Title:       "Recovery mechanisms",
		Description: "The entity provides recovery mechanisms to meet its objectives for availability.",
		Severity:    "high",
		Framework:   "soc2",
		Remediation: "Create PodDisruptionBudgets for all multi-replica workloads to ensure graceful disruption handling.",
	}

	if cis, ok := cisMap["CIS-5.2.1"]; ok {
		cr.Status = cis.Status
		cr.AffectedResources = cis.AffectedResources
		if cis.Status == "fail" {
			cr.Description = "Workloads without PodDisruptionBudgets detected (maps to CIS-5.2.1)."
		}
	} else {
		cr.Status = "pass"
	}

	return cr
}
