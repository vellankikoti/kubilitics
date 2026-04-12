package cli

import "strings"

// mapResourceName normalises resource type strings to their canonical plural form.
// It handles singular/plural forms and common kubectl abbreviations.
func mapResourceName(resourceType string) string {
	resourceType = strings.ToLower(resourceType)

	mappings := map[string]string{
		"pod": "pods", "pods": "pods", "po": "pods",
		"service": "services", "services": "services", "svc": "services",
		"deployment": "deployments", "deployments": "deployments", "deploy": "deployments",
		"node": "nodes", "nodes": "nodes",
		"namespace": "namespaces", "namespaces": "namespaces", "ns": "namespaces",
		"secret": "secrets", "secrets": "secrets",
		"configmap": "configmaps", "configmaps": "configmaps", "cm": "configmaps",
		"ingress": "ingresses", "ingresses": "ingresses", "ing": "ingresses",
		"job": "jobs", "jobs": "jobs",
		"cronjob": "cronjobs", "cronjobs": "cronjobs", "cj": "cronjobs",
		"pvc": "persistentvolumeclaims", "persistentvolumeclaim": "persistentvolumeclaims", "persistentvolumeclaims": "persistentvolumeclaims",
		"pv": "persistentvolumes", "persistentvolume": "persistentvolumes", "persistentvolumes": "persistentvolumes",
		"all": "all",
		"statefulset": "statefulsets", "statefulsets": "statefulsets", "sts": "statefulsets",
		"daemonset": "daemonsets", "daemonsets": "daemonsets", "ds": "daemonsets",
		"event": "events", "events": "events", "ev": "events",
		"replicaset": "replicasets", "replicasets": "replicasets", "rs": "replicasets",
		"hpa": "horizontalpodautoscalers", "horizontalpodautoscaler": "horizontalpodautoscalers",
		"sa": "serviceaccounts", "serviceaccount": "serviceaccounts", "serviceaccounts": "serviceaccounts",
		"ep": "endpoints", "endpoints": "endpoints",
	}

	if mapped, exists := mappings[resourceType]; exists {
		return mapped
	}
	return resourceType
}

// getMapKeys returns the keys of a map[string]bool as a string slice.
func getMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// matchesSelector returns true when all selector key-value pairs exist in labels.
func matchesSelector(labels map[string]string, selector map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	for key, val := range selector {
		if labels[key] != val {
			return false
		}
	}
	return true
}
