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

// OverviewCache manages real-time dashboard data for clusters using Informers.
type OverviewCache struct {
	mu        sync.RWMutex
	overviews map[string]*models.ClusterOverview
	informers map[string]*k8s.InformerManager
	stopChs   map[string]chan struct{}
	listeners map[string]map[chan *models.ClusterOverview]struct{}
	// podPhases tracks per-pod phase for O(1) incremental status updates.
	// Key: clusterID, Value: map[podUID]corev1.PodPhase
	podPhases map[string]map[string]corev1.PodPhase
}

func NewOverviewCache() *OverviewCache {
	return &OverviewCache{
		overviews: make(map[string]*models.ClusterOverview),
		informers: make(map[string]*k8s.InformerManager),
		stopChs:   make(map[string]chan struct{}),
		listeners: make(map[string]map[chan *models.ClusterOverview]struct{}),
		podPhases: make(map[string]map[string]corev1.PodPhase),
	}
}

// GetOverview returns the cached overview for a cluster.
func (c *OverviewCache) GetOverview(clusterID string) (*models.ClusterOverview, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ov, ok := c.overviews[clusterID]
	return ov, ok
}

// StartClusterCache initializes and starts informers for a cluster.
func (c *OverviewCache) StartClusterCache(ctx context.Context, clusterID string, client *k8s.Client) error {
	c.mu.Lock()
	if _, exists := c.informers[clusterID]; exists {
		c.mu.Unlock()
		return nil // Already running
	}

	im := k8s.NewInformerManager(client)
	c.informers[clusterID] = im

	overview := &models.ClusterOverview{
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
	c.overviews[clusterID] = overview
	c.podPhases[clusterID] = make(map[string]corev1.PodPhase)
	c.mu.Unlock()

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
	c.mu.Lock()
	c.stopChs[clusterID] = stopCh
	c.mu.Unlock()
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
			c.reconcilePodCountsFromStore(clusterID)
		}
	}
}

// reconcilePodCountsFromStore rebuilds Counts.Pods, the Running/Pending/
// Succeeded/Failed breakdown, and the podPhases tracking map directly from
// the informer's Pod store — the canonical source of truth — overwriting
// whatever the incremental counters currently say. A no-op if the cluster's
// cache has since been stopped.
func (c *OverviewCache) reconcilePodCountsFromStore(clusterID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ov, ok := c.overviews[clusterID]
	if !ok {
		return
	}
	im, ok := c.informers[clusterID]
	if !ok {
		return
	}
	store := im.GetStore("Pod")
	if store == nil {
		return
	}

	phases := make(map[string]corev1.PodPhase)
	ps := models.OverviewPodStatus{
		TotalRestarts:    ov.PodStatus.TotalRestarts,
		CrashLoopBackOff: ov.PodStatus.CrashLoopBackOff,
		OOMKilled:        ov.PodStatus.OOMKilled,
	}
	count := 0
	for _, obj := range store.List() {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			continue
		}
		count++
		phases[string(pod.UID)] = pod.Status.Phase
		incrementPhaseCounter(&ps, pod.Status.Phase)
	}

	ov.Counts.Pods = count
	ov.PodStatus.Running = ps.Running
	ov.PodStatus.Pending = ps.Pending
	ov.PodStatus.Succeeded = ps.Succeeded
	ov.PodStatus.Failed = ps.Failed
	c.podPhases[clusterID] = phases
}

// GetInformerManager returns the InformerManager for a cluster, or nil if not
// started. Used by the REST handler to serve resource lists from the informer
// cache (sub-millisecond) instead of hitting the K8s API every time.
func (c *OverviewCache) GetInformerManager(clusterID string) *k8s.InformerManager {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.informers[clusterID]
}

// StopClusterCache stops informers for a cluster.
func (c *OverviewCache) StopClusterCache(clusterID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if im, exists := c.informers[clusterID]; exists {
		im.Stop()
		delete(c.informers, clusterID)
		delete(c.overviews, clusterID)
		delete(c.podPhases, clusterID)
	}
	if stopCh, exists := c.stopChs[clusterID]; exists {
		close(stopCh)
		delete(c.stopChs, clusterID)
	}
}

