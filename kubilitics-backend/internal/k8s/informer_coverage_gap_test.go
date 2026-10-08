package k8s

import (
	"testing"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

// Regression tests for the Oct 2026 informer-cache coverage gap
// (docs/ai/KNOWN-ISSUES.md): resourceKindToStoreKey previously tracked 27
// kinds; these 8 were added without a cache entry, so they hit the live K8s
// API on every read instead of the sub-ms cache path. Each case proves
// ListFromCache now resolves the kind to a populated store instead of
// falling back to (nil, false).
func TestListFromCache_CoverageGapKinds_HitCache(t *testing.T) {
	newStore := func() cache.Indexer {
		return cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	}

	cases := []struct {
		resourceType string
		storeKey     string
		obj          interface{}
	}{
		{"resourcequotas", "ResourceQuota", &corev1.ResourceQuota{ObjectMeta: metav1.ObjectMeta{Name: "rq-1", Namespace: "default"}}},
		{"limitranges", "LimitRange", &corev1.LimitRange{ObjectMeta: metav1.ObjectMeta{Name: "lr-1", Namespace: "default"}}},
		{"endpointslices", "EndpointSlice", &discoveryv1.EndpointSlice{ObjectMeta: metav1.ObjectMeta{Name: "es-1", Namespace: "default"}}},
		{"leases", "Lease", &coordinationv1.Lease{ObjectMeta: metav1.ObjectMeta{Name: "lease-1", Namespace: "default"}}},
		{"volumeattachments", "VolumeAttachment", &storagev1.VolumeAttachment{ObjectMeta: metav1.ObjectMeta{Name: "va-1"}}},
		{"mutatingwebhookconfigurations", "MutatingWebhookConfiguration", &admissionregistrationv1.MutatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Name: "mwc-1"}}},
		{"validatingwebhookconfigurations", "ValidatingWebhookConfiguration", &admissionregistrationv1.ValidatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Name: "vwc-1"}}},
		{"customresourcedefinitions", "CustomResourceDefinition", &apiextensionsv1.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{Name: "crd-1"}}},
	}

	for _, tc := range cases {
		t.Run(tc.resourceType, func(t *testing.T) {
			store := newStore()
			if err := store.Add(tc.obj); err != nil {
				t.Fatalf("seed store: %v", err)
			}
			im := &InformerManager{stores: map[string]cache.Indexer{tc.storeKey: store}}
			im.synced.Store(true)

			result, ok := im.ListFromCache(tc.resourceType, "", metav1.ListOptions{})
			if !ok {
				t.Fatalf("expected cache hit for %q (store key %q) — resourceKindToStoreKey entry missing or wrong", tc.resourceType, tc.storeKey)
			}
			if len(result.Items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(result.Items))
			}
		})
	}
}

// NewInformerManager must not panic when client.Config is nil (true for
// NewClientForTest and any other caller that only sets Clientset) —
// apiextensionsclientset.NewForConfig panics rather than erroring on a nil
// *rest.Config, so the nil check in NewInformerManager must come first.
func TestNewInformerManager_NilConfig_DoesNotPanic(t *testing.T) {
	client := NewClientForTest(fake.NewSimpleClientset())
	if client.Config != nil {
		t.Fatal("test setup assumption violated: NewClientForTest now sets Config")
	}

	im := NewInformerManager(client)
	defer im.Stop()

	if im.crdFactory != nil {
		t.Error("expected crdFactory to stay nil when client.Config is nil")
	}
}
