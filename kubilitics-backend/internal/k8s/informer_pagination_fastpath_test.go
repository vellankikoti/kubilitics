package k8s

import (
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// Regression tests for the 10K-campaign P1 fix: ListFromCacheWithPagination
// previously converted (ToUnstructured) and sorted the ENTIRE cached
// collection on every request regardless of `limit`. These tests prove the
// typed fast path (sortBy = name/namespace/creationTimestamp) preserves
// exact output semantics while no longer paying O(N) conversion cost for a
// small page out of a large collection.

func newTestInformerManagerWithPods(n int) *InformerManager {
	store := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		ns := fmt.Sprintf("ns-%02d", i%5)
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              fmt.Sprintf("pod-%04d", i),
				Namespace:         ns,
				CreationTimestamp: metav1.NewTime(base.Add(time.Duration(i) * time.Minute)),
			},
			Status: corev1.PodStatus{Phase: corev1.PodPending},
		}
		_ = store.Add(pod)
	}
	im := &InformerManager{
		stores: map[string]cache.Indexer{"Pod": store},
	}
	im.synced.Store(true)
	return im
}

func TestListFromCacheWithPagination_FastPath_FirstPageCorrectness(t *testing.T) {
	im := newTestInformerManagerWithPods(250)

	result, ok := im.ListFromCacheWithPagination("pods", "", "", "name", "asc", 0, 10)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if result.Total != 250 {
		t.Fatalf("expected total=250, got %d", result.Total)
	}
	if len(result.Items) != 10 {
		t.Fatalf("expected 10 items, got %d", len(result.Items))
	}
	for i, item := range result.Items {
		want := fmt.Sprintf("pod-%04d", i)
		if got := item.GetName(); got != want {
			t.Errorf("item[%d]: want name=%s got=%s", i, want, got)
		}
	}
}

func TestListFromCacheWithPagination_FastPath_LaterPageCorrectness(t *testing.T) {
	im := newTestInformerManagerWithPods(250)

	result, ok := im.ListFromCacheWithPagination("pods", "", "", "name", "asc", 240, 10)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(result.Items) != 10 {
		t.Fatalf("expected 10 items on last page, got %d", len(result.Items))
	}
	for i, item := range result.Items {
		want := fmt.Sprintf("pod-%04d", 240+i)
		if got := item.GetName(); got != want {
			t.Errorf("item[%d]: want name=%s got=%s", i, want, got)
		}
	}
}

func TestListFromCacheWithPagination_FastPath_EmptyPageBeyondTotal(t *testing.T) {
	im := newTestInformerManagerWithPods(50)

	result, ok := im.ListFromCacheWithPagination("pods", "", "", "name", "asc", 1000, 10)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected 0 items beyond total, got %d", len(result.Items))
	}
	if result.Total != 50 {
		t.Fatalf("expected total=50 even on an out-of-range page, got %d", result.Total)
	}
}

func TestListFromCacheWithPagination_FastPath_DescendingOrder(t *testing.T) {
	im := newTestInformerManagerWithPods(20)

	result, ok := im.ListFromCacheWithPagination("pods", "", "", "name", "desc", 0, 5)
	if !ok {
		t.Fatal("expected cache hit")
	}
	for i, item := range result.Items {
		want := fmt.Sprintf("pod-%04d", 19-i)
		if got := item.GetName(); got != want {
			t.Errorf("item[%d]: want name=%s got=%s", i, want, got)
		}
	}
}

func TestListFromCacheWithPagination_FastPath_SortByCreationTimestamp(t *testing.T) {
	im := newTestInformerManagerWithPods(30)

	result, ok := im.ListFromCacheWithPagination("pods", "", "", "creationTimestamp", "desc", 0, 3)
	if !ok {
		t.Fatal("expected cache hit")
	}
	// Newest pod was created last (highest index), so desc-by-creation puts
	// pod-0029 first.
	if result.Items[0].GetName() != "pod-0029" {
		t.Errorf("expected newest pod first, got %s", result.Items[0].GetName())
	}
}

