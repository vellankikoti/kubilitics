package graph

import (
	"testing"

	"github.com/kubilitics/kubilitics-backend/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestComputeRemediations_SingleReplicaNoPDBNoHPA(t *testing.T) {
	remediations := ComputeRemediations(
		true,  // isSPOF
		false, // hasPDB
		false, // hasHPA
		1,     // replicas
		3,     // fanIn
		1,     // crossNsCount
		false, // isDataStore
	)

	// Should recommend: increase-replicas, add-pdb, add-hpa
	types := make(map[string]bool)
	for _, r := range remediations {
		types[r.Type] = true
	}
	assert.True(t, types["increase-replicas"], "should recommend increasing replicas")
	assert.True(t, types["add-pdb"], "should recommend adding PDB")
	assert.True(t, types["add-hpa"], "should recommend adding HPA")
	assert.False(t, types["resolve-critical-spof"], "fanIn=3 should not trigger critical SPOF (threshold is >5)")
}

func TestComputeRemediations_CriticalSPOFHighFanIn(t *testing.T) {
	remediations := ComputeRemediations(
		true,  // isSPOF
		false, // hasPDB
		false, // hasHPA
		1,     // replicas
		8,     // fanIn > 5
		1,     // crossNsCount
		false, // isDataStore
	)

	types := make(map[string]bool)
	for _, r := range remediations {
		types[r.Type] = true
	}
	assert.True(t, types["resolve-critical-spof"], "fanIn=8 should trigger critical SPOF")
	assert.Equal(t, "critical", findRemediation(remediations, "resolve-critical-spof").Priority)
}

func TestComputeRemediations_DataStoreInsufficientReplicas(t *testing.T) {
	remediations := ComputeRemediations(
		true,  // isSPOF
		true,  // hasPDB
		true,  // hasHPA
		1,     // replicas
		2,     // fanIn
		1,     // crossNsCount
		true,  // isDataStore
	)

	types := make(map[string]bool)
	for _, r := range remediations {
		types[r.Type] = true
	}
	assert.True(t, types["increase-replicas-datastore"], "data store with 1 replica should trigger critical remediation")
	assert.Equal(t, "critical", findRemediation(remediations, "increase-replicas-datastore").Priority)
	// Should NOT also recommend generic increase-replicas (data store remediation covers it)
	assert.False(t, types["increase-replicas"], "should not duplicate with generic increase-replicas")
}

func TestComputeRemediations_CrossNamespaceHub(t *testing.T) {
	remediations := ComputeRemediations(
		false, // isSPOF
		true,  // hasPDB
		true,  // hasHPA
		3,     // replicas
		5,     // fanIn
		4,     // crossNsCount > 2
		false, // isDataStore
	)

	types := make(map[string]bool)
	for _, r := range remediations {
		types[r.Type] = true
	}
	assert.True(t, types["reduce-cross-ns-coupling"], "crossNsCount=4 should trigger cross-ns remediation")
	assert.Equal(t, "medium", findRemediation(remediations, "reduce-cross-ns-coupling").Priority)
}

func TestComputeRemediations_FullyResilient(t *testing.T) {
	// Resource with 3 replicas, HPA, PDB, not SPOF, not data store, low cross-ns
	remediations := ComputeRemediations(
		false, // isSPOF
		true,  // hasPDB
		true,  // hasHPA
		3,     // replicas
		2,     // fanIn
		1,     // crossNsCount
		false, // isDataStore
	)

	assert.Empty(t, remediations, "fully resilient resource should have no remediations")
}

func TestComputeRemediations_ZeroReplicas_NoPDBNoHPA(t *testing.T) {
	// Non-workload resource (replicas=0) should not trigger PDB/HPA recommendations
	remediations := ComputeRemediations(
		false, // isSPOF
		false, // hasPDB
		false, // hasHPA
		0,     // replicas
		0,     // fanIn
		0,     // crossNsCount
		false, // isDataStore
	)

	assert.Empty(t, remediations, "non-workload resource with 0 replicas should have no remediations")
}

// findRemediation is a test helper to find a remediation by type.
func findRemediation(remediations []models.Remediation, remType string) *models.Remediation {
	for _, r := range remediations {
		if r.Type == remType {
			return &r
		}
	}
	return nil
}
