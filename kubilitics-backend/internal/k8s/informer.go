package k8s

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// ResourceEventHandler handles resource events
type ResourceEventHandler func(eventType string, obj interface{})

// InformerManager manages Kubernetes informers for real-time updates.
// After Start() completes and HasSynced() returns true, its stores contain
// a full snapshot of all cluster resources — reads are sub-millisecond.
type InformerManager struct {
	client   *Client
	factory  informers.SharedInformerFactory
	stopCh   chan struct{}
	handlers map[string]ResourceEventHandler
	stores   map[string]cache.Store
	synced   atomic.Bool // true after WaitForCacheSync succeeds
}

// NewInformerManager creates a new informer manager
func NewInformerManager(client *Client) *InformerManager {
	// Resync period: periodic full re-list from K8s API as a consistency check.
	// 30s was too aggressive — with 25+ resource types per cluster it generates
	// a constant stream of LIST calls even when nothing changes. 5 minutes matches
	// Headlamp's approach: informers get real-time Watch events; the resync is just
	// a safety net for missed events, not a data source.
	factory := informers.NewSharedInformerFactory(client.Clientset, 5*time.Minute)

	return &InformerManager{
		client:   client,
		factory:  factory,
		stopCh:   make(chan struct{}),
		handlers: make(map[string]ResourceEventHandler),
		stores:   make(map[string]cache.Store),
	}
}

// RegisterHandler registers an event handler for a resource type
func (im *InformerManager) RegisterHandler(resourceType string, handler ResourceEventHandler) {
	im.handlers[resourceType] = handler
}

// Start starts all informers
func (im *InformerManager) Start(ctx context.Context) error {
	// Core resources
	im.setupPodInformer()
	im.setupServiceInformer()
	im.setupConfigMapInformer()
	im.setupSecretInformer()
	im.setupNodeInformer()
	im.setupNamespaceInformer()
	im.setupPersistentVolumeInformer()
	im.setupPersistentVolumeClaimInformer()
	im.setupServiceAccountInformer()
	im.setupEndpointsInformer()
	im.setupEventInformer()

	// Apps resources
	im.setupDeploymentInformer()
	im.setupReplicaSetInformer()
	im.setupStatefulSetInformer()
	im.setupDaemonSetInformer()

	// Batch resources
	im.setupJobInformer()
	im.setupCronJobInformer()

	// Networking resources
	im.setupIngressInformer()
	im.setupIngressClassInformer()
	im.setupNetworkPolicyInformer()

	// RBAC resources
	im.setupRoleInformer()
	im.setupRoleBindingInformer()
	im.setupClusterRoleInformer()
	im.setupClusterRoleBindingInformer()

	// Storage resources
	im.setupStorageClassInformer()

	// Autoscaling resources
	im.setupHorizontalPodAutoscalerInformer()

	// Policy resources
	im.setupPodDisruptionBudgetInformer()

	// Start all informers
	im.factory.Start(im.stopCh)

	// LOADING-5 (docs/PRODUCTION-RELIABILITY-AUDIT.md): WaitForCacheSync(im.stopCh)
	// previously blocked forever if any single resource type never completed its
	// initial sync (e.g. an RBAC watch denial on one resource) — im.stopCh only
	// closes on an explicit Stop(), so nothing ever woke this up. This only ever
	// blocked a background goroutine (StartClusterCache launches Start via `go
	// func(){}`), never an HTTP request, but left the cluster's cache permanently
	// "still warming up" with no actionable signal and no further retry. Bound the
	// initial wait, then keep retrying in the background instead of giving up.
	if im.waitForSync(initialSyncTimeout) {
		im.synced.Store(true)
		return nil
	}

	select {
	case <-im.stopCh:
		// A real Stop() was requested while waiting — not a timeout. Don't retry.
		return fmt.Errorf("informer manager stopped before initial cache sync completed")
	default:
	}

	log.Printf("informer cache did not finish initial sync within %s; resource reads for this cluster will use the slower direct API fallback until it succeeds. Retrying in the background every %s.",
		initialSyncTimeout, syncRetryInterval)
	go im.retrySyncInBackground()
	return fmt.Errorf("informer cache sync did not complete within %s; continuing to retry in background", initialSyncTimeout)
}

