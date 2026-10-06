package service

import "testing"

// Batch 2 / Theme 2 #13: fetchNetworkStats previously called each node's
// kubelet /stats/summary proxy endpoint serially — O(nodes)*5s worst case
// if any node was slow/unreachable. Replaced with a bounded errgroup
// fan-out, with the actual parsing logic pulled into parseNodeStatsSummary
// so it's unit-testable (a fake clientset's RESTClient() is nil, not a
// usable stub, so fetchNetworkStats itself can't be exercised without a
// real HTTP server). No test existed for any of this before this change.

func TestParseNodeStatsSummary_DirectRxTx(t *testing.T) {
	raw := []byte(`{"pods":[{"podRef":{"name":"my-pod","namespace":"default"},"network":{"rxBytes":100,"txBytes":200}}]}`)
	result, err := parseNodeStatsSummary(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stats, ok := result["default/my-pod"]
	if !ok {
		t.Fatal("expected an entry for default/my-pod")
	}
	if stats.rx != 100 || stats.tx != 200 {
		t.Errorf("expected rx=100 tx=200, got rx=%d tx=%d", stats.rx, stats.tx)
	}
}

func TestParseNodeStatsSummary_FallsBackToInterfaceSumWhenTopLevelZero(t *testing.T) {
	raw := []byte(`{"pods":[{"podRef":{"name":"my-pod","namespace":"default"},"network":{"rxBytes":0,"txBytes":0,"interfaces":[{"rxBytes":10,"txBytes":20},{"rxBytes":5,"txBytes":15}]}}]}`)
	result, err := parseNodeStatsSummary(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stats := result["default/my-pod"]
	if stats.rx != 15 || stats.tx != 35 {
		t.Errorf("expected rx=15 (10+5) tx=35 (20+15), got rx=%d tx=%d", stats.rx, stats.tx)
	}
}

func TestParseNodeStatsSummary_MultiplePods(t *testing.T) {
	raw := []byte(`{"pods":[
		{"podRef":{"name":"pod-a","namespace":"ns-a"},"network":{"rxBytes":1,"txBytes":2}},
		{"podRef":{"name":"pod-b","namespace":"ns-b"},"network":{"rxBytes":3,"txBytes":4}}
	]}`)
	result, err := parseNodeStatsSummary(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result["ns-a/pod-a"] != (podNetStats{rx: 1, tx: 2}) {
		t.Errorf("unexpected stats for ns-a/pod-a: %+v", result["ns-a/pod-a"])
	}
	if result["ns-b/pod-b"] != (podNetStats{rx: 3, tx: 4}) {
		t.Errorf("unexpected stats for ns-b/pod-b: %+v", result["ns-b/pod-b"])
	}
}

func TestParseNodeStatsSummary_NoPods_EmptyMap(t *testing.T) {
	raw := []byte(`{"pods":[]}`)
	result, err := parseNodeStatsSummary(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(result))
	}
}

func TestParseNodeStatsSummary_MalformedJSON_ReturnsError(t *testing.T) {
	raw := []byte(`not json`)
	if _, err := parseNodeStatsSummary(raw); err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
}
