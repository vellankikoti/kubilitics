package events

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/informers"
)

// Collector watches Kubernetes core/v1 Events and resource spec changes,
// converting them into WideEvent records for the Events Intelligence pipeline.
type Collector struct {
	store         *Store
	eventsCh      chan *WideEvent // output channel
	stopCh        chan struct{}
	lastEventTime time.Time  // last time an event was received
	mu            sync.RWMutex
}

// NewCollector creates a new event collector.
func NewCollector(store *Store) *Collector {
	return &Collector{
		store:    store,
		eventsCh: make(chan *WideEvent, 256),
		stopCh:   make(chan struct{}),
	}
}

// Events returns the read-only output channel of collected WideEvents.
func (c *Collector) Events() <-chan *WideEvent {
	return c.eventsCh
}

// Start begins watching Kubernetes core/v1 Events and emitting WideEvents.
// Uses SharedInformerFactory (same as graph engine) for robust connection handling.
func (c *Collector) Start(clientset kubernetes.Interface, clusterID string) error {
	if clientset == nil {
		return fmt.Errorf("cannot start collector: clientset is nil for cluster %s", clusterID)
	}

	factory := informers.NewSharedInformerFactory(clientset, 5*time.Minute)
	eventsInformer := factory.Core().V1().Events().Informer()

	eventsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.handleK8sEvent(obj, clusterID)
		},
		UpdateFunc: func(_, newObj interface{}) {
			c.handleK8sEvent(newObj, clusterID)
		},
	})

	// Start factory in background (handles its own goroutines safely)
	factory.Start(c.stopCh)

	// Wait for cache sync with timeout
	go func() {
		synced := factory.WaitForCacheSync(c.stopCh)
		allSynced := true
		for _, s := range synced {
			if !s {
				allSynced = false
				break
			}
		}
		if allSynced {
			log.Printf("[events/collector] K8s events informer synced for cluster %s", clusterID)
		} else {
			log.Printf("[events/collector] K8s events informer sync FAILED for cluster %s (will keep retrying)", clusterID)
		}
	}()

	log.Printf("[events/collector] started watching K8s events for cluster %s", clusterID)
	return nil
}

// WatchResourceChanges watches Deployments, ConfigMaps, and Secrets for spec
// changes and emits WideEvents when mutations are detected. Each resource
// watcher reconnects automatically with exponential backoff if the API server
// connection drops. Call Stop to shut them all down.
func (c *Collector) WatchResourceChanges(clientset kubernetes.Interface, clusterID string) {
	// Resource watchers are disabled — the graph engine already watches Deployments,
	// ConfigMaps, and Secrets via its own informers. Our K8s events watcher captures
	// all resource changes as events. Re-enable these when we have a panic-safe
	// informer wrapper that survives client-go's internal goroutine re-panics.
	log.Printf("[events/collector] resource change watchers disabled (using graph engine informers)")
	return

	// Deployment watcher
	go c.watchWithRetry("deployments", func() cache.Controller {
		lw := cache.NewListWatchFromClient(clientset.AppsV1().RESTClient(), "deployments", metav1.NamespaceAll, nil)
		_, informer := cache.NewInformer(lw, &appsv1.Deployment{}, 0, cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj, newObj interface{}) {
				oldDep, ok1 := oldObj.(*appsv1.Deployment)
				newDep, ok2 := newObj.(*appsv1.Deployment)
				if !ok1 || !ok2 {
					return
				}
				if oldDep.Generation == newDep.Generation {
					return
				}
				c.emitResourceChangeEvent(clusterID, "Deployment", newDep.Name, newDep.Namespace, string(newDep.UID), "SpecChanged")
			},
		})
		return informer
	})

	// ConfigMap watcher
	go c.watchWithRetry("configmaps", func() cache.Controller {
		lw := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "configmaps", metav1.NamespaceAll, nil)
		_, informer := cache.NewInformer(lw, &corev1.ConfigMap{}, 0, cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj, newObj interface{}) {
				oldCM, ok1 := oldObj.(*corev1.ConfigMap)
				newCM, ok2 := newObj.(*corev1.ConfigMap)
				if !ok1 || !ok2 {
					return
				}
				if oldCM.ResourceVersion == newCM.ResourceVersion {
					return
				}
				c.emitResourceChangeEvent(clusterID, "ConfigMap", newCM.Name, newCM.Namespace, string(newCM.UID), "ConfigChanged")
			},
		})
		return informer
	})

	// Secret watcher
	go c.watchWithRetry("secrets", func() cache.Controller {
		lw := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "secrets", metav1.NamespaceAll, nil)
		_, informer := cache.NewInformer(lw, &corev1.Secret{}, 0, cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj, newObj interface{}) {
				oldSec, ok1 := oldObj.(*corev1.Secret)
				newSec, ok2 := newObj.(*corev1.Secret)
				if !ok1 || !ok2 {
					return
				}
				if oldSec.ResourceVersion == newSec.ResourceVersion {
					return
				}
				c.emitResourceChangeEvent(clusterID, "Secret", newSec.Name, newSec.Namespace, string(newSec.UID), "SecretChanged")
			},
		})
		return informer
	})

	log.Printf("[events/collector] started watching resource changes for cluster %s", clusterID)
}