// initialSyncTimeout bounds how long Start() waits for the first full cache
// sync before giving up and handing off to the background retry loop.
// syncRetryInterval paces the retry loop afterward — generous enough not to
// hammer the API server the way the resync-period comment above warns against.
const (
	initialSyncTimeout = 2 * time.Minute
	syncRetryInterval  = 30 * time.Second
)

// waitForSync blocks until every registered informer cache reports synced,
// the manager is stopped, or timeout elapses — whichever happens first.
// Returns true only if every cache reported synced before giving up.
func (im *InformerManager) waitForSync(timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	giveUp := make(chan struct{})
	done := make(chan struct{})
	defer close(done)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Default().Error("panic in informer cache-sync timeout goroutine", "error", r)
				// giveUp gates WaitForCacheSync below — if this goroutine
				// panics before the normal close(giveUp) below, that call
				// blocks forever with nothing else able to unblock it.
				close(giveUp)
			}
		}()
		select {
		case <-im.stopCh:
		case <-timer.C:
		case <-done:
			return
		}
		close(giveUp)
	}()

	syncMap := im.factory.WaitForCacheSync(giveUp)
	allSynced := true
	for informerType, ok := range syncMap {
		if !ok {
			allSynced = false
			log.Printf("informer cache sync: %s did not sync within the timeout (commonly RBAC — this service account may lack list/watch on that resource; the cache-first perf path stays permanently disabled for this cluster until it does)", informerType)
		}
	}
	return allSynced
}

// retrySyncInBackground polls (non-blocking) for sync completion every
// syncRetryInterval until it succeeds or the manager is stopped. Runs only
// after the initial bounded wait in Start() gave up without every cache
// having synced.
func (im *InformerManager) retrySyncInBackground() {
	ticker := time.NewTicker(syncRetryInterval)
	defer ticker.Stop()

	alreadyClosed := make(chan struct{})
	close(alreadyClosed)

	// Diagnostic logging of which informer(s) are stuck (e.g. permanently
	// RBAC-blocked) fires every 10th tick (~5min) instead of every tick, so a
	// cluster that never finishes syncing doesn't spam the log forever.
	tick := 0
	const diagnosticEvery = 10

	for {
		select {
		case <-im.stopCh:
			return
		case <-ticker.C:
			tick++
			// Non-blocking poll: an already-closed stop channel makes
			// WaitForCacheSync return immediately with current state.
			syncMap := im.factory.WaitForCacheSync(alreadyClosed)
			allSynced := true
			for informerType, ok := range syncMap {
				if !ok {
					allSynced = false
					if tick%diagnosticEvery == 0 {
						log.Printf("informer cache sync: %s still not synced after retrying in the background — commonly RBAC; the cache-first perf path stays disabled for this cluster until it does", informerType)
					}
				}
			}
			if allSynced {
				im.synced.Store(true)
				log.Printf("informer cache finished initial sync after retrying in the background")
				return
			}
		}
	}
}

// HasSynced returns true after all informer caches have completed their
// initial list+watch sync. Before this returns true, ListFromCache will
// return (nil, false) to force a direct API call.
func (im *InformerManager) HasSynced() bool {
	return im.synced.Load()
}

// resourceKindToStoreKey maps the lowercase-plural resource type used in REST URLs
// to the PascalCase kind used as the informer store key.
var resourceKindToStoreKey = map[string]string{
	"pods":                     "Pod",
	"services":                 "Service",
	"configmaps":               "ConfigMap",
	"secrets":                  "Secret",
	"nodes":                    "Node",
	"namespaces":               "Namespace",
	"persistentvolumes":        "PersistentVolume",
	"persistentvolumeclaims":   "PersistentVolumeClaim",
	"serviceaccounts":          "ServiceAccount",
	"endpoints":                "Endpoints",
	"events":                   "Event",
	"deployments":              "Deployment",
	"replicasets":              "ReplicaSet",
	"statefulsets":             "StatefulSet",
	"daemonsets":               "DaemonSet",
	"jobs":                     "Job",
	"cronjobs":                 "CronJob",
	"ingresses":                "Ingress",
	"ingressclasses":           "IngressClass",
	"networkpolicies":          "NetworkPolicy",
	"roles":                    "Role",
	"rolebindings":             "RoleBinding",
	"clusterroles":             "ClusterRole",
	"clusterrolebindings":      "ClusterRoleBinding",
	"storageclasses":           "StorageClass",
	"horizontalpodautoscalers": "HorizontalPodAutoscaler",
	"poddisruptionbudgets":     "PodDisruptionBudget",
}

