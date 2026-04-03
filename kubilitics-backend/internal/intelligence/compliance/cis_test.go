package compliance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// allPassingData returns a ClusterComplianceData where every CIS control passes.
func allPassingData() ClusterComplianceData {
	return ClusterComplianceData{
		Workloads: []WorkloadInfo{
			{
				Name:        "api-server",
				Kind:        "Deployment",
				Namespace:   "production",
				Replicas:    3,
				HasPDB:      true,
				HasHPA:      true,
				HasLimits:   true,
				HasRequests: true,
				IsSPOF:      false,
				Privileged:  false,
			},
			{
				Name:        "worker",
				Kind:        "Deployment",
				Namespace:   "production",
				Replicas:    2,
				HasPDB:      true,
				HasHPA:      true,
				HasLimits:   true,
				HasRequests: true,
				IsSPOF:      false,
				Privileged:  false,
			},
		},
		NetworkPolicies: map[string]bool{
			"production": true,
		},
		ResourceQuotas: map[string]bool{
			"production": true,
		},
	}
}

func TestCIS_AllPassing(t *testing.T) {
	cis := &CISFramework{}
	results := cis.Evaluate(allPassingData())

	for _, r := range results {
		assert.Equalf(t, "pass", r.Status,
			"control %s (%s) should pass but got %s", r.ControlID, r.Title, r.Status)
		assert.Empty(t, r.AffectedResources, "control %s should have no affected resources", r.ControlID)
	}
}

func TestCIS_NoPDB_Fails_5_2_1(t *testing.T) {
	data := allPassingData()
	data.Workloads[0].HasPDB = false // api-server has 3 replicas but no PDB

	cis := &CISFramework{}
	results := cis.Evaluate(data)

	var found bool
	for _, r := range results {
		if r.ControlID == "CIS-5.2.1" {
			found = true
			assert.Equal(t, "fail", r.Status)
			require.Len(t, r.AffectedResources, 1)
			assert.Equal(t, "api-server", r.AffectedResources[0].Name)
			assert.Equal(t, "critical", r.Severity)
		}
	}
	require.True(t, found, "CIS-5.2.1 control should be in results")
}

func TestCIS_PrivilegedContainer_Fails_5_2_5(t *testing.T) {
	data := allPassingData()
	data.Workloads[1].Privileged = true

	cis := &CISFramework{}
	results := cis.Evaluate(data)

	var found bool
	for _, r := range results {
		if r.ControlID == "CIS-5.2.5" {
			found = true
			assert.Equal(t, "fail", r.Status)
			require.Len(t, r.AffectedResources, 1)
			assert.Equal(t, "worker", r.AffectedResources[0].Name)
			assert.Equal(t, "critical", r.Severity)
		}
	}
	require.True(t, found, "CIS-5.2.5 control should be in results")
}

func TestCIS_NoNetworkPolicy_Fails_5_3_2(t *testing.T) {
	data := allPassingData()
	data.NetworkPolicies = map[string]bool{} // no policies

	cis := &CISFramework{}
	results := cis.Evaluate(data)

	var found bool
	for _, r := range results {
		if r.ControlID == "CIS-5.3.2" {
			found = true
			assert.Equal(t, "fail", r.Status)
			require.NotEmpty(t, r.AffectedResources)
			assert.Equal(t, "high", r.Severity)
		}
	}
	require.True(t, found, "CIS-5.3.2 control should be in results")
}

func TestCIS_SPOF_Fails_5_7_3(t *testing.T) {
	data := allPassingData()
	data.Workloads[0].IsSPOF = true

	cis := &CISFramework{}
	results := cis.Evaluate(data)

	var found bool
	for _, r := range results {
		if r.ControlID == "CIS-5.7.3" {
			found = true
			assert.Equal(t, "fail", r.Status)
			require.Len(t, r.AffectedResources, 1)
			assert.Equal(t, "api-server", r.AffectedResources[0].Name)
			assert.Equal(t, "critical", r.Severity)
		}
	}
	require.True(t, found, "CIS-5.7.3 control should be in results")
}

