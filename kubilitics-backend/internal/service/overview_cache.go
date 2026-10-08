package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/kubilitics/kubilitics-backend/internal/healthscore"
	"github.com/kubilitics/kubilitics-backend/internal/k8s"
	"github.com/kubilitics/kubilitics-backend/internal/models"
)

// maxListenersPerCluster is the upper bound on concurrent overview stream
// subscribers for a single cluster. Prevents unbounded memory growth from
// leaked or malicious connections.
const maxListenersPerCluster = 500

// clusterEntry holds all per-cluster state behind its own lock. Splitting
// this out of one cluster-wide sync.RWMutex (Theme 2 #9) means an event on
// cluster A's Pod informer no longer blocks a concurrent event on cluster
// B's — lock contention no longer grows with the number of connected
// clusters. OverviewCache's own mu guards only membership in the clusters
// map (insert/lookup/delete), never the per-cluster fields below.
type clusterEntry struct {
	mu       sync.RWMutex
	overview *models.ClusterOverview
	informer *k8s.InformerManager
	stopCh   chan struct{}
	listeners map[chan *models.ClusterOverview]struct{}
	// podPhases tracks per-pod phase for O(1) incremental status updates.
	// Key: podUID
	podPhases map[string]corev1.PodPhase
	// podMetrics tracks the last-known restart/crash-loop/OOM snapshot per
	// pod so MODIFIED/DELETED events can adjust the aggregate counters by
	// the delta instead of rescanning every pod in the cluster (see
	// updatePodStatus — this is what makes that O(1) instead of O(n)).
	// Key: podUID
	podMetrics map[string]podMetricsSnapshot
}

// podMetricsSnapshot is the subset of a pod's restart/crash-loop/OOM state
// that feeds OverviewCache's aggregate PodStatus counters.
type podMetricsSnapshot struct {
	restarts  int
	crashLoop bool
	oomKilled bool
}

// OverviewCache manages real-time dashboard data for clusters using Informers.
type OverviewCache struct {
	mu       sync.RWMutex
	clusters map[string]*clusterEntry
}

func NewOverviewCache() *OverviewCache {
	return &OverviewCache{
		clusters: make(map[string]*clusterEntry),
	}
}

// getEntry looks up a cluster's entry under the top-level lock. The lock is
// held only for the map read itself — all subsequent work happens under the
// returned entry's own lock, so this never serializes work across clusters.
func (c *OverviewCache) getEntry(clusterID string) (*clusterEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.clusters[clusterID]
	return e, ok
}

// getOrCreateEntry returns the cluster's entry, creating a bare one (with
// only its listeners map initialized) if none exists yet. This is what lets
// Subscribe register a listener before StartClusterCache has run for that
// cluster — the same ordering the previous separate c.listeners map allowed.
func (c *OverviewCache) getOrCreateEntry(clusterID string) *clusterEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.clusters[clusterID]
	if !ok {
		e = &clusterEntry{listeners: make(map[chan *models.ClusterOverview]struct{})}
		c.clusters[clusterID] = e
	}
	return e
}

// podCrashOOMFlags mirrors the exact per-pod classification previously done
// by the full-rescan recalculatePodConditions: iterate containers in order,
// and the FIRST container matching either condition decides the pod's
// classification (a pod contributes to at most one of crashLoop/oomKilled).
func podCrashOOMFlags(pod *corev1.Pod) (crashLoop bool, oomKilled bool) {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
			return true, false
		}
		if cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.Reason == "OOMKilled" {
			return false, true
		}
	}
	return false, false
}

func podRestartCount(pod *corev1.Pod) int {
	total := 0
	for _, cs := range pod.Status.ContainerStatuses {
		total += int(cs.RestartCount)
	}
	for _, cs := range pod.Status.InitContainerStatuses {
		total += int(cs.RestartCount)
	}
	return total
}

