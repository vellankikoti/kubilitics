package cli

import (
	"testing"
)

func TestMapResourceName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Pods
		{"pod", "pods"},
		{"pods", "pods"},
		{"po", "pods"},
		// Deployments
		{"deployment", "deployments"},
		{"deployments", "deployments"},
		{"deploy", "deployments"},
		// Services
		{"service", "services"},
		{"services", "services"},
		{"svc", "services"},
		// ConfigMaps
		{"configmap", "configmaps"},
		{"configmaps", "configmaps"},
		{"cm", "configmaps"},
		// Secrets
		{"secret", "secrets"},
		{"secrets", "secrets"},
		// StatefulSets
		{"statefulset", "statefulsets"},
		{"sts", "statefulsets"},
		// DaemonSets
		{"daemonset", "daemonsets"},
		{"ds", "daemonsets"},
		// Jobs
		{"job", "jobs"},
		{"jobs", "jobs"},
		// CronJobs
		{"cronjob", "cronjobs"},
		{"cj", "cronjobs"},
		// PVC
		{"pvc", "persistentvolumeclaims"},
		// PV
		{"pv", "persistentvolumes"},
		// Namespaces
		{"namespace", "namespaces"},
		{"ns", "namespaces"},
		// Nodes
		{"node", "nodes"},
		{"nodes", "nodes"},
		// Ingresses
		{"ingress", "ingresses"},
		{"ing", "ingresses"},
		// Events
		{"event", "events"},
		{"ev", "events"},
		// ServiceAccount
		{"sa", "serviceaccounts"},
		// Unknown passes through
		{"customresource", "customresource"},
		// Case insensitive (already lowercased in function)
		{"PODS", "pods"}, // function lowercases input before lookup
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := mapResourceName(tc.input)
			if got != tc.expected {
				t.Errorf("mapResourceName(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestMatchesSelector(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		selector map[string]string
		expected bool
	}{
		{
			name:     "matching labels",
			labels:   map[string]string{"app": "api", "env": "prod"},
			selector: map[string]string{"app": "api"},
			expected: true,
		},
		{
			name:     "non-matching labels",
			labels:   map[string]string{"app": "worker"},
			selector: map[string]string{"app": "api"},
			expected: false,
		},
		{
			name:     "empty selector",
			labels:   map[string]string{"app": "api"},
			selector: map[string]string{},
			expected: false,
		},
		{
			name:     "nil selector",
			labels:   map[string]string{"app": "api"},
			selector: nil,
			expected: false,
		},
		{
			name:     "missing label key",
			labels:   map[string]string{"env": "prod"},
			selector: map[string]string{"app": "api"},
			expected: false,
		},
		{
			name:     "multi-key match",
			labels:   map[string]string{"app": "api", "env": "prod", "version": "v2"},
			selector: map[string]string{"app": "api", "env": "prod"},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchesSelector(tc.labels, tc.selector)
			if got != tc.expected {
				t.Errorf("matchesSelector() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestGetMapKeys(t *testing.T) {
	m := map[string]bool{"a": true, "b": true, "c": true}
	keys := getMapKeys(m)
	if len(keys) != 3 {
		t.Errorf("getMapKeys() returned %d keys, want 3", len(keys))
	}
	// Check all keys are present
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	for k := range m {
		if !keySet[k] {
			t.Errorf("getMapKeys() missing key %q", k)
		}
	}
}