func (c *OverviewCache) notifyStream(clusterID string) {
	c.mu.RLock()
	ov := c.overviews[clusterID]
	listeners := c.listeners[clusterID]
	if ov == nil || len(listeners) == 0 {
		c.mu.RUnlock()
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
	c.mu.RUnlock()

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
func (c *OverviewCache) Subscribe(clusterID string) (chan *models.ClusterOverview, func(), error) {
	ch := make(chan *models.ClusterOverview, 10)

	c.mu.Lock()
	if c.listeners[clusterID] == nil {
		c.listeners[clusterID] = make(map[chan *models.ClusterOverview]struct{})
	}
	if len(c.listeners[clusterID]) >= maxListenersPerCluster {
		c.mu.Unlock()
		log.Printf("overview cache: listener limit reached for cluster %s (%d)", clusterID, maxListenersPerCluster)
		return nil, nil, ErrTooManyListeners
	}
	c.listeners[clusterID][ch] = struct{}{}
	c.mu.Unlock()

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
		c.mu.Lock()
		defer c.mu.Unlock()
		if _, exists := c.listeners[clusterID][ch]; exists {
			delete(c.listeners[clusterID], ch)
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
	c.mu.Lock()
	defer c.mu.Unlock()

	ov, ok := c.overviews[clusterID]
	if !ok {
		return
	}

	if tombstone, isTombstone := obj.(cache.DeletedFinalStateUnknown); isTombstone {
		obj = tombstone.Obj
	}

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return
	}

	phases := c.podPhases[clusterID]
	if phases == nil {
		phases = make(map[string]corev1.PodPhase)
		c.podPhases[clusterID] = phases
	}

	uid := string(pod.UID)
	newPhase := pod.Status.Phase

	// Track total restart count from container statuses
	podRestarts := 0
	for _, cs := range pod.Status.ContainerStatuses {
		podRestarts += int(cs.RestartCount)
	}
	for _, cs := range pod.Status.InitContainerStatuses {
		podRestarts += int(cs.RestartCount)
	}

	switch eventType {
	case "ADDED":
		if _, exists := phases[uid]; !exists {
			incrementPhaseCounter(&ov.PodStatus, newPhase)
			phases[uid] = newPhase
			ov.Counts.Pods++
			ov.PodStatus.TotalRestarts += podRestarts
		}
	case "MODIFIED":
		if oldPhase, exists := phases[uid]; exists {
			if oldPhase != newPhase {
				decrementPhaseCounter(&ov.PodStatus, oldPhase)
				incrementPhaseCounter(&ov.PodStatus, newPhase)
				phases[uid] = newPhase
			}
		} else {
			// Pod not tracked yet (missed ADDED event); treat as add
			incrementPhaseCounter(&ov.PodStatus, newPhase)
			phases[uid] = newPhase
			ov.Counts.Pods++
		}
		// Recalculate total restarts from all tracked pods would be expensive;
		// instead, on MODIFIED we re-list total from store for accuracy.
		c.recalculateTotalRestarts(clusterID, ov)
	case "DELETED":
		if oldPhase, exists := phases[uid]; exists {
			decrementPhaseCounter(&ov.PodStatus, oldPhase)
			delete(phases, uid)
			ov.Counts.Pods--
			if ov.Counts.Pods < 0 {
				ov.Counts.Pods = 0
			}
			ov.PodStatus.TotalRestarts -= podRestarts
			if ov.PodStatus.TotalRestarts < 0 {
				ov.PodStatus.TotalRestarts = 0
			}
		}
		_ = newPhase // suppress unused warning for deleted pods
	}

	c.recalculatePodConditions(clusterID, ov)
	c.recalculateHealthRLocked(ov)
}

func (c *OverviewCache) updateNodeCount(clusterID string, _ string, _ interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ov, ok := c.overviews[clusterID]
	if !ok {
		return
	}
	nodeItems := c.informers[clusterID].GetStore("Node").List()
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
	c.mu.Lock()
	defer c.mu.Unlock()
	ov, ok := c.overviews[clusterID]
	if !ok {
		return
	}
	ov.Counts.Namespaces = len(c.informers[clusterID].GetStore("Namespace").List())
}

func (c *OverviewCache) updateDeploymentCount(clusterID string, _ string, _ interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ov, ok := c.overviews[clusterID]
	if !ok {
		return
	}
	items := c.informers[clusterID].GetStore("Deployment").List()
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
	c.mu.Lock()
	defer c.mu.Unlock()
	ov, ok := c.overviews[clusterID]
	if !ok {
		return
	}
	items := c.informers[clusterID].GetStore("DaemonSet").List()
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
	c.mu.Lock()
	defer c.mu.Unlock()
	ov, ok := c.overviews[clusterID]
	if !ok {
		return
	}
	items := c.informers[clusterID].GetStore("StatefulSet").List()
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
	c.mu.Lock()
	defer c.mu.Unlock()
	ov, ok := c.overviews[clusterID]
	if !ok {
		return
	}

	events := c.informers[clusterID].GetStore("Event").List()
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

// recalculateTotalRestarts recomputes the total restart count from the Pod informer store.
// Called under write lock from updatePodStatus on MODIFIED events for accuracy.
func (c *OverviewCache) recalculateTotalRestarts(clusterID string, ov *models.ClusterOverview) {
	im, ok := c.informers[clusterID]
	if !ok {
		return
	}
	store := im.GetStore("Pod")
	if store == nil {
		return
	}
	total := 0
	for _, obj := range store.List() {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			continue
		}
		for _, cs := range pod.Status.ContainerStatuses {
			total += int(cs.RestartCount)
		}
		for _, cs := range pod.Status.InitContainerStatuses {
			total += int(cs.RestartCount)
		}
	}
	ov.PodStatus.TotalRestarts = total
}

// recalculatePodConditions recomputes CrashLoopBackOff and OOMKilled counts from the Pod informer store.
// Called under write lock from updatePodStatus for accuracy.
func (c *OverviewCache) recalculatePodConditions(clusterID string, ov *models.ClusterOverview) {
	im, ok := c.informers[clusterID]
	if !ok {
		return
	}
	store := im.GetStore("Pod")
	if store == nil {
		return
	}
	crashLoop := 0
	oomKilled := 0
	for _, obj := range store.List() {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			continue
		}
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
				crashLoop++
				break
			}
			if cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.Reason == "OOMKilled" {
				oomKilled++
				break
			}
		}
	}
	ov.PodStatus.CrashLoopBackOff = crashLoop
	ov.PodStatus.OOMKilled = oomKilled
}

// recalculateHealthRLocked builds a ClusterState from cached data and delegates
// to the enterprise healthscore.Score() engine. Must be called under write lock.
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