// ListFromCache reads resources from the in-memory informer cache.
// Returns (list, true) on cache hit or (nil, false) when the cache is
// unavailable or the resource type is not tracked by informers.
//
// This is the Lens/Headlamp model: informers maintain a live mirror of
// the cluster state via Watch; reads are served from local memory in <1ms.
// The caller should fall back to a direct K8s API call on cache miss.
//
// Supports optional namespace filtering and basic limit/offset pagination.
// Label selectors and field selectors are NOT supported — cache miss.
func (im *InformerManager) ListFromCache(resourceType, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, bool) {
	// Cannot serve from cache if informers haven't synced yet
	if !im.HasSynced() {
		return nil, false
	}

	// Label/field selectors require server-side filtering — cache miss
	if opts.LabelSelector != "" || opts.FieldSelector != "" {
		return nil, false
	}

	// Continue tokens are K8s API server state — not applicable to local cache
	if opts.Continue != "" {
		return nil, false
	}

	// Map resource type to store key
	storeKey, ok := resourceKindToStoreKey[strings.ToLower(resourceType)]
	if !ok {
		return nil, false
	}

	store := im.stores[storeKey]
	if store == nil {
		return nil, false
	}

	// Read all items from the informer store (lock-free, O(n))
	items := store.List()
	result := &unstructured.UnstructuredList{}
	skipped := 0

	for _, item := range items {
		// Convert runtime.Object to unstructured
		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(item)
		if err != nil {
			skipped++
			continue
		}
		u := unstructured.Unstructured{Object: obj}

		// Namespace filter
		if namespace != "" && u.GetNamespace() != namespace {
			continue
		}

		result.Items = append(result.Items, u)
	}
	if skipped > 0 {
		log.Printf("ListFromCache: skipped %d/%d items (%s) due to conversion errors", skipped, len(items), resourceType)
	}

	// Apply limit if specified
	if opts.Limit > 0 && int64(len(result.Items)) > opts.Limit {
		result.Items = result.Items[:opts.Limit]
	}

	return result, true
}

// CacheListResult holds paginated cache results with metadata.
type CacheListResult struct {
	Items []unstructured.Unstructured
	Total int64 // total matching items before pagination
}

