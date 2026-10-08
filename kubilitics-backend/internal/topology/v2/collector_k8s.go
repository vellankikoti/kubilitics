package v2

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	nodev1 "k8s.io/api/node/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	storagev1 "k8s.io/api/storage/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"golang.org/x/sync/errgroup"

	"github.com/kubilitics/kubilitics-backend/internal/k8s"
)

// BLASTRADIUS-2 (docs/PRODUCTION-RELIABILITY-AUDIT.md): every resource type
// below except Pods/Events previously used a single ListOptions{Limit: 500}
// call with no continuation handling — any resource type with more than 500
// live instances was silently truncated, with no signal that the topology
// (and therefore blast-radius) graph was incomplete. paginatedCollect
// generalizes the pods/events continuation-loop pattern (which already
// existed and was already correct) to every resource type.
const (
	collectPageSize = 500
	// collectMaxPages bounds total pagination per resource type (50 x 500 =
	// 25,000 items) so a pathological resource count can't make a single
	// blast-radius/topology request paginate unboundedly. If the cap is hit
	// before the API server reports an empty continuation token, the
	// resource type is recorded via recordFailure (reusing the existing
	// FailedResources signal — see collector.go/response.go — rather than
	// inventing a second "incomplete" mechanism) so the caller can tell the
	// difference between "genuinely empty" and "stopped early."
	collectMaxPages = 50
)

// continuable is satisfied by every client-go generated List type (they all
// embed metav1.ListMeta, which promotes GetContinue()).
type continuable interface {
	GetContinue() string
}

// paginatedCollect lists a resource type to completion (or until
// collectMaxPages is reached), accumulating every page's items. Errors and a
// reached safety cap are both reported via recordFailure rather than
// propagated, matching every existing call site's "degrade gracefully, don't
// fail the whole bundle for one resource type" behavior.
func paginatedCollect[TList continuable, TItem any](
	ctx context.Context,
	resourceType string,
	recordFailure func(resourceType string, err error),
	listFn func(ctx context.Context, opts metav1.ListOptions) (TList, error),
	itemsOf func(TList) []TItem,
) []TItem {
	var all []TItem
	var continueToken string
	for page := 0; page < collectMaxPages; page++ {
		list, err := listFn(ctx, metav1.ListOptions{Limit: collectPageSize, Continue: continueToken})
		if err != nil {
			recordFailure(resourceType, err)
			return all
		}
		all = append(all, itemsOf(list)...)
		continueToken = list.GetContinue()
		if continueToken == "" {
			return all
		}
	}
	recordFailure(resourceType, fmt.Errorf("pagination safety cap reached after %d items (%d pages x %d) — result is incomplete",
		len(all), collectMaxPages, collectPageSize))
	return all
}

// CollectFromClient fills a ResourceBundle by listing resources from the given k8s client.
// Namespace filters namespaced resources; empty means all namespaces. im is
// the cluster's InformerManager, used to serve any informer-tracked resource
// type from cache (<1ms) instead of a live API call; pass nil to always
// live-fetch (e.g. a cluster whose informers haven't started yet).
func CollectFromClient(ctx context.Context, client *k8s.Client, namespace string, im *k8s.InformerManager) (*ResourceBundle, error) {
	return collectFromClient(ctx, client, namespace, nil, im)
}

// CollectRemainderFromClient behaves like CollectFromClient, except that any
// resource type already populated on seed is copied from seed instead of
// re-listed from the live K8s API.
//
// BLASTRADIUS-1 (docs/PRODUCTION-RELIABILITY-AUDIT.md): GetResourceTopology
// previously re-fetched every resource type live on each topology cache
// miss, even though the cluster's ClusterGraphEngine (internal/graph) was
// already maintaining an informer-cached copy of the overlapping resource
// types for blast-radius scoring. seed lets the caller pass that
// already-synced data through so this function only live-fetches the
// resource types the engine does not track (RBAC, Nodes, Namespaces,
// storage, events, etc.), without reducing the bundle's completeness — every
// field is still populated, just from two different sources.
func CollectRemainderFromClient(ctx context.Context, client *k8s.Client, namespace string, seed *ResourceBundle, im *k8s.InformerManager) (*ResourceBundle, error) {
	return collectFromClient(ctx, client, namespace, seed, im)
}