// GetOverview returns the cached overview for a cluster.
func (c *OverviewCache) GetOverview(clusterID string) (*models.ClusterOverview, bool) {
	entry, ok := c.getEntry(clusterID)
	if !ok {
		return nil, false
	}
	entry.mu.RLock()
	defer entry.mu.RUnlock()
	if entry.overview == nil {
		return nil, false
	}
	return entry.overview, true
}

// StartClusterCache initializes and starts informers for a cluster.
func (c *OverviewCache) StartClusterCache(ctx context.Context, clusterID string, client *k8s.Client) error {
	c.mu.Lock()
	entry, exists := c.clusters[clusterID]
	if exists {
		entry.mu.RLock()
		alreadyRunning := entry.informer != nil
		entry.mu.RUnlock()
		if alreadyRunning {
			c.mu.Unlock()
			return nil
		}
	} else {
		entry = &clusterEntry{listeners: make(map[chan *models.ClusterOverview]struct{})}
		c.clusters[clusterID] = entry
	}
	c.mu.Unlock()

	im := k8s.NewInformerManager(client)

	entry.mu.Lock()
	entry.informer = im
	entry.overview = &models.ClusterOverview{
		Health: models.OverviewHealth{
			Score:     100,
			Grade:     "A",
			Status:    "healthy",
			Breakdown: map[string]int{},
			Insight:   "Cluster is operating normally.",
		},
		Counts:    models.OverviewCounts{},
		PodStatus: models.OverviewPodStatus{},
		Alerts:    models.OverviewAlerts{Top3: []models.OverviewAlert{}},
	}
	entry.podPhases = make(map[string]corev1.PodPhase)
	entry.podMetrics = make(map[string]podMetricsSnapshot)
	entry.mu.Unlock()

	// Register handlers for real-time updates
	im.RegisterHandler("Pod", func(eventType string, obj interface{}) {
		c.updatePodStatus(clusterID, eventType, obj)
		c.notifyStream(clusterID)
	})
	im.RegisterHandler("Node", func(eventType string, obj interface{}) {
		c.updateNodeCount(clusterID, eventType, obj)
		c.notifyStream(clusterID)
	})
	im.RegisterHandler("Namespace", func(eventType string, obj interface{}) {
		c.updateNamespaceCount(clusterID, eventType, obj)
		c.notifyStream(clusterID)
	})
	im.RegisterHandler("Deployment", func(eventType string, obj interface{}) {
		c.updateDeploymentCount(clusterID, eventType, obj)
		c.notifyStream(clusterID)
	})
	im.RegisterHandler("DaemonSet", func(eventType string, obj interface{}) {
		c.updateDaemonSetCount(clusterID, eventType, obj)
		c.notifyStream(clusterID)
	})
	im.RegisterHandler("StatefulSet", func(eventType string, obj interface{}) {
		c.updateStatefulSetCount(clusterID, eventType, obj)
		c.notifyStream(clusterID)
	})
	im.RegisterHandler("Event", func(eventType string, obj interface{}) {
		c.updateAlerts(clusterID, eventType, obj)
		c.notifyStream(clusterID)
	})

	// Start Informers in background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Default().Error("panic in informer startup goroutine", "cluster", clusterID, "error", r)
			}
		}()
		if err := im.Start(ctx); err != nil {
			fmt.Printf("Error starting informers for cluster %s: %v\n", clusterID, err)
		}
	}()

	// COUNTS-1: periodic self-heal. updatePodStatus's incremental tracking is
	// O(1) per event (needed — recomputing the full phase breakdown from the
	// store on every single event would reintroduce the O(n)-per-event cost
	// this design deliberately avoided), but any incremental counter can in
	// principle drift from the canonical informer store through event-delivery
	// edge cases beyond the specific tombstone case fixed above (e.g. a missed
	// event during a reflector relist). Reconciling from the store periodically
	// — at the same cadence as the informer factory's own resync period
	// (podReconcileInterval, matching NewInformerManager's 5*time.Minute) —
	// bounds any such drift without paying a per-event cost.
	stopCh := make(chan struct{})
	entry.mu.Lock()
	entry.stopCh = stopCh
	entry.mu.Unlock()
	go c.runPodCountReconciliation(clusterID, stopCh)

	return nil
}