// ListFromCacheWithPagination serves resources from the in-memory informer cache
// with server-side search, sort, and offset-based pagination.
// - search: case-insensitive substring match on resource name or namespace
// - sortBy: "name", "namespace", "creationTimestamp", "status" (default: "name")
// - sortOrder: "asc" (default), "desc"
// - offset: skip first N items (for page navigation)
// - limit: max items to return (0 = all)
func (im *InformerManager) ListFromCacheWithPagination(resourceType, namespace string, search string, sortBy string, sortOrder string, offset int, limit int) (*CacheListResult, bool) {
	// Cannot serve from cache if informers haven't synced yet
	if !im.HasSynced() {
		return nil, false
	}

	// Map resource type to store key
	storeKey, ok := resourceKindToStoreKey[strings.ToLower(resourceType)]
	if !ok {
		return nil, false
	}

	store := im.stores[storeKey]
	if store == nil {
		return nil, false
	}

	// Read all items from the informer store (lock-free, O(n))
	rawItems := store.List()

	normalizedSortBy := strings.ToLower(sortBy)
	if normalizedSortBy == "" {
		normalizedSortBy = "name"
	}
	descending := strings.ToLower(sortOrder) == "desc"

	// PERF (10K campaign P1): name/namespace/creationTimestamp — the default
	// sort and the two most common explicit ones — are available directly on
	// every runtime.Object via metav1.Object, with no reflection-based
	// ToUnstructured conversion required. Filter+sort+paginate on the typed
	// objects first, and convert only the page actually being returned, so a
	// request's cost scales with `limit`, not with total cluster object
	// count. Exotic computed sort keys (status.phase, restarts, etc.) still
	// need resource-specific nested-field access and fall back to the
	// original full-conversion path unchanged below.
	switch normalizedSortBy {
	case "name", "namespace", "creationtimestamp":
		return listFromCacheTypedFastPath(rawItems, namespace, search, normalizedSortBy, descending, offset, limit)
	}

	filtered := make([]unstructured.Unstructured, 0, len(rawItems))

	searchLower := strings.ToLower(search)

	for _, item := range rawItems {
		// Convert runtime.Object to unstructured
		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(item)
		if err != nil {
			continue
		}
		u := unstructured.Unstructured{Object: obj}

		// Namespace filter
		if namespace != "" && u.GetNamespace() != namespace {
			continue
		}

		// Search filter: case-insensitive substring on name or namespace
		if searchLower != "" {
			nameLower := strings.ToLower(u.GetName())
			nsLower := strings.ToLower(u.GetNamespace())
			if !strings.Contains(nameLower, searchLower) && !strings.Contains(nsLower, searchLower) {
				continue
			}
		}

		filtered = append(filtered, u)
	}

	// Cross-cutting tie-break: equal primary keys fall through to (namespace, name)
	// so the user-visible ordering is deterministic across reloads.
	tieBreak := func(i, j int) bool {
		ni, nj := filtered[i].GetNamespace(), filtered[j].GetNamespace()
		if ni != nj {
			return ni < nj
		}
		return filtered[i].GetName() < filtered[j].GetName()
	}
	getNStr := func(idx int, path ...string) string {
		v, _, _ := unstructured.NestedString(filtered[idx].Object, path...)
		return v
	}
	getNInt := func(idx int, path ...string) int64 {
		if v, found, _ := unstructured.NestedInt64(filtered[idx].Object, path...); found {
			return v
		}
		if v, found, _ := unstructured.NestedFloat64(filtered[idx].Object, path...); found {
			return int64(v)
		}
		return 0
	}
	getRestarts := func(idx int) int64 {
		stats, found, err := unstructured.NestedSlice(filtered[idx].Object, "status", "containerStatuses")
		if err != nil || !found {
			return 0
		}
		var total int64
		for _, s := range stats {
			m, ok := s.(map[string]interface{})
			if !ok {
				continue
			}
			if rc, found, _ := unstructured.NestedInt64(m, "restartCount"); found {
				total += rc
			} else if rcF, found, _ := unstructured.NestedFloat64(m, "restartCount"); found {
				total += int64(rcF)
			}
		}
		return total
	}

	sort.Slice(filtered, func(i, j int) bool {
		// String comparator with tie-break.
		strLess := func(a, b string) bool {
			if a == b {
				return tieBreak(i, j)
			}
			return a < b
		}
		// Int comparator with tie-break.
		intLess := func(a, b int64) bool {
			if a == b {
				return tieBreak(i, j)
			}
			return a < b
		}
		var less bool
		switch normalizedSortBy {
		case "status", "status.phase":
			a, b := getNStr(i, "status", "phase"), getNStr(j, "status", "phase")
			if a == "" {
				a = filtered[i].GetName()
			}
			if b == "" {
				b = filtered[j].GetName()
			}
			less = strLess(a, b)
		case "status.podip":
			less = strLess(getNStr(i, "status", "podIP"), getNStr(j, "status", "podIP"))
		case "spec.nodename":
			less = strLess(getNStr(i, "spec", "nodeName"), getNStr(j, "spec", "nodeName"))
		case "restarts":
			less = intLess(getRestarts(i), getRestarts(j))
		case "spec.replicas":
			less = intLess(getNInt(i, "spec", "replicas"), getNInt(j, "spec", "replicas"))
		case "status.replicas":
			less = intLess(getNInt(i, "status", "replicas"), getNInt(j, "status", "replicas"))
		case "status.readyreplicas":
			less = intLess(getNInt(i, "status", "readyReplicas"), getNInt(j, "status", "readyReplicas"))
		default: // "name"
			less = strLess(filtered[i].GetName(), filtered[j].GetName())
		}
		if descending {
			return !less
		}
		return less
	})

	// Record total after filtering and sorting, before pagination
	total := int64(len(filtered))

	// Apply offset
	if offset > 0 {
		if offset >= len(filtered) {
			return &CacheListResult{Items: []unstructured.Unstructured{}, Total: total}, true
		}
		filtered = filtered[offset:]
	}

	// Apply limit
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return &CacheListResult{Items: filtered, Total: total}, true
}

