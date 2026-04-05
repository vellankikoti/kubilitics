package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ChangeDetector identifies and records resource changes by computing
// field-level diffs between old and new resource specs.
type ChangeDetector struct {
	store *Store
}

// NewChangeDetector creates a new change detector.
func NewChangeDetector(store *Store) *ChangeDetector {
	return &ChangeDetector{store: store}
}

// DetectChange computes a diff between old and new resource specs, classifies
// the change type, and returns a Change record ready for persistence.
// Returns nil if no differences are found.
func (cd *ChangeDetector) DetectChange(
	ctx context.Context,
	clusterID, kind, name, namespace, uid string,
	oldSpec, newSpec map[string]interface{},
) *Change {
	diffs := ComputeDiff(oldSpec, newSpec)
	if len(diffs) == 0 {
		return nil
	}

	changeType := ClassifyChange(kind, diffs)

	fieldChangesJSON, _ := json.Marshal(diffs)

	return &Change{
		ChangeID:          fmt.Sprintf("chg_%s", uuid.New().String()[:12]),
		Timestamp:         UnixMillis(),
		ClusterID:         clusterID,
		ResourceKind:      kind,
		ResourceName:      name,
		ResourceNamespace: namespace,
		ResourceUID:       uid,
		ChangeType:        changeType,
		FieldChanges:      fieldChangesJSON,
		ChangeSource:      "k8s-watch",
	}
}

// ComputeDiff recursively walks two maps and returns all fields that differ.
// Field paths use dot-notation (e.g., "spec.replicas", "spec.template.spec.containers.0.image").
func ComputeDiff(old, new map[string]interface{}) []FieldChange {
	var changes []FieldChange
	computeDiffRecursive("", old, new, &changes)
	return changes
}

// computeDiffRecursive is the recursive helper for ComputeDiff.
func computeDiffRecursive(prefix string, old, new map[string]interface{}, changes *[]FieldChange) {
	// Check for fields in new that are added or changed
	for key, newVal := range new {
		path := joinPath(prefix, key)
		oldVal, exists := old[key]

		if !exists {
			*changes = append(*changes, FieldChange{
				Field:    path,
				OldValue: "",
				NewValue: formatValue(newVal),
			})
			continue
		}

		// Both exist — compare
		oldMap, oldIsMap := oldVal.(map[string]interface{})
		newMap, newIsMap := newVal.(map[string]interface{})

		if oldIsMap && newIsMap {
			computeDiffRecursive(path, oldMap, newMap, changes)
			continue
		}

		oldSlice, oldIsSlice := oldVal.([]interface{})
		newSlice, newIsSlice := newVal.([]interface{})

		if oldIsSlice && newIsSlice {
			computeSliceDiff(path, oldSlice, newSlice, changes)
			continue
		}

		// Scalar comparison
		if formatValue(oldVal) != formatValue(newVal) {
			*changes = append(*changes, FieldChange{
				Field:    path,
				OldValue: formatValue(oldVal),
				NewValue: formatValue(newVal),
			})
		}
	}

	// Check for fields removed in new
	for key, oldVal := range old {
		if _, exists := new[key]; !exists {
			path := joinPath(prefix, key)
			*changes = append(*changes, FieldChange{
				Field:    path,
				OldValue: formatValue(oldVal),
				NewValue: "",
			})
		}
	}
}

// computeSliceDiff compares two slices element by element.
func computeSliceDiff(prefix string, old, new []interface{}, changes *[]FieldChange) {
	maxLen := len(old)
	if len(new) > maxLen {
		maxLen = len(new)
	}

	for i := 0; i < maxLen; i++ {
		path := fmt.Sprintf("%s.%d", prefix, i)

		if i >= len(old) {
			*changes = append(*changes, FieldChange{
				Field:    path,
				OldValue: "",
				NewValue: formatValue(new[i]),
			})
			continue
		}
		if i >= len(new) {
			*changes = append(*changes, FieldChange{
				Field:    path,
				OldValue: formatValue(old[i]),
				NewValue: "",
			})
			continue
		}

		oldMap, oldIsMap := old[i].(map[string]interface{})
		newMap, newIsMap := new[i].(map[string]interface{})

		if oldIsMap && newIsMap {
			computeDiffRecursive(path, oldMap, newMap, changes)
		} else if formatValue(old[i]) != formatValue(new[i]) {
			*changes = append(*changes, FieldChange{
				Field:    path,
				OldValue: formatValue(old[i]),
				NewValue: formatValue(new[i]),
			})
		}
	}
}

// ClassifyChange determines the change type based on the resource kind and
// which fields were modified.
func ClassifyChange(kind string, diffs []FieldChange) string {
	for _, d := range diffs {
		field := strings.ToLower(d.Field)

		// Image update detection
		if strings.Contains(field, "image") {
			// Check if only the tag changed (same image name, different tag)
			if isTagOnlyChange(d.OldValue, d.NewValue) {
				return "image_update"
			}
			return "rollout"
		}

		// Replica count change
		if strings.Contains(field, "replicas") {
			return "scale"
		}
	}

	// Kind-based classification
	switch kind {
	case "ConfigMap", "Secret":
		return "config_update"
	case "NetworkPolicy", "Role", "ClusterRole", "RoleBinding", "ClusterRoleBinding":
		return "policy_change"
	case "Deployment", "StatefulSet", "DaemonSet":
		return "rollout"
	}

	return "update"
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// joinPath creates a dot-separated path, handling empty prefix.
func joinPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// formatValue converts an arbitrary value to its string representation.
func formatValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

// isTagOnlyChange checks if two image references differ only in the tag.
// e.g., "nginx:1.19" vs "nginx:1.20" -> true
// e.g., "nginx:1.19" vs "redis:latest" -> false
func isTagOnlyChange(oldImage, newImage string) bool {
	oldParts := strings.SplitN(oldImage, ":", 2)
	newParts := strings.SplitN(newImage, ":", 2)

	if len(oldParts) < 2 || len(newParts) < 2 {
		return false
	}

	return oldParts[0] == newParts[0] && oldParts[1] != newParts[1]
}