// podReconcileInterval matches the informer factory's own resync period
// (see NewInformerManager) so reconciliation never runs more often than the
// underlying cache itself is refreshed.
const podReconcileInterval = 5 * time.Minute

// runPodCountReconciliation periodically recomputes pod counts/phases from
// the canonical informer store, self-healing any drift in the incremental
// counter updatePodStatus maintains. Exits when stopCh closes (cluster
// removed/cache stopped).
func (c *OverviewCache) runPodCountReconciliation(clusterID string, stopCh <-chan struct{}) {
	ticker := time.NewTicker(podReconcileInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			c.safeReconcilePodCountsFromStore(clusterID)
		}
	}
}

// safeReconcilePodCountsFromStore wraps reconcilePodCountsFromStore with
// panic recovery per-tick so a single bad reconciliation (e.g. an unexpected
// store item shape) degrades to "skip this tick, log it, try again next
// interval" instead of crashing the whole backend process
// (docs/ai/ARCHITECTURE.md: every goroutine needs this).
func (c *OverviewCache) safeReconcilePodCountsFromStore(clusterID string) {
	defer func() {
		if r := recover(); r != nil {
			slog.Default().Error("panic in pod-count reconciliation tick — will retry next interval", "cluster_id", clusterID, "error", r)
		}
	}()
	c.reconcilePodCountsFromStore(clusterID)
}

// reconcilePodCountsFromStore rebuilds Counts.Pods, the Running/Pending/
// Succeeded/Failed breakdown, and the podPhases/podMetrics tracking maps
// directly from the informer's Pod store — the canonical source of truth —
// overwriting whatever the incremental counters currently say. A no-op if
// the cluster's cache has since been stopped.
func (c *OverviewCache) reconcilePodCountsFromStore(clusterID string) {
	entry, ok := c.getEntry(clusterID)
	if !ok {
		return
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.overview == nil || entry.informer == nil {
		return
	}
	store := entry.informer.GetStore("Pod")
	if store == nil {
		return
	}

	phases := make(map[string]corev1.PodPhase)
	metrics := make(map[string]podMetricsSnapshot)
	var ps models.OverviewPodStatus
	totalRestarts := 0
	count := 0
	for _, obj := range store.List() {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			continue
		}
		count++
		uid := string(pod.UID)
		phases[uid] = pod.Status.Phase
		incrementPhaseCounter(&ps, pod.Status.Phase)

		crashLoop, oomKilled := podCrashOOMFlags(pod)
		snap := podMetricsSnapshot{restarts: podRestartCount(pod), crashLoop: crashLoop, oomKilled: oomKilled}
		metrics[uid] = snap
		totalRestarts += snap.restarts
		if crashLoop {
			ps.CrashLoopBackOff++
		}
		if oomKilled {
			ps.OOMKilled++
		}
	}
	ps.TotalRestarts = totalRestarts

	ov := entry.overview
	ov.Counts.Pods = count
	ov.PodStatus.Running = ps.Running
	ov.PodStatus.Pending = ps.Pending
	ov.PodStatus.Succeeded = ps.Succeeded
	ov.PodStatus.Failed = ps.Failed
	ov.PodStatus.TotalRestarts = ps.TotalRestarts
	ov.PodStatus.CrashLoopBackOff = ps.CrashLoopBackOff
	ov.PodStatus.OOMKilled = ps.OOMKilled
	entry.podPhases = phases
	entry.podMetrics = metrics
}

