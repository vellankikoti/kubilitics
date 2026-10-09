package redact

import "testing"

// Regression test for a real security finding (docs/ai/STABILIZATION-PLAN.md
// Phase 0.1): IsSecretKind was an exact-string switch, but the paths that
// actually resolve `kind` to real data (informer.go's lowercase cache-key
// lookup, discovery.go's ResolveGVR EqualFold fallback) are case-insensitive.
// A request for e.g. "SECRETS" returned real, unredacted Secret data because
// IsSecretKind("SECRETS") was false and redaction never ran.
func TestIsSecretKind(t *testing.T) {
	trueCases := []string{
		"Secret", "secret", "Secrets", "secrets",
		"SECRET", "SECRETS", "SeCrEt", "SeCrEtS",
	}
	for _, kind := range trueCases {
		if !IsSecretKind(kind) {
			t.Errorf("IsSecretKind(%q) = false, want true — this case would bypass redaction", kind)
		}
	}

	falseCases := []string{"", "Pod", "pods", "ConfigMap", "SecretList"}
	for _, kind := range falseCases {
		if IsSecretKind(kind) {
			t.Errorf("IsSecretKind(%q) = true, want false", kind)
		}
	}
}