func TestCIS_LowReplicaCount_Fails_5_7_1(t *testing.T) {
	data := allPassingData()
	data.Workloads[0].Replicas = 1

	cis := &CISFramework{}
	results := cis.Evaluate(data)

	var found bool
	for _, r := range results {
		if r.ControlID == "CIS-5.7.1" {
			found = true
			assert.Equal(t, "fail", r.Status)
			require.Len(t, r.AffectedResources, 1)
			assert.Equal(t, "api-server", r.AffectedResources[0].Name)
			assert.Equal(t, "high", r.Severity)
		}
	}
	require.True(t, found, "CIS-5.7.1 control should be in results")
}

func TestCIS_NoResourceLimits_Fails_5_4_1(t *testing.T) {
	data := allPassingData()
	data.Workloads[0].HasLimits = false

	cis := &CISFramework{}
	results := cis.Evaluate(data)

	var found bool
	for _, r := range results {
		if r.ControlID == "CIS-5.4.1" {
			found = true
			assert.Equal(t, "fail", r.Status)
			require.Len(t, r.AffectedResources, 1)
			assert.Equal(t, "api-server", r.AffectedResources[0].Name)
		}
	}
	require.True(t, found, "CIS-5.4.1 control should be in results")
}

func TestCIS_NoResourceRequests_Warns_5_4_2(t *testing.T) {
	data := allPassingData()
	data.Workloads[0].HasRequests = false

	cis := &CISFramework{}
	results := cis.Evaluate(data)

	var found bool
	for _, r := range results {
		if r.ControlID == "CIS-5.4.2" {
			found = true
			assert.Equal(t, "warn", r.Status, "CIS-5.4.2 should warn, not fail")
		}
	}
	require.True(t, found, "CIS-5.4.2 control should be in results")
}

func TestCIS_NoHPA_Warns_5_7_2(t *testing.T) {
	data := allPassingData()
	data.Workloads[0].HasHPA = false

	cis := &CISFramework{}
	results := cis.Evaluate(data)

	var found bool
	for _, r := range results {
		if r.ControlID == "CIS-5.7.2" {
			found = true
			assert.Equal(t, "warn", r.Status, "CIS-5.7.2 should warn, not fail")
		}
	}
	require.True(t, found, "CIS-5.7.2 control should be in results")
}

func TestCIS_NoResourceQuota_Fails_5_1_1(t *testing.T) {
	data := allPassingData()
	data.ResourceQuotas = map[string]bool{} // no quotas

	cis := &CISFramework{}
	results := cis.Evaluate(data)

	var found bool
	for _, r := range results {
		if r.ControlID == "CIS-5.1.1" {
			found = true
			assert.Equal(t, "fail", r.Status)
			require.NotEmpty(t, r.AffectedResources)
			assert.Equal(t, "medium", r.Severity)
		}
	}
	require.True(t, found, "CIS-5.1.1 control should be in results")
}

func TestCIS_FrameworkName(t *testing.T) {
	cis := &CISFramework{}
	assert.Equal(t, "cis-1.8", cis.Name())
}

func TestCIS_AllControlsHaveFrameworkField(t *testing.T) {
	cis := &CISFramework{}
	results := cis.Evaluate(allPassingData())
	for _, r := range results {
		assert.Equal(t, "cis-1.8", r.Framework, "control %s should have framework field set", r.ControlID)
	}
}

func TestCIS_PDB_SingleReplica_Passes(t *testing.T) {
	// PDB check only applies to workloads with replicas > 1.
	// A single-replica workload without PDB should NOT fail CIS-5.2.1.
	data := ClusterComplianceData{
		Workloads: []WorkloadInfo{
			{
				Name:        "singleton",
				Kind:        "Deployment",
				Namespace:   "default",
				Replicas:    1,
				HasPDB:      false,
				HasHPA:      true,
				HasLimits:   true,
				HasRequests: true,
			},
		},
		NetworkPolicies: map[string]bool{"default": true},
		ResourceQuotas:  map[string]bool{"default": true},
	}

	cis := &CISFramework{}
	results := cis.Evaluate(data)

	for _, r := range results {
		if r.ControlID == "CIS-5.2.1" {
			assert.Equal(t, "pass", r.Status, "CIS-5.2.1 should pass for single-replica workload without PDB")
		}
	}
}
