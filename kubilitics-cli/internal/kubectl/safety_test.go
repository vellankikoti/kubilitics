package kubectl

import (
	"testing"
)

func TestClassifyRisk(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected RiskLevel
	}{
		// Read-only commands — RiskNone
		{"get pods", []string{"get", "pods"}, RiskNone},
		{"describe deploy", []string{"describe", "deployment/api"}, RiskNone},
		{"logs", []string{"logs", "pod/api"}, RiskNone},
		{"explain", []string{"explain", "pods"}, RiskNone},
		{"api-resources", []string{"api-resources"}, RiskNone},
		{"auth can-i", []string{"auth", "can-i", "get", "pods"}, RiskNone},
		{"top", []string{"top", "nodes"}, RiskNone},
		{"version", []string{"version"}, RiskNone},
		{"config", []string{"config", "get-contexts"}, RiskNone},
		{"cluster-info", []string{"cluster-info"}, RiskNone},
		{"empty args", []string{}, RiskNone},

		// Apply/Patch — RiskMedium
		{"apply", []string{"apply", "-f", "deploy.yaml"}, RiskMedium},
		{"patch", []string{"patch", "deploy/api", "-p", "{}"}, RiskMedium},

		// Annotate/Label — RiskMedium
		{"annotate", []string{"annotate", "pod/api", "foo=bar"}, RiskMedium},
		{"label", []string{"label", "pod/api", "env=prod"}, RiskMedium},

		// Edit — RiskMedium
		{"edit", []string{"edit", "deploy/api"}, RiskMedium},

		// Replace/Create — RiskMedium
		{"replace", []string{"replace", "-f", "deploy.yaml"}, RiskMedium},
		{"create", []string{"create", "-f", "deploy.yaml"}, RiskMedium},

		// Scale with replicas — RiskLow
		{"scale up", []string{"scale", "deploy/api", "--replicas", "5"}, RiskLow},
		// Scale to zero — RiskCritical
		{"scale to 0", []string{"scale", "deploy/api", "--replicas", "0"}, RiskCritical},
		// Scale without replicas flag
		{"scale no flag", []string{"scale", "deploy/api"}, RiskLow},

		// Rollout restart — RiskMedium
		{"rollout restart", []string{"rollout", "restart", "deploy/api"}, RiskMedium},
		// Rollout undo — RiskHigh
		{"rollout undo", []string{"rollout", "undo", "deploy/api"}, RiskHigh},

		// Drain — RiskCritical
		{"drain", []string{"drain", "node01"}, RiskCritical},
		// Cordon — RiskHigh
		{"cordon", []string{"cordon", "node01"}, RiskHigh},
		// Uncordon — RiskHigh
		{"uncordon", []string{"uncordon", "node01"}, RiskHigh},

		// Taint — RiskHigh
		{"taint", []string{"taint", "node01", "key=val:NoSchedule"}, RiskHigh},

		// Delete namespace — RiskCritical
		{"delete namespace", []string{"delete", "namespace/prod"}, RiskCritical},
		{"delete ns", []string{"delete", "ns/prod"}, RiskCritical},

		// Delete deployment — RiskCritical
		{"delete deploy", []string{"delete", "deployment/api"}, RiskCritical},

		// Delete pod with name — RiskHigh
		{"delete specific pod", []string{"delete", "pod/api-123"}, RiskHigh},
		// Delete pods without name — RiskCritical
		{"delete pods broad", []string{"delete", "pods"}, RiskCritical},

		// Delete with --all — RiskCritical
		{"delete all pods", []string{"delete", "pods", "--all"}, RiskCritical},

		// Delete service — RiskCritical
		{"delete service", []string{"delete", "service/api"}, RiskCritical},

		// Delete PVC — RiskHigh
		{"delete pvc", []string{"delete", "pvc/data"}, RiskHigh},

		// Delete configmap — RiskMedium
		{"delete configmap", []string{"delete", "configmap/app-config"}, RiskMedium},
		{"delete secret", []string{"delete", "secret/api-key"}, RiskMedium},

		// Delete job with name — RiskHigh
		{"delete specific job", []string{"delete", "job/migrate"}, RiskHigh},
		// Delete jobs without name — RiskCritical
		{"delete jobs broad", []string{"delete", "jobs"}, RiskCritical},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyRisk(tc.args)
			if got != tc.expected {
				t.Errorf("ClassifyRisk(%v) = %d, want %d", tc.args, got, tc.expected)
			}
		})
	}
}

func TestClassifyRisk_DeleteWithSelector(t *testing.T) {
	// Delete pods with -l selector (broad) should be Critical
	args := []string{"delete", "pods", "-l", "app=api"}
	risk := ClassifyRisk(args)
	if risk != RiskCritical {
		t.Errorf("ClassifyRisk(%v) = %d, want RiskCritical(%d)", args, risk, RiskCritical)
	}
}
