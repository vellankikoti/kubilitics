// Package redact provides helpers to avoid exposing secret values in API responses or logs (C3.2).
package redact

import "strings"

const redactedValue = "***REDACTED***"

// SecretData redacts Kubernetes Secret .data and .stringData values in obj (in place).
// Keeps key names so clients know which keys exist; values are replaced with ***REDACTED***.
func SecretData(obj map[string]interface{}) {
	if obj == nil {
		return
	}
	if data, ok := obj["data"].(map[string]interface{}); ok {
		for k := range data {
			data[k] = redactedValue
		}
	}
	if stringData, ok := obj["stringData"].(map[string]interface{}); ok {
		for k := range stringData {
			stringData[k] = redactedValue
		}
	}
}

// IsSecretKind returns true if kind (e.g. "Secret", "secrets") indicates a Kubernetes Secret.
//
// Deliberately case-insensitive: the kind-resolution paths that actually fetch data
// (informer.go's resourceKindToStoreKey lookup, discovery.go's ResolveGVR fallback)
// are both case-insensitive, so an exact-string match here (the original
// implementation) let a request for e.g. "SECRETS" resolve real Secret data while
// silently skipping redaction — a real, exploitable unredacted-Secret-data bypass.
func IsSecretKind(kind string) bool {
	switch strings.ToLower(kind) {
	case "secret", "secrets":
		return true
	}
	return false
}
