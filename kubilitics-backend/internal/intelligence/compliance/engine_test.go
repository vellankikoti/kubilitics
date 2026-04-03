package compliance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_ListFrameworks(t *testing.T) {
	e := NewEngine()
	frameworks := e.ListFrameworks()
	assert.Contains(t, frameworks, "cis-1.8")
	assert.Contains(t, frameworks, "soc2")
}

func TestEngine_UnknownFramework_ReturnsError(t *testing.T) {
	e := NewEngine()
	result, err := e.Evaluate("unknown-framework", "cluster-1", ClusterComplianceData{})
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown compliance framework")
}

func TestEngine_Evaluate_CIS_Score100(t *testing.T) {
	e := NewEngine()
	result, err := e.Evaluate("cis-1.8", "cluster-1", allPassingData())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "cluster-1", result.ClusterID)
	assert.Equal(t, "cis-1.8", result.Framework)
	assert.Equal(t, float64(100), result.Score)
	assert.Equal(t, 0, result.FailCount)
	assert.Equal(t, 0, result.WarnCount)
	assert.Greater(t, result.TotalCount, 0)
	assert.Equal(t, result.PassCount, result.TotalCount)
	assert.False(t, result.GeneratedAt.IsZero())
}

func TestEngine_Evaluate_CIS_PartialFailures(t *testing.T) {
	data := allPassingData()
	data.Workloads[0].HasPDB = false  // CIS-5.2.1 fails
	data.Workloads[0].IsSPOF = true   // CIS-5.7.3 fails
	data.Workloads[0].Privileged = true // CIS-5.2.5 fails

	e := NewEngine()
	result, err := e.Evaluate("cis-1.8", "cluster-1", data)
	require.NoError(t, err)

	assert.Equal(t, 3, result.FailCount)
	assert.Less(t, result.Score, float64(100))
	assert.Greater(t, result.Score, float64(0))
}

func TestEngine_Evaluate_SOC2(t *testing.T) {
	e := NewEngine()
	result, err := e.Evaluate("soc2", "cluster-1", allPassingData())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "soc2", result.Framework)
	assert.Greater(t, result.TotalCount, 0)
	// SOC2-CC6.1 always warns (RBAC), SOC2-CC8.1 always warns (CI/CD)
	assert.GreaterOrEqual(t, result.WarnCount, 2, "SOC2 should have at least 2 warnings (RBAC + change mgmt)")
}

func TestEngine_Evaluate_SOC2_MapsCorrectly(t *testing.T) {
	// When CIS-5.7.1 fails (low replicas), SOC2-CC7.2 should also fail
	data := allPassingData()
	data.Workloads[0].Replicas = 1

	e := NewEngine()
	result, err := e.Evaluate("soc2", "cluster-1", data)
	require.NoError(t, err)

	var cc72Found bool
	for _, c := range result.Controls {
		if c.ControlID == "SOC2-CC7.2" {
			cc72Found = true
			assert.Equal(t, "fail", c.Status, "SOC2-CC7.2 should fail when CIS-5.7.1 fails")
			require.NotEmpty(t, c.AffectedResources)
		}
	}
	require.True(t, cc72Found, "SOC2-CC7.2 should be in results")
}

func TestEngine_Evaluate_SOC2_PDBMapping(t *testing.T) {
	// When CIS-5.2.1 fails (no PDB), SOC2-A1.2 should also fail
	data := allPassingData()
	data.Workloads[0].HasPDB = false

	e := NewEngine()
	result, err := e.Evaluate("soc2", "cluster-1", data)
	require.NoError(t, err)

	var a12Found bool
	for _, c := range result.Controls {
		if c.ControlID == "SOC2-A1.2" {
			a12Found = true
			assert.Equal(t, "fail", c.Status, "SOC2-A1.2 should fail when CIS-5.2.1 fails")
		}
	}
	require.True(t, a12Found, "SOC2-A1.2 should be in results")
}

func TestEngine_Evaluate_EmptyCluster(t *testing.T) {
	e := NewEngine()
	result, err := e.Evaluate("cis-1.8", "empty-cluster", ClusterComplianceData{
		NetworkPolicies: map[string]bool{},
		ResourceQuotas:  map[string]bool{},
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// With no workloads and no namespaces, all controls should pass
	assert.Equal(t, float64(100), result.Score)
	assert.Equal(t, 0, result.FailCount)
}

func TestEngine_Register_CustomFramework(t *testing.T) {
	e := NewEngine()
	custom := &mockFramework{name: "custom-1.0"}
	e.Register(custom)

	assert.Contains(t, e.ListFrameworks(), "custom-1.0")
}

// mockFramework is a test double for the Framework interface.
type mockFramework struct {
	name string
}

func (m *mockFramework) Name() string { return m.name }
func (m *mockFramework) Evaluate(_ ClusterComplianceData) []ControlResult {
	return []ControlResult{
		{ControlID: "MOCK-1", Status: "pass", Framework: m.name},
	}
}

func TestEngine_ScoreCalculation(t *testing.T) {
	// 1 pass, 1 fail, 1 warn => score = 1/3 * 100 = 33.33...
	e := NewEngine()
	e.Register(&mockFramework{name: "test-calc"})

	// Override with a framework that returns mixed results
	e.frameworks["test-calc"] = &mixedFramework{}

	result, err := e.Evaluate("test-calc", "c1", ClusterComplianceData{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.PassCount)
	assert.Equal(t, 1, result.FailCount)
	assert.Equal(t, 1, result.WarnCount)
	assert.Equal(t, 3, result.TotalCount)
	assert.InDelta(t, 33.33, result.Score, 0.01)
}

type mixedFramework struct{}

func (m *mixedFramework) Name() string { return "test-calc" }
func (m *mixedFramework) Evaluate(_ ClusterComplianceData) []ControlResult {
	return []ControlResult{
		{ControlID: "T-1", Status: "pass"},
		{ControlID: "T-2", Status: "fail"},
		{ControlID: "T-3", Status: "warn"},
	}
}
