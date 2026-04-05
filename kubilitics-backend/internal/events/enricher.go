package events

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

// MetricsProvider fetches current CPU/memory metrics for Kubernetes resources.
// Defined as an interface to avoid circular dependencies with the metrics package.
type MetricsProvider interface {
	GetPodMetrics(ctx context.Context, clusterID, namespace, podName string) (cpuMilli float64, memoryMiB float64, err error)
	GetNodeMetrics(ctx context.Context, clusterID, nodeName string) (cpuMilli float64, memoryMiB float64, err error)
}

// Enricher adds Kubilitics context (health scores, SPOF status, blast radius)
// to raw WideEvents. This is intentionally a stub — real enrichment will be
// wired in when the health service and graph engine integrations are built.
type Enricher struct {
	store   *Store
	metrics MetricsProvider
}

// NewEnricher creates a new event enricher. The metrics parameter is optional
// (pass nil to skip metrics enrichment).
func NewEnricher(store *Store, metrics MetricsProvider) *Enricher {
	return &Enricher{store: store, metrics: metrics}
}

// Enrich augments a raw WideEvent with Kubilitics intelligence fields.
// Currently applies basic heuristics; health scores, SPOF detection, and
// blast radius will be populated by service integrations in the Pipeline.
func (e *Enricher) Enrich(ctx context.Context, event *WideEvent) *WideEvent {
	// Health scores: nil until health service integration
	event.HealthScore = nil

	// SPOF: false until graph engine integration
	event.IsSPOF = 0

	// Blast radius: 0 until graph engine integration
	event.BlastRadius = 0

	// Severity: compute from event type and reason if not already set
	if event.Severity == "" {
		event.Severity = e.classifyEventSeverity(event)
	}

	// For Warning events, set basic impact estimates
	if event.EventType == "Warning" {
		event.BlastRadius = e.estimateBlastRadius(event)
	}

	// Enrich with real-time metrics if a provider is available.
	if e.metrics != nil {
		e.enrichWithMetrics(ctx, event)
	}

	return event
}

// enrichWithMetrics fetches current CPU/memory metrics for the event's resource
// and merges them into the event's dimensions JSON. Uses a 2-second timeout so
// metrics fetching never blocks the pipeline. Failures are logged and skipped.
func (e *Enricher) enrichWithMetrics(ctx context.Context, event *WideEvent) {
	metricsCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var dims map[string]interface{}
	if len(event.Dimensions) > 0 {
		_ = json.Unmarshal(event.Dimensions, &dims)
	}
	if dims == nil {
		dims = make(map[string]interface{})
	}

	switch event.ResourceKind {
	case "Pod":
		if event.ResourceNamespace == "" || event.ResourceName == "" {
			return
		}
		cpuMilli, memoryMiB, err := e.metrics.GetPodMetrics(metricsCtx, event.ClusterID, event.ResourceNamespace, event.ResourceName)
		if err != nil {
			log.Printf("[events/enricher] metrics fetch failed for Pod %s/%s: %v", event.ResourceNamespace, event.ResourceName, err)
			return
		}
		dims["metrics.cpu_milli"] = cpuMilli
		dims["metrics.memory_mib"] = memoryMiB

	case "Node":
		nodeName := event.NodeName
		if nodeName == "" {
			nodeName = event.ResourceName
		}
		if nodeName == "" {
			return
		}
		cpuMilli, memoryMiB, err := e.metrics.GetNodeMetrics(metricsCtx, event.ClusterID, nodeName)
		if err != nil {
			log.Printf("[events/enricher] metrics fetch failed for Node %s: %v", nodeName, err)
			return
		}
		dims["metrics.node_cpu_milli"] = cpuMilli
		dims["metrics.node_memory_mib"] = memoryMiB

	default:
		// Metrics enrichment is only supported for Pod and Node events currently.
		return
	}

	if data, err := json.Marshal(dims); err == nil {
		event.Dimensions = JSONText(data)
	}
}

// classifyEventSeverity determines severity from event characteristics.
func (e *Enricher) classifyEventSeverity(event *WideEvent) string {
	if event.EventType == "Warning" {
		switch event.Reason {
		case "OOMKilled", "CrashLoopBackOff", "Evicted", "FailedScheduling":
			return "critical"
		case "Unhealthy", "BackOff", "FailedMount", "FailedAttachVolume",
			"FailedCreate", "FailedDelete":
			return "warning"
		default:
			return "warning"
		}
	}
	return "info"
}

// estimateBlastRadius provides a rough blast radius estimate for warning events.
// This is a simple heuristic until the real graph engine is wired in.
func (e *Enricher) estimateBlastRadius(event *WideEvent) int {
	switch event.ResourceKind {
	case "Node":
		// Node issues can affect many pods
		return 10
	case "Deployment", "StatefulSet", "DaemonSet":
		// Workload controller issues affect their pods
		return 5
	case "Service":
		// Service issues affect consumers
		return 3
	case "ConfigMap", "Secret":
		// Config changes can affect multiple pods
		return 3
	case "PersistentVolumeClaim", "PersistentVolume":
		// Storage issues affect pods using the volume
		return 2
	default:
		return 1
	}
}
