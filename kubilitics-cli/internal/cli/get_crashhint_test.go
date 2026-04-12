package cli

import (
	"testing"
)

func TestIsCrashHintEligible(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		// Eligible cases
		{"simple pods", []string{"pods"}, true},
		{"pod singular", []string{"pod"}, true},
		{"po abbreviation", []string{"po"}, true},
		{"pods/name", []string{"pods/api-server"}, true},
		{"pods with wide", []string{"pods", "-o", "wide"}, true},
		{"pods with table", []string{"pods", "--output=table"}, true},

		// Not eligible — non-pod resources
		{"deployments", []string{"deployments"}, false},
		{"services", []string{"services"}, false},
		{"nodes", []string{"nodes"}, false},

		// Not eligible — non-table output
		{"pods yaml", []string{"pods", "-o", "yaml"}, false},
		{"pods json", []string{"pods", "-o", "json"}, false},
		{"pods jsonpath", []string{"pods", "-o", "jsonpath={.items}"}, false},
		{"pods output=json", []string{"pods", "--output=json"}, false},
		{"pods -o=json", []string{"pods", "-o=json"}, false},

		// Not eligible — watch mode
		{"pods watch", []string{"pods", "-w"}, false},
		{"pods --watch", []string{"pods", "--watch"}, false},
		{"pods output-watch-events", []string{"pods", "--output-watch-events"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isCrashHintEligible(tc.args)
			if got != tc.expected {
				t.Errorf("isCrashHintEligible(%v) = %v, want %v", tc.args, got, tc.expected)
			}
		})
	}
}

func TestParsePodCrashHints(t *testing.T) {
	tests := []struct {
		name       string
		tableInput string
		wantCount  int
	}{
		{
			name: "no problems",
			tableInput: `NAME            READY   STATUS    RESTARTS   AGE
api-7f9d        1/1     Running   0          2d
worker-5f8b7    1/1     Running   0          1d`,
			wantCount: 0,
		},
		{
			name: "one crash loop",
			tableInput: `NAME                READY   STATUS             RESTARTS   AGE
api-7f9d            1/1     Running            0          2d
worker-crash-5f8b7  0/1     CrashLoopBackOff   12         5m`,
			wantCount: 1,
		},
		{
			name: "multiple problems",
			tableInput: `NAME              READY   STATUS             RESTARTS   AGE
api-7f9d          1/1     Running            0          2d
worker-crash      0/1     CrashLoopBackOff   12         5m
oom-victim        0/1     OOMKilled          3          10m
img-err           0/1     ImagePullBackOff   0          2m`,
			wantCount: 3,
		},
		{
			name: "evicted and pending",
			tableInput: `NAME          READY   STATUS      RESTARTS   AGE
good-pod      1/1     Running     0          2d
evicted-pod   0/1     Evicted     0          1d
pending-pod   0/1     Pending     0          5m`,
			wantCount: 2,
		},
		{
			name:       "empty output",
			tableInput: "",
			wantCount:  0,
		},
		{
			name:       "header only",
			tableInput: "NAME   READY   STATUS   RESTARTS   AGE",
			wantCount:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hints := parsePodCrashHints(tc.tableInput)
			if len(hints) != tc.wantCount {
				t.Errorf("parsePodCrashHints() returned %d hints, want %d", len(hints), tc.wantCount)
				for _, h := range hints {
					t.Logf("  hint: pod=%s status=%s", h.PodName, h.Status)
				}
			}
		})
	}
}

func TestHasWithModifier(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{"has with", []string{"pods", "with", "ip,node"}, true},
		{"WITH uppercase", []string{"pods", "WITH", "ip"}, true},
		{"no with", []string{"pods", "-o", "wide"}, false},
		{"empty", []string{}, false},
		{"with at start", []string{"with"}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasWithModifier(tc.args)
			if got != tc.expected {
				t.Errorf("hasWithModifier(%v) = %v, want %v", tc.args, got, tc.expected)
			}
		})
	}
}