func TestListFromCacheWithPagination_FastPath_NamespaceFilter(t *testing.T) {
	im := newTestInformerManagerWithPods(100) // 5 namespaces, 20 pods each

	result, ok := im.ListFromCacheWithPagination("pods", "ns-02", "", "name", "asc", 0, 100)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if result.Total != 20 {
		t.Fatalf("expected 20 pods in ns-02, got total=%d", result.Total)
	}
	for _, item := range result.Items {
		if item.GetNamespace() != "ns-02" {
			t.Errorf("leaked item from namespace %s into ns-02 filter", item.GetNamespace())
		}
	}
}

func TestListFromCacheWithPagination_FastPath_SearchFilter(t *testing.T) {
	im := newTestInformerManagerWithPods(50)

	result, ok := im.ListFromCacheWithPagination("pods", "", "pod-002", "name", "asc", 0, 50)
	if !ok {
		t.Fatal("expected cache hit")
	}
	// pod-0020..pod-0029 => 10 matches
	if result.Total != 10 {
		t.Fatalf("expected 10 search matches, got %d", result.Total)
	}
}

// TestListFromCacheWithPagination_FastPathOrderingAgainstKnownExpectation
// proves the optimized typed path produces the exact ordering (including
// the namespace-then-name tie-break) that the pagination contract promises,
// across all three fast-path sort keys and both sort orders. The synthetic
// dataset's properties are known exactly (name is sequential with index,
// namespace is i%5 cyclic, creation timestamp is monotonic with index), so
// the expected order can be computed independently and compared directly —
// this is the core correctness guarantee for the 10K P1 remediation:
// callers (REST handler, frontend pagination) must see no behavioral
// difference from before the optimization.
func TestListFromCacheWithPagination_FastPathOrderingAgainstKnownExpectation(t *testing.T) {
	const n = 137 // deliberately not a round number
	im := newTestInformerManagerWithPods(n)

	type idxName struct {
		idx  int
		name string
	}
	all := make([]idxName, n)
	for i := 0; i < n; i++ {
		all[i] = idxName{idx: i, name: fmt.Sprintf("pod-%04d", i)}
	}

	for _, tc := range []struct {
		sortBy string
		order  string
		less   func(a, b idxName) bool // expected "a before b" when order=asc
	}{
		{"name", "asc", func(a, b idxName) bool { return a.name < b.name }},
		{"name", "desc", func(a, b idxName) bool { return a.name < b.name }},
		{"namespace", "asc", func(a, b idxName) bool {
			nsA, nsB := fmt.Sprintf("ns-%02d", a.idx%5), fmt.Sprintf("ns-%02d", b.idx%5)
			if nsA != nsB {
				return nsA < nsB
			}
			return a.name < b.name // tie-break
		}},
		{"namespace", "desc", func(a, b idxName) bool {
			nsA, nsB := fmt.Sprintf("ns-%02d", a.idx%5), fmt.Sprintf("ns-%02d", b.idx%5)
			if nsA != nsB {
				return nsA < nsB
			}
			return a.name < b.name
		}},
		{"creationTimestamp", "asc", func(a, b idxName) bool { return a.idx < b.idx }},
		{"creationTimestamp", "desc", func(a, b idxName) bool { return a.idx < b.idx }},
	} {
		expected := make([]idxName, n)
		copy(expected, all)
		descending := tc.order == "desc"
		sort.Slice(expected, func(i, j int) bool {
			if descending {
				return !tc.less(expected[i], expected[j])
			}
			return tc.less(expected[i], expected[j])
		})

		result, ok := im.ListFromCacheWithPagination("pods", "", "", tc.sortBy, tc.order, 0, n)
		if !ok {
			t.Fatalf("sortBy=%s order=%s: expected cache hit", tc.sortBy, tc.order)
		}
		if len(result.Items) != n {
			t.Fatalf("sortBy=%s order=%s: expected %d items, got %d", tc.sortBy, tc.order, n, len(result.Items))
		}
		for i, item := range result.Items {
			if got, want := item.GetName(), expected[i].name; got != want {
				t.Errorf("sortBy=%s order=%s: position %d: got name=%s want=%s", tc.sortBy, tc.order, i, got, want)
			}
		}
	}
}