// collectWithCache serves resourceType from the cluster's informer cache
// when available (Theme 1 #3/#4 — collectFromClient previously bypassed the
// informer cache entirely, unlike buildClusterSummary/GetClusterSummary),
// falling back to the existing live-API pagination path on any cache miss
// (not yet synced, untracked resource type, or im == nil). A cache hit
// returns the complete set directly — no continuation-token pagination is
// needed since the informer store already holds everything in memory.
func collectWithCache[TList continuable, TItem any](
	ctx context.Context,
	im *k8s.InformerManager,
	namespace string,
	resourceType string,
	recordFailure func(resourceType string, err error),
	listFn func(ctx context.Context, opts metav1.ListOptions) (TList, error),
	itemsOf func(TList) []TItem,
) []TItem {
	if items, ok := k8s.ListTypedFromCache[TItem](im, resourceType, namespace, metav1.ListOptions{}); ok {
		return items
	}
	return paginatedCollect(ctx, resourceType, recordFailure, listFn, itemsOf)
}

func collectFromClient(ctx context.Context, client *k8s.Client, namespace string, seed *ResourceBundle, im *k8s.InformerManager) (*ResourceBundle, error) {
	if client == nil || client.Clientset == nil {
		return nil, nil
	}
	cs := client.Clientset
	nsOpts := namespace
	if namespace == "" {
		nsOpts = metav1.NamespaceAll
	}
	bundle := &ResourceBundle{}
	var failMu sync.Mutex
	recordFailure := func(resourceType string, err error) {
		slog.Warn("topology v2 collect "+resourceType, "error", err)
		failMu.Lock()
		bundle.FailedResources = append(bundle.FailedResources, resourceType)
		failMu.Unlock()
	}
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if seed != nil {
			bundle.Pods = seed.Pods
			return nil
		}
		bundle.Pods = collectWithCache(gctx, im, namespace, "pods", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error) {
				return cs.CoreV1().Pods(nsOpts).List(ctx, opts)
			},
			func(l *corev1.PodList) []corev1.Pod { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		if seed != nil {
			bundle.Deployments = seed.Deployments
			return nil
		}
		bundle.Deployments = collectWithCache(gctx, im, namespace, "deployments", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*appsv1.DeploymentList, error) {
				return cs.AppsV1().Deployments(nsOpts).List(ctx, opts)
			},
			func(l *appsv1.DeploymentList) []appsv1.Deployment { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		if seed != nil {
			bundle.ReplicaSets = seed.ReplicaSets
			return nil
		}
		bundle.ReplicaSets = collectWithCache(gctx, im, namespace, "replicasets", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*appsv1.ReplicaSetList, error) {
				return cs.AppsV1().ReplicaSets(nsOpts).List(ctx, opts)
			},
			func(l *appsv1.ReplicaSetList) []appsv1.ReplicaSet { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		if seed != nil {
			bundle.StatefulSets = seed.StatefulSets
			return nil
		}
		bundle.StatefulSets = collectWithCache(gctx, im, namespace, "statefulsets", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*appsv1.StatefulSetList, error) {
				return cs.AppsV1().StatefulSets(nsOpts).List(ctx, opts)
			},
			func(l *appsv1.StatefulSetList) []appsv1.StatefulSet { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		if seed != nil {
			bundle.DaemonSets = seed.DaemonSets
			return nil
		}
		bundle.DaemonSets = collectWithCache(gctx, im, namespace, "daemonsets", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*appsv1.DaemonSetList, error) {
				return cs.AppsV1().DaemonSets(nsOpts).List(ctx, opts)
			},
			func(l *appsv1.DaemonSetList) []appsv1.DaemonSet { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		if seed != nil {
			bundle.Jobs = seed.Jobs
			return nil
		}
		bundle.Jobs = collectWithCache(gctx, im, namespace, "jobs", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*batchv1.JobList, error) {
				return cs.BatchV1().Jobs(nsOpts).List(ctx, opts)
			},
			func(l *batchv1.JobList) []batchv1.Job { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		if seed != nil {
			bundle.CronJobs = seed.CronJobs
			return nil
		}
		bundle.CronJobs = collectWithCache(gctx, im, namespace, "cronjobs", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*batchv1.CronJobList, error) {
				return cs.BatchV1().CronJobs(nsOpts).List(ctx, opts)
			},
			func(l *batchv1.CronJobList) []batchv1.CronJob { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		if seed != nil {
			bundle.Services = seed.Services
			return nil
		}
		bundle.Services = collectWithCache(gctx, im, namespace, "services", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*corev1.ServiceList, error) {
				return cs.CoreV1().Services(nsOpts).List(ctx, opts)
			},
			func(l *corev1.ServiceList) []corev1.Service { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		if seed != nil {
			bundle.Endpoints = seed.Endpoints
			return nil
		}
		bundle.Endpoints = collectWithCache(gctx, im, namespace, "endpoints", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*corev1.EndpointsList, error) {
				return cs.CoreV1().Endpoints(nsOpts).List(ctx, opts)
			},
			//nolint:staticcheck // corev1.Endpoints is deprecated in favor of
			// discoveryv1.EndpointSlice (collected separately below as
			// bundle.EndpointSlices), but still collected here for clusters
			// where relationship inference needs the legacy object (and for
			// older clusters that may not fully populate EndpointSlice).
			func(l *corev1.EndpointsList) []corev1.Endpoints { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.EndpointSlices = paginatedCollect(gctx, "endpointslices", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*discoveryv1.EndpointSliceList, error) {
				return cs.DiscoveryV1().EndpointSlices(nsOpts).List(ctx, opts)
			},
			func(l *discoveryv1.EndpointSliceList) []discoveryv1.EndpointSlice { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		if seed != nil {
			bundle.Ingresses = seed.Ingresses
			return nil
		}
		bundle.Ingresses = collectWithCache(gctx, im, namespace, "ingresses", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*networkingv1.IngressList, error) {
				return cs.NetworkingV1().Ingresses(nsOpts).List(ctx, opts)
			},
			func(l *networkingv1.IngressList) []networkingv1.Ingress { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.IngressClasses = collectWithCache(gctx, im, namespace, "ingressclasses", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*networkingv1.IngressClassList, error) {
				return cs.NetworkingV1().IngressClasses().List(ctx, opts)
			},
			func(l *networkingv1.IngressClassList) []networkingv1.IngressClass { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		if seed != nil {
			bundle.ConfigMaps = seed.ConfigMaps
			return nil
		}
		bundle.ConfigMaps = collectWithCache(gctx, im, namespace, "configmaps", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*corev1.ConfigMapList, error) {
				return cs.CoreV1().ConfigMaps(nsOpts).List(ctx, opts)
			},
			func(l *corev1.ConfigMapList) []corev1.ConfigMap { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		if seed != nil {
			bundle.Secrets = seed.Secrets
			return nil
		}
		bundle.Secrets = collectWithCache(gctx, im, namespace, "secrets", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*corev1.SecretList, error) {
				return cs.CoreV1().Secrets(nsOpts).List(ctx, opts)
			},
			func(l *corev1.SecretList) []corev1.Secret { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		if seed != nil {
			bundle.PVCs = seed.PVCs
			return nil
		}
		bundle.PVCs = collectWithCache(gctx, im, namespace, "persistentvolumeclaims", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*corev1.PersistentVolumeClaimList, error) {
				return cs.CoreV1().PersistentVolumeClaims(nsOpts).List(ctx, opts)
			},
			func(l *corev1.PersistentVolumeClaimList) []corev1.PersistentVolumeClaim { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.PVs = collectWithCache(gctx, im, namespace, "persistentvolumes", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*corev1.PersistentVolumeList, error) {
				return cs.CoreV1().PersistentVolumes().List(ctx, opts)
			},
			func(l *corev1.PersistentVolumeList) []corev1.PersistentVolume { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.StorageClasses = collectWithCache(gctx, im, namespace, "storageclasses", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*storagev1.StorageClassList, error) {
				return cs.StorageV1().StorageClasses().List(ctx, opts)
			},
			func(l *storagev1.StorageClassList) []storagev1.StorageClass { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.Nodes = collectWithCache(gctx, im, namespace, "nodes", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*corev1.NodeList, error) {
				return cs.CoreV1().Nodes().List(ctx, opts)
			},
			func(l *corev1.NodeList) []corev1.Node { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.Namespaces = collectWithCache(gctx, im, namespace, "namespaces", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*corev1.NamespaceList, error) {
				return cs.CoreV1().Namespaces().List(ctx, opts)
			},
			func(l *corev1.NamespaceList) []corev1.Namespace { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		if seed != nil {
			bundle.ServiceAccounts = seed.ServiceAccounts
			return nil
		}
		bundle.ServiceAccounts = collectWithCache(gctx, im, namespace, "serviceaccounts", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*corev1.ServiceAccountList, error) {
				return cs.CoreV1().ServiceAccounts(nsOpts).List(ctx, opts)
			},
			func(l *corev1.ServiceAccountList) []corev1.ServiceAccount { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.Roles = collectWithCache(gctx, im, namespace, "roles", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*rbacv1.RoleList, error) {
				return cs.RbacV1().Roles(nsOpts).List(ctx, opts)
			},
			func(l *rbacv1.RoleList) []rbacv1.Role { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.RoleBindings = collectWithCache(gctx, im, namespace, "rolebindings", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*rbacv1.RoleBindingList, error) {
				return cs.RbacV1().RoleBindings(nsOpts).List(ctx, opts)
			},
			func(l *rbacv1.RoleBindingList) []rbacv1.RoleBinding { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.ClusterRoles = collectWithCache(gctx, im, namespace, "clusterroles", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*rbacv1.ClusterRoleList, error) {
				return cs.RbacV1().ClusterRoles().List(ctx, opts)
			},
			func(l *rbacv1.ClusterRoleList) []rbacv1.ClusterRole { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.ClusterRoleBindings = collectWithCache(gctx, im, namespace, "clusterrolebindings", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*rbacv1.ClusterRoleBindingList, error) {
				return cs.RbacV1().ClusterRoleBindings().List(ctx, opts)
			},
			func(l *rbacv1.ClusterRoleBindingList) []rbacv1.ClusterRoleBinding { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.HPAs = collectWithCache(gctx, im, namespace, "horizontalpodautoscalers", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*autoscalingv2.HorizontalPodAutoscalerList, error) {
				return cs.AutoscalingV2().HorizontalPodAutoscalers(nsOpts).List(ctx, opts)
			},
			func(l *autoscalingv2.HorizontalPodAutoscalerList) []autoscalingv2.HorizontalPodAutoscaler { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		if seed != nil {
			bundle.PDBs = seed.PDBs
			return nil
		}
		bundle.PDBs = collectWithCache(gctx, im, namespace, "poddisruptionbudgets", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*policyv1.PodDisruptionBudgetList, error) {
				return cs.PolicyV1().PodDisruptionBudgets(nsOpts).List(ctx, opts)
			},
			func(l *policyv1.PodDisruptionBudgetList) []policyv1.PodDisruptionBudget { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		if seed != nil {
			bundle.NetworkPolicies = seed.NetworkPolicies
			return nil
		}
		bundle.NetworkPolicies = collectWithCache(gctx, im, namespace, "networkpolicies", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*networkingv1.NetworkPolicyList, error) {
				return cs.NetworkingV1().NetworkPolicies(nsOpts).List(ctx, opts)
			},
			func(l *networkingv1.NetworkPolicyList) []networkingv1.NetworkPolicy { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.PriorityClasses = paginatedCollect(gctx, "priorityclasses", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*schedulingv1.PriorityClassList, error) {
				return cs.SchedulingV1().PriorityClasses().List(ctx, opts)
			},
			func(l *schedulingv1.PriorityClassList) []schedulingv1.PriorityClass { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.RuntimeClasses = paginatedCollect(gctx, "runtimeclasses", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*nodev1.RuntimeClassList, error) {
				return cs.NodeV1().RuntimeClasses().List(ctx, opts)
			},
			func(l *nodev1.RuntimeClassList) []nodev1.RuntimeClass { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.MutatingWebhooks = paginatedCollect(gctx, "mutatingwebhooks", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*admissionregistrationv1.MutatingWebhookConfigurationList, error) {
				return cs.AdmissionregistrationV1().MutatingWebhookConfigurations().List(ctx, opts)
			},
			func(l *admissionregistrationv1.MutatingWebhookConfigurationList) []admissionregistrationv1.MutatingWebhookConfiguration {
				return l.Items
			},
		)
		return nil
	})
	g.Go(func() error {
		bundle.ValidatingWebhooks = paginatedCollect(gctx, "validatingwebhooks", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*admissionregistrationv1.ValidatingWebhookConfigurationList, error) {
				return cs.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(ctx, opts)
			},
			func(l *admissionregistrationv1.ValidatingWebhookConfigurationList) []admissionregistrationv1.ValidatingWebhookConfiguration {
				return l.Items
			},
		)
		return nil
	})
	g.Go(func() error {
		bundle.Events = collectWithCache(gctx, im, namespace, "events", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*corev1.EventList, error) {
				return cs.CoreV1().Events(nsOpts).List(ctx, opts)
			},
			func(l *corev1.EventList) []corev1.Event { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.ResourceQuotas = paginatedCollect(gctx, "resourcequotas", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*corev1.ResourceQuotaList, error) {
				return cs.CoreV1().ResourceQuotas(nsOpts).List(ctx, opts)
			},
			func(l *corev1.ResourceQuotaList) []corev1.ResourceQuota { return l.Items },
		)
		return nil
	})
	g.Go(func() error {
		bundle.LimitRanges = paginatedCollect(gctx, "limitranges", recordFailure,
			func(ctx context.Context, opts metav1.ListOptions) (*corev1.LimitRangeList, error) {
				return cs.CoreV1().LimitRanges(nsOpts).List(ctx, opts)
			},
			func(l *corev1.LimitRangeList) []corev1.LimitRange { return l.Items },
		)
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return bundle, nil
}
