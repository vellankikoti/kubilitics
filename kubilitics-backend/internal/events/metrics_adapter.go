package events

import (
	"context"
	"strconv"
	"strings"
)

// MetricsAdapter wraps the existing service.MetricsService to implement
// the MetricsProvider interface used by the Enricher. It uses function
// closures to avoid importing the service package directly — the caller
// in main.go provides closures that call the actual MetricsService methods.
type MetricsAdapter struct {
	getPodMetrics  func(ctx context.Context, clusterID, namespace, podName string) (cpu string, memory string, err error)
	getNodeMetrics func(ctx context.Context, clusterID, nodeName string) (cpu string, memory string, err error)
}

// NewMetricsAdapter creates a MetricsAdapter from function closures.
func NewMetricsAdapter(
	getPodMetrics func(ctx context.Context, clusterID, namespace, podName string) (cpu string, memory string, err error),
	getNodeMetrics func(ctx context.Context, clusterID, nodeName string) (cpu string, memory string, err error),
) *MetricsAdapter {
	return &MetricsAdapter{
		getPodMetrics:  getPodMetrics,
		getNodeMetrics: getNodeMetrics,
	}
}

// GetPodMetrics implements MetricsProvider. It calls the underlying service,
// parses the formatted strings back to numeric values, and returns 0,0,nil
// when metrics are unavailable.
func (a *MetricsAdapter) GetPodMetrics(ctx context.Context, clusterID, namespace, podName string) (float64, float64, error) {
	cpu, mem, err := a.getPodMetrics(ctx, clusterID, namespace, podName)
	if err != nil {
		// Metrics unavailable — not an error for enrichment purposes.
		return 0, 0, nil
	}
	cpuMilli := parseCPUMilli(cpu)
	memoryMiB := parseMemoryMiB(mem)
	return cpuMilli, memoryMiB, nil
}

// GetNodeMetrics implements MetricsProvider. Same semantics as GetPodMetrics.
func (a *MetricsAdapter) GetNodeMetrics(ctx context.Context, clusterID, nodeName string) (float64, float64, error) {
	cpu, mem, err := a.getNodeMetrics(ctx, clusterID, nodeName)
	if err != nil {
		return 0, 0, nil
	}
	cpuMilli := parseCPUMilli(cpu)
	memoryMiB := parseMemoryMiB(mem)
	return cpuMilli, memoryMiB, nil
}

// parseCPUMilli parses a formatted CPU string like "2.79m" into millicores.
func parseCPUMilli(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	s = strings.TrimSuffix(s, "m")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// parseMemoryMiB parses a formatted memory string like "35.60Mi" into MiB.
func parseMemoryMiB(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	s = strings.TrimSuffix(s, "Mi")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// Verify MetricsAdapter implements MetricsProvider at compile time.
var _ MetricsProvider = (*MetricsAdapter)(nil)