// GetInformerManager returns the InformerManager for a cluster, or nil if not
// started. Used by the REST handler to serve resource lists from the informer
// cache (sub-millisecond) instead of hitting the K8s API every time.
func (c *OverviewCache) GetInformerManager(clusterID string) *k8s.InformerManager {
	entry, ok := c.getEntry(clusterID)
	if !ok {
		return nil
	}
	entry.mu.RLock()
	defer entry.mu.RUnlock()
	return entry.informer
}

// StopClusterCache stops informers for a cluster. Deliberately does not
// remove the cluster's entry (or its listeners) from the top-level map —
// a subscriber's channel from Subscribe survives a stop/restart cycle,
// matching the previous behavior where listeners lived in a separate map
// from overviews/informers.
func (c *OverviewCache) StopClusterCache(clusterID string) {
	entry, ok := c.getEntry(clusterID)
	if !ok {
		return
	}

	entry.mu.Lock()
	if entry.informer != nil {
		entry.informer.Stop()
		entry.informer = nil
	}
	entry.overview = nil
	entry.podPhases = nil
	entry.podMetrics = nil
	stopCh := entry.stopCh
	entry.stopCh = nil
	entry.mu.Unlock()

	if stopCh != nil {
		close(stopCh)
	}
}

func (c *OverviewCache) notifyStream(clusterID string) {
	entry, ok := c.getEntry(clusterID)
	if !ok {
		return
	}
	entry.mu.RLock()
	ov := entry.overview
	listeners := entry.listeners
	if ov == nil || len(listeners) == 0 {
		entry.mu.RUnlock()
		return
	}
	// Deep-copy the overview under the lock to avoid sending a shared pointer
	// that may be concurrently modified by another informer event handler.
	healthCopy := ov.Health
	// Deep-copy Breakdown map
	if ov.Health.Breakdown != nil {
		healthCopy.Breakdown = make(map[string]int, len(ov.Health.Breakdown))
		for k, v := range ov.Health.Breakdown {
			healthCopy.Breakdown[k] = v
		}
	}
	// Deep-copy Findings slice
	if len(ov.Health.Findings) > 0 {
		healthCopy.Findings = make([]models.OverviewHealthFinding, len(ov.Health.Findings))
		copy(healthCopy.Findings, ov.Health.Findings)
	}
	snapshot := &models.ClusterOverview{
		Health:    healthCopy,
		Counts:    ov.Counts,
		PodStatus: ov.PodStatus,
		Alerts: models.OverviewAlerts{
			Warnings: ov.Alerts.Warnings,
			Critical: ov.Alerts.Critical,
			Top3:     make([]models.OverviewAlert, len(ov.Alerts.Top3)),
		},
	}
	copy(snapshot.Alerts.Top3, ov.Alerts.Top3)
	if ov.Utilization != nil {
		u := *ov.Utilization
		snapshot.Utilization = &u
	}
	entry.mu.RUnlock()

	for ch := range listeners {
		select {
		case ch <- snapshot:
		default:
			log.Printf("overview cache: dropped notification for cluster %s (listener channel full)", clusterID)
		}
	}
}

// ErrTooManyListeners is returned when a cluster has reached the max listener limit.
var ErrTooManyListeners = errors.New("too many overview stream listeners for this cluster")