// TestListFromCacheWithPagination_FastPath_DoesNotConvertBeyondRequestedPage
// is the direct performance-regression proof: with a large backing store, a
// small-limit request must not pay for converting the entire collection.
// We can't easily assert "didn't call ToUnstructured N times" via a black
// box test without instrumentation, so this test asserts the behavioral
// proxy that matters to callers: correctness holds at a scale (5,000) where
// the old implementation was measured (10K campaign) to take ~260-380ms of
// conversion alone, while this test's wall-clock budget assumes fast-path
// behavior (converting ~100 items, not 5,000).
func TestListFromCacheWithPagination_FastPath_SmallPageFromLargeStore(t *testing.T) {
	im := newTestInformerManagerWithPods(5000)

	start := time.Now()
	result, ok := im.ListFromCacheWithPagination("pods", "", "", "name", "asc", 0, 100)
	elapsed := time.Since(start)

	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(result.Items) != 100 {
		t.Fatalf("expected 100 items, got %d", len(result.Items))
	}
	if result.Total != 5000 {
		t.Fatalf("expected total=5000, got %d", result.Total)
	}
	// Generous upper bound — this is a correctness/sanity bound, not a
	// precise benchmark (CI machines vary). The old full-conversion path
	// measured ~260-380ms for conversion alone at ~10K objects in the 10K
	// campaign; this asserts we're well under that order of magnitude.
	if elapsed > 150*time.Millisecond {
		t.Errorf("fast path took %v for a 100-item page out of 5000 — expected well under 150ms", elapsed)
	}
}

func TestListFromCacheWithPagination_FastPath_ConcurrentRequestsNoRace(t *testing.T) {
	im := newTestInformerManagerWithPods(1000)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			result, ok := im.ListFromCacheWithPagination("pods", "", "", "name", "asc", offset*10, 10)
			if !ok {
				t.Errorf("offset=%d: expected cache hit", offset)
				return
			}
			if len(result.Items) != 10 {
				t.Errorf("offset=%d: expected 10 items, got %d", offset, len(result.Items))
			}
		}(i)
	}
	wg.Wait()
}

func TestListFromCacheWithPagination_SlowPath_StillWorksForExoticSortKeys(t *testing.T) {
	im := newTestInformerManagerWithPods(30)

	// "restarts" is not one of the fast-path keys — must still hit the
	// original unstructured-conversion path and return correct results.
	result, ok := im.ListFromCacheWithPagination("pods", "", "", "restarts", "asc", 0, 30)
	if !ok {
		t.Fatal("expected cache hit for exotic sort key")
	}
	if len(result.Items) != 30 || result.Total != 30 {
		t.Fatalf("expected all 30 items via slow path, got items=%d total=%d", len(result.Items), result.Total)
	}
}

func TestListFromCacheWithPagination_FastPath_ClusterIsolation(t *testing.T) {
	imA := newTestInformerManagerWithPods(10)
	imB := newTestInformerManagerWithPods(20)

	resultA, ok := imA.ListFromCacheWithPagination("pods", "", "", "name", "asc", 0, 100)
	if !ok || resultA.Total != 10 {
		t.Fatalf("cluster A: expected total=10, got ok=%v total=%v", ok, resultA.Total)
	}
	resultB, ok := imB.ListFromCacheWithPagination("pods", "", "", "name", "asc", 0, 100)
	if !ok || resultB.Total != 20 {
		t.Fatalf("cluster B: expected total=20, got ok=%v total=%v", ok, resultB.Total)
	}
}