// typedEntry carries the metadata needed to filter and sort a cached object
// without first paying for a full ToUnstructured conversion.
type typedEntry struct {
	obj       interface{}
	name      string
	namespace string
	created   time.Time
}

// listFromCacheTypedFastPath implements the name/namespace/creationTimestamp
// sort paths (see PERF comment above the call site) by filtering and sorting
// on typed-object metadata, then converting only the final requested page to
// unstructured.Unstructured. Output ordering and tie-break semantics are
// identical to the generic unstructured path for these three sort keys.
func listFromCacheTypedFastPath(rawItems []interface{}, namespace, search, sortBy string, descending bool, offset, limit int) (*CacheListResult, bool) {
	entries := make([]typedEntry, 0, len(rawItems))
	searchLower := strings.ToLower(search)

	for _, item := range rawItems {
		accessor, err := meta.Accessor(item)
		if err != nil {
			continue
		}

		ns := accessor.GetNamespace()
		if namespace != "" && ns != namespace {
			continue
		}

		name := accessor.GetName()
		if searchLower != "" {
			if !strings.Contains(strings.ToLower(name), searchLower) && !strings.Contains(strings.ToLower(ns), searchLower) {
				continue
			}
		}

		entries = append(entries, typedEntry{
			obj:       item,
			name:      name,
			namespace: ns,
			created:   accessor.GetCreationTimestamp().Time,
		})
	}

	tieBreak := func(i, j int) bool {
		if entries[i].namespace != entries[j].namespace {
			return entries[i].namespace < entries[j].namespace
		}
		return entries[i].name < entries[j].name
	}

	sort.Slice(entries, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "namespace":
			if entries[i].namespace == entries[j].namespace {
				less = tieBreak(i, j)
			} else {
				less = entries[i].namespace < entries[j].namespace
			}
		case "creationtimestamp":
			if entries[i].created.Equal(entries[j].created) {
				less = tieBreak(i, j)
			} else {
				less = entries[i].created.Before(entries[j].created)
			}
		default: // "name"
			if entries[i].name == entries[j].name {
				less = tieBreak(i, j)
			} else {
				less = entries[i].name < entries[j].name
			}
		}
		if descending {
			return !less
		}
		return less
	})

	total := int64(len(entries))

	if offset > 0 {
		if offset >= len(entries) {
			return &CacheListResult{Items: []unstructured.Unstructured{}, Total: total}, true
		}
		entries = entries[offset:]
	}
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	// Convert only the page actually being returned. A ToUnstructured failure
	// here (vanishingly rare for objects already deserialized by the
	// informer) silently shrinks this page below `limit` rather than
	// adjusting `total` to exclude it, unlike the slow path below which
	// filters unconvertible items out before counting. Accepted trade-off:
	// re-converting every item up front to pre-validate would reintroduce
	// the O(N) cost this fast path exists to remove.
	items := make([]unstructured.Unstructured, 0, len(entries))
	for _, e := range entries {
		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(e.obj)
		if err != nil {
			continue
		}
		items = append(items, unstructured.Unstructured{Object: obj})
	}

	return &CacheListResult{Items: items, Total: total}, true
}