// Subscribe returns a channel that receives overview updates for a cluster.
// Returns ErrTooManyListeners if the per-cluster listener limit is reached.
// May be called before StartClusterCache for the same clusterID — the entry
// is created lazily and StartClusterCache reuses it rather than replacing it,
// so a listener registered early is never lost.
func (c *OverviewCache) Subscribe(clusterID string) (chan *models.ClusterOverview, func(), error) {
	ch := make(chan *models.ClusterOverview, 10)

	entry := c.getOrCreateEntry(clusterID)

	entry.mu.Lock()
	if entry.listeners == nil {
		entry.listeners = make(map[chan *models.ClusterOverview]struct{})
	}
	if len(entry.listeners) >= maxListenersPerCluster {
		entry.mu.Unlock()
		log.Printf("overview cache: listener limit reached for cluster %s (%d)", clusterID, maxListenersPerCluster)
		return nil, nil, ErrTooManyListeners
	}
	entry.listeners[ch] = struct{}{}
	entry.mu.Unlock()

	// Initial push — deep-copy to avoid sending a shared pointer (same as notifyStream).
	if ov, ok := c.GetOverview(clusterID); ok {
		healthCopy := ov.Health
		if ov.Health.Breakdown != nil {
			healthCopy.Breakdown = make(map[string]int, len(ov.Health.Breakdown))
			for k, v := range ov.Health.Breakdown {
				healthCopy.Breakdown[k] = v
			}
		}
		if len(ov.Health.Findings) > 0 {
			healthCopy.Findings = make([]models.OverviewHealthFinding, len(ov.Health.Findings))
			copy(healthCopy.Findings, ov.Health.Findings)
		}
		snapshot := &models.ClusterOverview{
			Health:    healthCopy,
			Counts:    ov.Counts,
			PodStatus: ov.PodStatus,
			Alerts: models.OverviewAlerts{
				Warnings: ov.Alerts.Warnings,
				Critical: ov.Alerts.Critical,
				Top3:     make([]models.OverviewAlert, len(ov.Alerts.Top3)),
			},
		}
		copy(snapshot.Alerts.Top3, ov.Alerts.Top3)
		if ov.Utilization != nil {
			u := *ov.Utilization
			snapshot.Utilization = &u
		}
		ch <- snapshot
	}

	unsubscribe := func() {
		entry.mu.Lock()
		defer entry.mu.Unlock()
		if _, exists := entry.listeners[ch]; exists {
			delete(entry.listeners, ch)
			close(ch)
		}
	}

	return ch, unsubscribe, nil
}

// decrementPhaseCounter decrements the counter for the given phase.
func decrementPhaseCounter(ps *models.OverviewPodStatus, phase corev1.PodPhase) {
	switch phase {
	case corev1.PodRunning:
		if ps.Running > 0 {
			ps.Running--
		}
	case corev1.PodPending:
		if ps.Pending > 0 {
			ps.Pending--
		}
	case corev1.PodSucceeded:
		if ps.Succeeded > 0 {
			ps.Succeeded--
		}
	case corev1.PodFailed, corev1.PodUnknown:
		if ps.Failed > 0 {
			ps.Failed--
		}
	}
}

// incrementPhaseCounter increments the counter for the given phase.
func incrementPhaseCounter(ps *models.OverviewPodStatus, phase corev1.PodPhase) {
	switch phase {
	case corev1.PodRunning:
		ps.Running++
	case corev1.PodPending:
		ps.Pending++
	case corev1.PodSucceeded:
		ps.Succeeded++
	case corev1.PodFailed, corev1.PodUnknown:
		ps.Failed++
	}
}

