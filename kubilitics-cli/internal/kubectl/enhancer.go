package kubectl

import (
	"fmt"
	"strings"
)

// ParseWithModifiers extracts "with" modifiers from command args.
// Example: ["get", "pods", "with", "ip,node"] →
//   verb="get", resource="pods", modifiers=["ip","node"], remainingArgs=[]
func ParseWithModifiers(args []string) (verb string, resource string, modifiers []string, remainingArgs []string, err error) {
	if len(args) < 2 {
		return "", "", nil, args, fmt.Errorf("invalid command: expected at least verb and resource")
	}

	verb = strings.ToLower(args[0])
	resource = strings.ToLower(args[1])

	if verb != "get" {
		return verb, resource, nil, args[2:], fmt.Errorf("'with' modifiers only supported with 'get' command")
	}

	withIdx := -1
	for i := 2; i < len(args); i++ {
		if strings.EqualFold(args[i], "with") {
			withIdx = i
			break
		}
	}

	if withIdx == -1 {
		return verb, resource, nil, args[2:], nil
	}

	if withIdx+1 >= len(args) {
		return verb, resource, nil, args[2:], fmt.Errorf("'with' keyword requires modifiers")
	}

	// Collect all modifier tokens after "with" until we hit a flag (-) or end.
	// Supports both comma-separated ("ip,node") and space-separated ("ip node").
	for i := withIdx + 1; i < len(args); i++ {
		token := strings.TrimSpace(args[i])
		if strings.HasPrefix(token, "-") {
			// This is a flag — stop collecting modifiers, rest are remaining args
			remainingArgs = args[i:]
			break
		}
		// Split by comma in case of "ip,node" format
		parts := strings.Split(token, ",")
		for _, p := range parts {
			p = strings.TrimSpace(strings.ToLower(p))
			if p != "" {
				modifiers = append(modifiers, p)
			}
		}
		if i == len(args)-1 {
			remainingArgs = nil
		}
	}

	return verb, resource, modifiers, remainingArgs, nil
}

// EnhancedGet performs a client-go API call and renders enriched output.
// Routes to resource-specific handlers based on the resource type.
func EnhancedGet(kubeconfigPath, context, namespace string, resource string, modifiers []string, allNamespaces bool, sortBy string, outputFormat string) error {
	resource = strings.ToLower(resource)

	switch resource {
	case "pod", "pods", "po":
		return enhancePods(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "deployment", "deployments", "deploy":
		return enhanceDeployments(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "service", "services", "svc":
		return enhanceServices(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "node", "nodes":
		return enhanceNodes(kubeconfigPath, context, modifiers, sortBy, outputFormat)

	case "persistentvolume", "persistentvolumes", "pv":
		return enhancePersistentVolumes(kubeconfigPath, modifiers, sortBy, outputFormat)

	case "persistentvolumeclaim", "persistentvolumeclaims", "pvc":
		return enhancePersistentVolumeClaims(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "ingress", "ingresses", "ing":
		return enhanceIngresses(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "event", "events", "ev":
		return enhanceEvents(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "statefulset", "statefulsets", "sts":
		return enhanceStatefulSets(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "daemonset", "daemonsets", "ds":
		return enhanceDaemonSets(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "namespace", "namespaces", "ns":
		return enhanceNamespaces(kubeconfigPath, context, modifiers, sortBy, outputFormat)

	case "configmap", "configmaps", "cm":
		return enhanceConfigMaps(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "secret", "secrets":
		return enhanceSecrets(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "job", "jobs":
		return enhanceJobs(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "cronjob", "cronjobs", "cj":
		return enhanceCronJobs(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "replicaset", "replicasets", "rs":
		return enhanceReplicaSets(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "serviceaccount", "serviceaccounts", "sa":
		return enhanceServiceAccounts(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "endpoints", "ep":
		return enhanceEndpoints(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "horizontalpodautoscaler", "horizontalpodautoscalers", "hpa":
		return enhanceHPAs(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "networkpolicy", "networkpolicies", "netpol":
		return enhanceNetworkPolicies(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "role", "roles":
		return enhanceRoles(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "rolebinding", "rolebindings":
		return enhanceRoleBindings(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "clusterrole", "clusterroles":
		return enhanceClusterRoles(kubeconfigPath, context, modifiers, sortBy, outputFormat)

	case "clusterrolebinding", "clusterrolebindings":
		return enhanceClusterRoleBindings(kubeconfigPath, context, modifiers, sortBy, outputFormat)

	case "storageclass", "storageclasses", "sc":
		return enhanceStorageClasses(kubeconfigPath, context, modifiers, sortBy, outputFormat)

	case "limitrange", "limitranges":
		return enhanceLimitRanges(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	case "resourcequota", "resourcequotas":
		return enhanceResourceQuotas(kubeconfigPath, context, namespace, modifiers, allNamespaces, sortBy, outputFormat)

	default:
		// Unsupported resource type — fall back to kubectl passthrough
		return fmt.Errorf("enhanced output not available for: %s (use kubectl get %s)", resource, resource)
	}
}

func validateModifiers(resource string, requested []string, supported map[string]bool) error {
	for _, mod := range requested {
		if mod == "all" {
			continue
		}
		if !supported[mod] {
			return fmt.Errorf("modifier '%s' not supported for %s", mod, resource)
		}
	}
	return nil
}

func parseAllModifier(requested []string, supported map[string]bool) []string {
	for _, mod := range requested {
		if mod == "all" {
			expanded := make([]string, 0)
			for key := range supported {
				if key != "all" {
					expanded = append(expanded, key)
				}
			}
			return expanded
		}
	}
	return requested
}
