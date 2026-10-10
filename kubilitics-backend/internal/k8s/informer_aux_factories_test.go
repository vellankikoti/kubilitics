package k8s

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
)

// Regression tests for the Oct 2026 Phase 4 informer additions
// (docs/ai/STABILIZATION-PLAN.md / docs/ai/KNOWN-ISSUES.md): APIService,
// VerticalPodAutoscaler, and ResourceSlice/DeviceClass (DRA) each use their
// own auxInformerFactory, tracked with an independent sync flag via
// auxByKind so one cluster lacking a given API group/version can never
// disable caching for any other kind — the same isolation CRD already had,
// generalized instead of hand-duplicated per kind.

func withDiscoveryResources(clientset *fake.Clientset, lists ...*metav1.APIResourceList) {
	fd := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	fd.Resources = lists
}

func TestDiscoverDRAVersion_PicksMostGAVersionServed(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	withDiscoveryResources(clientset,
		&metav1.APIResourceList{
			GroupVersion: "resource.k8s.io/v1beta1",
			APIResources: []metav1.APIResource{{Name: "resourceslices"}, {Name: "deviceclasses"}},
		},
		&metav1.APIResourceList{
			GroupVersion: "resource.k8s.io/v1",
			APIResources: []metav1.APIResource{{Name: "resourceslices"}, {Name: "deviceclasses"}},
		},
	)
	client := NewClientForTest(clientset)

	version, ok := discoverDRAVersion(client)
	if !ok {
		t.Fatal("expected a DRA version to be found")
	}
	if version != "v1" {
		t.Fatalf("expected v1 (most GA) to be preferred over v1beta1, got %q", version)
	}
}

func TestDiscoverDRAVersion_FallsBackWhenPreferredIncomplete(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	// v1 only serves resourceslices (not deviceclasses) — must not be picked;
	// v1beta2 serves both and should be used instead.
	withDiscoveryResources(clientset,
		&metav1.APIResourceList{
			GroupVersion: "resource.k8s.io/v1",
			APIResources: []metav1.APIResource{{Name: "resourceslices"}},
		},
		&metav1.APIResourceList{
			GroupVersion: "resource.k8s.io/v1beta2",
			APIResources: []metav1.APIResource{{Name: "resourceslices"}, {Name: "deviceclasses"}},
		},
	)
	client := NewClientForTest(clientset)

	version, ok := discoverDRAVersion(client)
	if !ok {
		t.Fatal("expected a DRA version to be found")
	}
	if version != "v1beta2" {
		t.Fatalf("expected v1beta2 (the only version serving both resources) to be picked, got %q", version)
	}
}

func TestDiscoverDRAVersion_NoneServed_ReturnsFalse(t *testing.T) {
	client := NewClientForTest(fake.NewSimpleClientset())

	if _, ok := discoverDRAVersion(client); ok {
		t.Fatal("expected no DRA version to be found when the cluster serves no resource.k8s.io version at all")
	}
}

func TestDiscoverDRAVersion_NilClientOrClientset_ReturnsFalse(t *testing.T) {
	if _, ok := discoverDRAVersion(nil); ok {
		t.Fatal("expected false for a nil client")
	}
	if _, ok := discoverDRAVersion(&Client{}); ok {
		t.Fatal("expected false when client.Clientset is nil")
	}
}

// TestKindSynced_AuxFactories_IndependentOfEachOtherAndMain proves that one
// aux factory (APIService) staying unsynced neither blocks another aux
// factory (VerticalPodAutoscaler) that HAS synced, nor the main cache — the
// same bug class CRD's isolation was built to prevent, now generalized.
func TestKindSynced_AuxFactories_IndependentOfEachOtherAndMain(t *testing.T) {
	im := &InformerManager{auxByKind: make(map[string]*auxInformerFactory)}
	im.synced.Store(true) // main factory synced fine

	unsyncedAux := &auxInformerFactory{name: "APIService", kinds: map[string]bool{"APIService": true}}
	// unsyncedAux.synced left false — simulates this cluster not serving
	// apiregistration.k8s.io (or RBAC-denied), same as a real deployment.
	im.auxByKind["APIService"] = unsyncedAux

	syncedAux := &auxInformerFactory{name: "VerticalPodAutoscaler", kinds: map[string]bool{"VerticalPodAutoscaler": true}}
	syncedAux.synced.Store(true)
	im.auxByKind["VerticalPodAutoscaler"] = syncedAux

	if im.kindSynced("APIService") {
		t.Fatal("expected APIService to report not-synced")
	}
	if !im.kindSynced("VerticalPodAutoscaler") {
		t.Fatal("expected VerticalPodAutoscaler to report synced independently of APIService's state")
	}
	if !im.kindSynced("Pod") {
		t.Fatal("expected an ordinary main-factory kind to report synced via HasSynced(), unaffected by either aux factory")
	}
}

// NewInformerManager must not panic for the APIService/VPA clientsets either
// (same nil-client.Config guard as CRD) — see
// TestNewInformerManager_NilConfig_DoesNotPanic in informer_coverage_gap_test.go
// for the original CRD-only version of this guarantee.
func TestNewInformerManager_NilConfig_APIServiceAndVPAStayNil(t *testing.T) {
	client := NewClientForTest(fake.NewSimpleClientset())
	if client.Config != nil {
		t.Fatal("test setup assumption violated: NewClientForTest now sets Config")
	}

	im := NewInformerManager(client)
	defer im.Stop()

	if im.apiServiceFactory != nil {
		t.Error("expected apiServiceFactory to stay nil when client.Config is nil")
	}
	if im.vpaFactory != nil {
		t.Error("expected vpaFactory to stay nil when client.Config is nil")
	}
	// draFactory doesn't depend on client.Config (resource.k8s.io informers
	// live on the main Clientset), but the fake clientset's discovery has no
	// configured Resources, so it must still come up nil here.
	if im.draFactory != nil {
		t.Error("expected draFactory to stay nil when the cluster serves no resource.k8s.io version")
	}
}