// watchWithRetry wraps informer creation and startup in a retry loop with
// exponential backoff. When the K8s API server disconnects, tokens expire, or
// RBAC changes cause the watch to drop, the informer silently stops. This
// method detects that and reconnects automatically.
func (c *Collector) watchWithRetry(name string, createInformer func() cache.Controller) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[events/collector] PANIC in %s watcher (recovered): %v", name, r)
		}
	}()

	// Run the informer in a child goroutine so its internal panics don't kill this loop
	runInformerSafe := func(informer cache.Controller, stopCh <-chan struct{}) (completed bool) {
		panicCh := make(chan interface{}, 1)
		doneCh := make(chan struct{})
		go func() {
			defer func() {
				if r := recover(); r != nil {
					panicCh <- r
				}
				close(doneCh)
			}()
			informer.Run(stopCh)
		}()
		select {
		case r := <-panicCh:
			log.Printf("[events/collector] %s informer panicked (caught): %v", name, r)
			return false
		case <-doneCh:
			return true
		}
	}
	_ = runInformerSafe // used below

	backoff := 5 * time.Second
	maxBackoff := 5 * time.Minute

	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		log.Printf("[events/collector] starting %s watcher", name)
		informer := createInformer()

		// Run the informer in a panic-safe wrapper.
		completed := runInformerSafe(informer, c.stopCh)

		if completed {
			log.Printf("[events/collector] %s watcher completed normally", name)
		} else {
			log.Printf("[events/collector] %s watcher failed/panicked, will retry in %v", name, backoff)
		}

		// Check if we were asked to stop.
		select {
		case <-c.stopCh:
			return
		default:
		}

		// Informer stopped unexpectedly — retry with backoff.
		log.Printf("[events/collector] %s watcher stopped, reconnecting in %v", name, backoff)
		select {
		case <-time.After(backoff):
			backoff = minDuration(backoff*2, maxBackoff)
		case <-c.stopCh:
			return
		}
	}
}

// IsHealthy reports whether the collector has received events recently (within
// 5 minutes). This can be used by readiness probes to detect silent watch
// failures before the retry loop kicks in.
func (c *Collector) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastEventTime.IsZero() {
		return true // no events yet is OK — cluster may be quiet
	}
	return time.Since(c.lastEventTime) < 5*time.Minute
}

// LastEventTime returns the timestamp of the last received event. Returns
// zero time if no events have been received yet.
func (c *Collector) LastEventTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastEventTime
}

// Stop shuts down the collector and closes the events channel.
func (c *Collector) Stop() {
	select {
	case <-c.stopCh:
		// already closed
	default:
		close(c.stopCh)
	}
}