// Stop stops all informers
func (im *InformerManager) Stop() {
	if im.stopCh != nil {
		close(im.stopCh)
	}
}

// GetStore returns the store for a resource type
func (im *InformerManager) GetStore(resourceType string) cache.Store {
	return im.stores[resourceType]
}

// setupPodInformer sets up Pod informer
func (im *InformerManager) setupPodInformer() {
	informer := im.factory.Core().V1().Pods().Informer()
	im.stores["Pod"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Pod"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["Pod"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Pod"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupServiceInformer sets up Service informer
func (im *InformerManager) setupServiceInformer() {
	informer := im.factory.Core().V1().Services().Informer()
	im.stores["Service"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Service"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["Service"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Service"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupDeploymentInformer sets up Deployment informer
func (im *InformerManager) setupDeploymentInformer() {
	informer := im.factory.Apps().V1().Deployments().Informer()
	im.stores["Deployment"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Deployment"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["Deployment"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Deployment"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupReplicaSetInformer sets up ReplicaSet informer
func (im *InformerManager) setupReplicaSetInformer() {
	informer := im.factory.Apps().V1().ReplicaSets().Informer()
	im.stores["ReplicaSet"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["ReplicaSet"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["ReplicaSet"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["ReplicaSet"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupStatefulSetInformer sets up StatefulSet informer
func (im *InformerManager) setupStatefulSetInformer() {
	informer := im.factory.Apps().V1().StatefulSets().Informer()
	im.stores["StatefulSet"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["StatefulSet"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["StatefulSet"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["StatefulSet"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupDaemonSetInformer sets up DaemonSet informer
func (im *InformerManager) setupDaemonSetInformer() {
	informer := im.factory.Apps().V1().DaemonSets().Informer()
	im.stores["DaemonSet"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["DaemonSet"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["DaemonSet"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["DaemonSet"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupJobInformer sets up Job informer
func (im *InformerManager) setupJobInformer() {
	informer := im.factory.Batch().V1().Jobs().Informer()
	im.stores["Job"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Job"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["Job"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Job"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupCronJobInformer sets up CronJob informer
func (im *InformerManager) setupCronJobInformer() {
	informer := im.factory.Batch().V1().CronJobs().Informer()
	im.stores["CronJob"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["CronJob"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["CronJob"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["CronJob"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupConfigMapInformer sets up ConfigMap informer
func (im *InformerManager) setupConfigMapInformer() {
	informer := im.factory.Core().V1().ConfigMaps().Informer()
	im.stores["ConfigMap"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["ConfigMap"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["ConfigMap"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["ConfigMap"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupSecretInformer sets up Secret informer
func (im *InformerManager) setupSecretInformer() {
	informer := im.factory.Core().V1().Secrets().Informer()
	im.stores["Secret"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Secret"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["Secret"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Secret"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupNodeInformer sets up Node informer
func (im *InformerManager) setupNodeInformer() {
	informer := im.factory.Core().V1().Nodes().Informer()
	im.stores["Node"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Node"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["Node"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Node"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupNamespaceInformer sets up Namespace informer
func (im *InformerManager) setupNamespaceInformer() {
	informer := im.factory.Core().V1().Namespaces().Informer()
	im.stores["Namespace"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Namespace"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["Namespace"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Namespace"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupPersistentVolumeInformer sets up PersistentVolume informer
func (im *InformerManager) setupPersistentVolumeInformer() {
	informer := im.factory.Core().V1().PersistentVolumes().Informer()
	im.stores["PersistentVolume"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["PersistentVolume"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["PersistentVolume"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["PersistentVolume"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupPersistentVolumeClaimInformer sets up PersistentVolumeClaim informer
func (im *InformerManager) setupPersistentVolumeClaimInformer() {
	informer := im.factory.Core().V1().PersistentVolumeClaims().Informer()
	im.stores["PersistentVolumeClaim"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["PersistentVolumeClaim"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["PersistentVolumeClaim"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["PersistentVolumeClaim"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupServiceAccountInformer sets up ServiceAccount informer
func (im *InformerManager) setupServiceAccountInformer() {
	informer := im.factory.Core().V1().ServiceAccounts().Informer()
	im.stores["ServiceAccount"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["ServiceAccount"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["ServiceAccount"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["ServiceAccount"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupEndpointsInformer sets up Endpoints informer
func (im *InformerManager) setupEndpointsInformer() {
	informer := im.factory.Core().V1().Endpoints().Informer()
	im.stores["Endpoints"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Endpoints"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["Endpoints"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Endpoints"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupEventInformer sets up Event informer
func (im *InformerManager) setupEventInformer() {
	informer := im.factory.Core().V1().Events().Informer()
	im.stores["Event"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Event"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["Event"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Event"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupIngressInformer sets up Ingress informer
func (im *InformerManager) setupIngressInformer() {
	informer := im.factory.Networking().V1().Ingresses().Informer()
	im.stores["Ingress"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Ingress"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["Ingress"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Ingress"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupIngressClassInformer sets up IngressClass informer
func (im *InformerManager) setupIngressClassInformer() {
	informer := im.factory.Networking().V1().IngressClasses().Informer()
	im.stores["IngressClass"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["IngressClass"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["IngressClass"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["IngressClass"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupNetworkPolicyInformer sets up NetworkPolicy informer
func (im *InformerManager) setupNetworkPolicyInformer() {
	informer := im.factory.Networking().V1().NetworkPolicies().Informer()
	im.stores["NetworkPolicy"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["NetworkPolicy"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["NetworkPolicy"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["NetworkPolicy"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupRoleInformer sets up Role informer
func (im *InformerManager) setupRoleInformer() {
	informer := im.factory.Rbac().V1().Roles().Informer()
	im.stores["Role"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Role"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["Role"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["Role"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupRoleBindingInformer sets up RoleBinding informer
func (im *InformerManager) setupRoleBindingInformer() {
	informer := im.factory.Rbac().V1().RoleBindings().Informer()
	im.stores["RoleBinding"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["RoleBinding"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["RoleBinding"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["RoleBinding"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupClusterRoleInformer sets up ClusterRole informer
func (im *InformerManager) setupClusterRoleInformer() {
	informer := im.factory.Rbac().V1().ClusterRoles().Informer()
	im.stores["ClusterRole"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["ClusterRole"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["ClusterRole"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["ClusterRole"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupClusterRoleBindingInformer sets up ClusterRoleBinding informer
func (im *InformerManager) setupClusterRoleBindingInformer() {
	informer := im.factory.Rbac().V1().ClusterRoleBindings().Informer()
	im.stores["ClusterRoleBinding"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["ClusterRoleBinding"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["ClusterRoleBinding"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["ClusterRoleBinding"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupStorageClassInformer sets up StorageClass informer
func (im *InformerManager) setupStorageClassInformer() {
	informer := im.factory.Storage().V1().StorageClasses().Informer()
	im.stores["StorageClass"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["StorageClass"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["StorageClass"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["StorageClass"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupHorizontalPodAutoscalerInformer sets up HorizontalPodAutoscaler informer
func (im *InformerManager) setupHorizontalPodAutoscalerInformer() {
	informer := im.factory.Autoscaling().V2().HorizontalPodAutoscalers().Informer()
	im.stores["HorizontalPodAutoscaler"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["HorizontalPodAutoscaler"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["HorizontalPodAutoscaler"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["HorizontalPodAutoscaler"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}

// setupPodDisruptionBudgetInformer sets up PodDisruptionBudget informer
func (im *InformerManager) setupPodDisruptionBudgetInformer() {
	informer := im.factory.Policy().V1().PodDisruptionBudgets().Informer()
	im.stores["PodDisruptionBudget"] = informer.GetStore()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if handler, ok := im.handlers["PodDisruptionBudget"]; ok {
				handler("ADDED", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if handler, ok := im.handlers["PodDisruptionBudget"]; ok {
				handler("MODIFIED", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if handler, ok := im.handlers["PodDisruptionBudget"]; ok {
				handler("DELETED", obj)
			}
		},
	})
}