// updatePodStatus performs O(1) incremental pod status updates using per-pod phase tracking.
// Instead of re-listing all pods on every event, it tracks each pod's last known phase
// and adjusts counters incrementally.
//
// COUNTS-1 (docs/PRODUCTION-RELIABILITY-AUDIT.md): client-go's DeleteFunc can
// legitimately deliver a cache.DeletedFinalStateUnknown wrapper (not a raw
// *corev1.Pod) when a delete is inferred from a relist rather than observed
// directly on the watch. Previously the type assertion below failed silently
// for that case and returned before decrementing — the counter only ever grew.
func (c *OverviewCache) updatePodStatus(clusterID string, eventType string, obj interface{}) {
	entry, ok := c.getEntry(clusterID)
	if !ok {
		return
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.overview == nil {
		return
	}
	ov := entry.overview

	if tombstone, isTombstone := obj.(cache.DeletedFinalStateUnknown); isTombstone {
		obj = tombstone.Obj
	}

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return
	}

	phases := entry.podPhases
	if phases == nil {
		phases = make(map[string]corev1.PodPhase)
		entry.podPhases = phases
	}
	metrics := entry.podMetrics
	if metrics == nil {
		metrics = make(map[string]podMetricsSnapshot)
		entry.podMetrics = metrics
	}

	uid := string(pod.UID)
	newPhase := pod.Status.Phase
	newCrashLoop, newOOMKilled := podCrashOOMFlags(pod)
	newSnapshot := podMetricsSnapshot{
		restarts:  podRestartCount(pod),
		crashLoop: newCrashLoop,
		oomKilled: newOOMKilled,
	}

	// applyMetricsDelta adjusts the aggregate counters by the difference
	// between old and new per-pod snapshots — O(1) regardless of cluster
	// size, instead of the full-rescan recalculateTotalRestarts/
	// recalculatePodConditions this replaces.
	applyMetricsDelta := func(old, new podMetricsSnapshot) {
		ov.PodStatus.TotalRestarts += new.restarts - old.restarts
		if ov.PodStatus.TotalRestarts < 0 {
			ov.PodStatus.TotalRestarts = 0
		}
		if old.crashLoop != new.crashLoop {
			if new.crashLoop {
				ov.PodStatus.CrashLoopBackOff++
			} else {
				ov.PodStatus.CrashLoopBackOff--
				if ov.PodStatus.CrashLoopBackOff < 0 {
					ov.PodStatus.CrashLoopBackOff = 0
				}
			}
		}
		if old.oomKilled != new.oomKilled {
			if new.oomKilled {
				ov.PodStatus.OOMKilled++
			} else {
				ov.PodStatus.OOMKilled--
				if ov.PodStatus.OOMKilled < 0 {
					ov.PodStatus.OOMKilled = 0
				}
			}
		}
	}

	switch eventType {
	case "ADDED":
		if _, exists := phases[uid]; !exists {
			incrementPhaseCounter(&ov.PodStatus, newPhase)
			phases[uid] = newPhase
			ov.Counts.Pods++
			applyMetricsDelta(podMetricsSnapshot{}, newSnapshot)
			metrics[uid] = newSnapshot
		}
	case "MODIFIED":
		if oldPhase, exists := phases[uid]; exists {
			if oldPhase != newPhase {
				decrementPhaseCounter(&ov.PodStatus, oldPhase)
				incrementPhaseCounter(&ov.PodStatus, newPhase)
				phases[uid] = newPhase
			}
			applyMetricsDelta(metrics[uid], newSnapshot)
			metrics[uid] = newSnapshot
		} else {
			// Pod not tracked yet (missed ADDED event); treat as add
			incrementPhaseCounter(&ov.PodStatus, newPhase)
			phases[uid] = newPhase
			ov.Counts.Pods++
			applyMetricsDelta(podMetricsSnapshot{}, newSnapshot)
			metrics[uid] = newSnapshot
		}
	case "DELETED":
		if oldPhase, exists := phases[uid]; exists {
			decrementPhaseCounter(&ov.PodStatus, oldPhase)
			delete(phases, uid)
			ov.Counts.Pods--
			if ov.Counts.Pods < 0 {
				ov.Counts.Pods = 0
			}
			// Subtract the last-tracked snapshot, not a fresh read of obj —
			// a DeletedFinalStateUnknown tombstone's object can be stale or
			// incomplete, but our own tracked state is exactly what we added.
			applyMetricsDelta(metrics[uid], podMetricsSnapshot{})
			delete(metrics, uid)
		}
		_ = newPhase // suppress unused warning for deleted pods
	}

	c.recalculateHealthRLocked(ov)
}