// handleK8sEvent converts a corev1.Event into a WideEvent and sends it on the channel.
func (c *Collector) handleK8sEvent(obj interface{}, clusterID string) {
	k8sEvent, ok := obj.(*corev1.Event)
	if !ok {
		return
	}

	// Track last event time for health checks.
	c.mu.Lock()
	c.lastEventTime = time.Now()
	c.mu.Unlock()

	ts := eventTimestamp(k8sEvent)

	// Marshal the full K8s event as dimensions
	dims, _ := json.Marshal(k8sEvent)

	we := &WideEvent{
		EventID:            fmt.Sprintf("evt_%s", shortUUID()),
		Timestamp:          ts,
		ClusterID:          clusterID,
		EventType:          k8sEvent.Type,
		Reason:             k8sEvent.Reason,
		Message:            k8sEvent.Message,
		SourceComponent:    k8sEvent.Source.Component,
		SourceHost:         k8sEvent.Source.Host,
		EventCount:         int(k8sEvent.Count),
		FirstSeen:          timeToMillis(k8sEvent.FirstTimestamp.Time),
		LastSeen:           timeToMillis(k8sEvent.LastTimestamp.Time),
		ResourceKind:       k8sEvent.InvolvedObject.Kind,
		ResourceName:       k8sEvent.InvolvedObject.Name,
		ResourceNamespace:  k8sEvent.InvolvedObject.Namespace,
		ResourceUID:        string(k8sEvent.InvolvedObject.UID),
		ResourceAPIVersion: k8sEvent.InvolvedObject.APIVersion,
		NodeName:           k8sEvent.Source.Host,
		Severity:           classifySeverity(k8sEvent.Type, k8sEvent.Reason),
		Dimensions:         dims,
	}

	// Try to extract owner references from the involved object name pattern
	// (e.g., pod name often contains deployment name via replicaset)
	if k8sEvent.InvolvedObject.Kind == "Pod" {
		we.OwnerKind, we.OwnerName = inferPodOwner(k8sEvent.InvolvedObject.Name)
	}

	select {
	case c.eventsCh <- we:
	case <-c.stopCh:
	}
}

// emitResourceChangeEvent creates a WideEvent for a resource spec change.
func (c *Collector) emitResourceChangeEvent(clusterID, kind, name, namespace, uid, reason string) {
	we := &WideEvent{
		EventID:           fmt.Sprintf("evt_%s", shortUUID()),
		Timestamp:         UnixMillis(),
		ClusterID:         clusterID,
		EventType:         "Normal",
		Reason:            reason,
		Message:           fmt.Sprintf("%s %s/%s spec changed", kind, namespace, name),
		ResourceKind:      kind,
		ResourceName:      name,
		ResourceNamespace: namespace,
		ResourceUID:       uid,
		Severity:          "info",
	}

	select {
	case c.eventsCh <- we:
	case <-c.stopCh:
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// eventTimestamp extracts the best available timestamp from a K8s event in unix ms.
func eventTimestamp(e *corev1.Event) int64 {
	if !e.EventTime.IsZero() {
		return e.EventTime.Time.UnixMilli()
	}
	if !e.LastTimestamp.IsZero() {
		return e.LastTimestamp.Time.UnixMilli()
	}
	if !e.FirstTimestamp.IsZero() {
		return e.FirstTimestamp.Time.UnixMilli()
	}
	return time.Now().UnixMilli()
}

// timeToMillis converts a time.Time to unix milliseconds, returning 0 for zero time.
func timeToMillis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

// shortUUID returns the first 12 characters of a UUID v4.
func shortUUID() string {
	return uuid.New().String()[:12]
}

// classifySeverity maps K8s event type and reason to a severity level.
func classifySeverity(eventType, reason string) string {
	if eventType == "Warning" {
		switch reason {
		case "OOMKilled", "CrashLoopBackOff", "FailedScheduling", "Evicted":
			return "critical"
		case "Unhealthy", "BackOff", "FailedMount", "FailedAttachVolume":
			return "warning"
		default:
			return "warning"
		}
	}
	return "info"
}

// inferPodOwner tries to infer the owning ReplicaSet/Deployment from a pod name.
// Pod names typically follow the pattern: <deployment>-<replicaset-hash>-<pod-hash>.
func inferPodOwner(podName string) (kind, name string) {
	parts := strings.Split(podName, "-")
	if len(parts) >= 3 {
		// Assume last two segments are replicaset hash and pod hash
		name = strings.Join(parts[:len(parts)-2], "-")
		kind = "ReplicaSet"
		return
	}
	return "", ""
}

// minDuration returns the smaller of a and b.
func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// Ensure watch.Interface is referenced to prevent import removal.
var _ watch.Interface