func (c *OverviewCache) updateNodeCount(clusterID string, _ string, _ interface{}) {
	entry, ok := c.getEntry(clusterID)
	if !ok {
		return
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.overview == nil || entry.informer == nil {
		return
	}
	ov := entry.overview
	nodeItems := entry.informer.GetStore("Node").List()
	ov.Counts.Nodes = len(nodeItems)
	readyNodes := 0
	diskPressure := 0
	memPressure := 0
	pidPressure := 0
	for _, obj := range nodeItems {
		if node, ok := obj.(*corev1.Node); ok {
			for _, cond := range node.Status.Conditions {
				switch cond.Type {
				case corev1.NodeReady:
					if cond.Status == corev1.ConditionTrue {
						readyNodes++
					}
				case corev1.NodeDiskPressure:
					if cond.Status == corev1.ConditionTrue {
						diskPressure++
					}
				case corev1.NodeMemoryPressure:
					if cond.Status == corev1.ConditionTrue {
						memPressure++
					}
				case corev1.NodePIDPressure:
					if cond.Status == corev1.ConditionTrue {
						pidPressure++
					}
				}
			}
		}
	}
	ov.Counts.ReadyNodes = readyNodes
	ov.Counts.DiskPressureNodes = diskPressure
	ov.Counts.MemoryPressureNodes = memPressure
	ov.Counts.PIDPressureNodes = pidPressure
	c.recalculateHealthRLocked(ov)
}

func (c *OverviewCache) updateNamespaceCount(clusterID string, _ string, _ interface{}) {
	entry, ok := c.getEntry(clusterID)
	if !ok {
		return
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.overview == nil || entry.informer == nil {
		return
	}
	entry.overview.Counts.Namespaces = len(entry.informer.GetStore("Namespace").List())
}

func (c *OverviewCache) updateDeploymentCount(clusterID string, _ string, _ interface{}) {
	entry, ok := c.getEntry(clusterID)
	if !ok {
		return
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.overview == nil || entry.informer == nil {
		return
	}
	ov := entry.overview
	items := entry.informer.GetStore("Deployment").List()
	ov.Counts.Deployments = len(items)
	available, unavailable := 0, 0
	for _, obj := range items {
		if dep, ok := obj.(*appsv1.Deployment); ok {
			if dep.Status.AvailableReplicas > 0 && dep.Status.UnavailableReplicas == 0 {
				available++
			} else if dep.Status.UnavailableReplicas > 0 {
				unavailable++
			} else {
				available++ // no replicas requested or all satisfied
			}
		}
	}
	ov.Counts.DeploymentsAvailable = available
	ov.Counts.DeploymentsUnavailable = unavailable
	c.recalculateHealthRLocked(ov)
}

func (c *OverviewCache) updateDaemonSetCount(clusterID string, _ string, _ interface{}) {
	entry, ok := c.getEntry(clusterID)
	if !ok {
		return
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.overview == nil || entry.informer == nil {
		return
	}
	ov := entry.overview
	items := entry.informer.GetStore("DaemonSet").List()
	ov.Counts.DaemonSetsTotal = len(items)
	ready := 0
	for _, obj := range items {
		if ds, ok := obj.(*appsv1.DaemonSet); ok {
			if ds.Status.NumberReady == ds.Status.DesiredNumberScheduled {
				ready++
			}
		}
	}
	ov.Counts.DaemonSetsReady = ready
	c.recalculateHealthRLocked(ov)
}

func (c *OverviewCache) updateStatefulSetCount(clusterID string, _ string, _ interface{}) {
	entry, ok := c.getEntry(clusterID)
	if !ok {
		return
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.overview == nil || entry.informer == nil {
		return
	}
	ov := entry.overview
	items := entry.informer.GetStore("StatefulSet").List()
	ov.Counts.StatefulSetsTotal = len(items)
	ready := 0
	for _, obj := range items {
		if sts, ok := obj.(*appsv1.StatefulSet); ok {
			if sts.Status.ReadyReplicas == sts.Status.Replicas {
				ready++
			}
		}
	}
	ov.Counts.StatefulSetsReady = ready
	c.recalculateHealthRLocked(ov)
}

func (c *OverviewCache) updateAlerts(clusterID string, _ string, _ interface{}) {
	entry, ok := c.getEntry(clusterID)
	if !ok {
		return
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.overview == nil || entry.informer == nil {
		return
	}
	ov := entry.overview

	events := entry.informer.GetStore("Event").List()
	warnings := 0
	critical := 0
	var top3 []models.OverviewAlert

	for _, eObj := range events {
		e := eObj.(*corev1.Event)
		if e.Type == corev1.EventTypeWarning {
			warnings++
			if len(top3) < 3 {
				top3 = append(top3, models.OverviewAlert{
					Reason:    e.Reason,
					Resource:  fmt.Sprintf("%s/%s", e.InvolvedObject.Kind, e.InvolvedObject.Name),
					Namespace: e.Namespace,
				})
			}
		} else if e.Type != corev1.EventTypeNormal {
			critical++
		}
	}

	ov.Alerts.Warnings = warnings
	ov.Alerts.Critical = critical
	ov.Alerts.Top3 = top3
	c.recalculateHealthRLocked(ov)
}

// recalculateHealthRLocked builds a ClusterState from cached data and delegates
// to the enterprise healthscore.Score() engine. Must be called under the
// owning entry's write lock.
func (c *OverviewCache) recalculateHealthRLocked(ov *models.ClusterOverview) {
	state := healthscore.ClusterState{
		TotalNodes:    ov.Counts.Nodes,
		ReadyNodes:    ov.Counts.ReadyNodes,
		DiskPressure:  ov.Counts.DiskPressureNodes,
		MemPressure:   ov.Counts.MemoryPressureNodes,
		PIDPressure:   ov.Counts.PIDPressureNodes,
		PodsRunning:   ov.PodStatus.Running,
		PodsPending:   ov.PodStatus.Pending,
		PodsFailed:    ov.PodStatus.Failed,
		PodsSucceeded: ov.PodStatus.Succeeded,
		PodsCrashLoop: ov.PodStatus.CrashLoopBackOff,
		PodsOOMKilled: ov.PodStatus.OOMKilled,
		TotalRestarts: ov.PodStatus.TotalRestarts,
		WarningEvents: ov.Alerts.Warnings,
		CriticalEvents: ov.Alerts.Critical,

		DeploymentsTotal:       ov.Counts.Deployments,
		DeploymentsAvailable:   ov.Counts.DeploymentsAvailable,
		DeploymentsUnavailable: ov.Counts.DeploymentsUnavailable,
		DeploymentsProgressing: ov.Counts.Deployments - ov.Counts.DeploymentsAvailable - ov.Counts.DeploymentsUnavailable,

		DaemonSetsTotal:   ov.Counts.DaemonSetsTotal,
		DaemonSetsReady:   ov.Counts.DaemonSetsReady,
		StatefulSetsTotal: ov.Counts.StatefulSetsTotal,
		StatefulSetsReady: ov.Counts.StatefulSetsReady,
	}

	// Clamp progressing to 0 if negative (rounding)
	if state.DeploymentsProgressing < 0 {
		state.DeploymentsProgressing = 0
	}

	result := healthscore.Score(state)

	ov.Health.Score = result.Score
	ov.Health.Grade = result.Grade
	ov.Health.Status = result.Status
	ov.Health.Insight = result.Insight

	// Build breakdown map from category scores
	breakdown := make(map[string]int, len(result.Categories))
	for cat, cs := range result.Categories {
		breakdown[string(cat)] = cs.Score
	}
	ov.Health.Breakdown = breakdown

	// Top 5 findings
	findings := make([]models.OverviewHealthFinding, 0, 5)
	for i, f := range result.Findings {
		if i >= 5 {
			break
		}
		findings = append(findings, models.OverviewHealthFinding{
			Category: string(f.Category),
			Severity: int(f.Severity),
			Check:    f.Check,
			Message:  f.Message,
		})
	}
	ov.Health.Findings = findings
}
