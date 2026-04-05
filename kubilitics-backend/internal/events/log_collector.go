package events

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kubilitics/kubilitics-backend/internal/service"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// LogCollector periodically fetches recent pod logs and persists them as
// structured StoredLog records for cross-pod search and aggregation.
type LogCollector struct {
	store     *Store
	clusterID string
	stopCh    chan struct{}
}

// NewLogCollector creates a new LogCollector.
func NewLogCollector(store *Store, clusterID string) *LogCollector {
	return &LogCollector{
		store:     store,
		clusterID: clusterID,
		stopCh:    make(chan struct{}),
	}
}

// Start begins the periodic log collection goroutine.
func (lc *LogCollector) Start(logsService service.LogsService, clientset kubernetes.Interface) {
	go lc.run(logsService, clientset)
	log.Printf("[events/log_collector] started for cluster %s", lc.clusterID)
}

// Stop shuts down the log collector.
func (lc *LogCollector) Stop() {
	select {
	case <-lc.stopCh:
	default:
		close(lc.stopCh)
	}
	log.Printf("[events/log_collector] stopped for cluster %s", lc.clusterID)
}

// run is the main loop that collects logs every 30 seconds.
func (lc *LogCollector) run(logsService service.LogsService, clientset kubernetes.Interface) {
	// Initial collection after a short delay to let pods settle.
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			lc.collect(logsService, clientset)
			timer.Reset(30 * time.Second)
		case <-lc.stopCh:
			return
		}
	}
}

// collect fetches recent logs for all running pods and stores them.
func (lc *LogCollector) collect(logsService service.LogsService, clientset kubernetes.Interface) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	// List running pods across all namespaces.
	pods, err := clientset.CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{
		FieldSelector: "status.phase=Running",
	})
	if err != nil {
		log.Printf("[events/log_collector] failed to list pods: %v", err)
		return
	}

	var allLogs []StoredLog
	const maxLogsPerCycle = 1000
	now := time.Now()

	for _, pod := range pods.Items {
		if len(allLogs) >= maxLogsPerCycle {
			break
		}

		namespace := pod.Namespace
		podName := pod.Name

		// Determine owner from ownerReferences.
		ownerKind, ownerName := "", ""
		for _, ref := range pod.OwnerReferences {
			if ref.Controller != nil && *ref.Controller {
				ownerKind = ref.Kind
				ownerName = ref.Name
				break
			}
		}
		// If owner is a ReplicaSet, try to infer Deployment name.
		if ownerKind == "ReplicaSet" {
			depName := inferDeploymentFromRS(ownerName)
			if depName != "" {
				ownerKind = "Deployment"
				ownerName = depName
			}
		}

		// Fetch last 30 seconds of logs (tailLines=100 as a reasonable cap per pod).
		for _, container := range pod.Spec.Containers {
			if len(allLogs) >= maxLogsPerCycle {
				break
			}

			stream, err := logsService.GetPodLogs(ctx, lc.clusterID, namespace, podName, container.Name, false, 100)
			if err != nil {
				continue // Pod may have restarted or container not ready.
			}

			scanner := bufio.NewScanner(stream)
			scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)
			for scanner.Scan() {
				if len(allLogs) >= maxLogsPerCycle {
					break
				}

				rawLine := scanner.Text()
				if rawLine == "" {
					continue
				}

				sl := lc.parseLine(rawLine, namespace, podName, container.Name, ownerKind, ownerName, now)
				allLogs = append(allLogs, sl)
			}
			_ = stream.Close()
		}
	}

	if len(allLogs) == 0 {
		return
	}

	if err := lc.store.InsertLogs(ctx, allLogs); err != nil {
		log.Printf("[events/log_collector] failed to store %d logs: %v", len(allLogs), err)
	} else {
		log.Printf("[events/log_collector] stored %d log lines from %d pods", len(allLogs), len(pods.Items))
	}
}

// parseLine parses a single log line into a StoredLog, extracting structured
// fields from JSON lines.
func (lc *LogCollector) parseLine(rawLine, namespace, podName, containerName, ownerKind, ownerName string, now time.Time) StoredLog {
	timestamp := now.UnixMilli()
	level := ""
	message := rawLine
	isStructured := false
	fields := JSONText("{}")

	// Try to parse as JSON.
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(rawLine), &parsed); err == nil {
		isStructured = true
		fieldsBytes, _ := json.Marshal(parsed)
		fields = JSONText(fieldsBytes)

		// Extract common fields.
		if l, ok := extractStringField(parsed, "level", "severity", "log.level", "loglevel"); ok {
			level = normalizeLevel(l)
		}
		if m, ok := extractStringField(parsed, "message", "msg", "log", "text"); ok {
			message = m
		}
		if t, ok := extractTimestamp(parsed); ok {
			timestamp = t
		}
	} else {
		// Try to extract level from non-JSON log lines.
		level = extractLevelFromText(rawLine)
	}

	// Generate deterministic log_id for dedup.
	logID := generateLogID(lc.clusterID, podName, rawLine, timestamp)

	return StoredLog{
		LogID:         logID,
		Timestamp:     timestamp,
		ClusterID:     lc.clusterID,
		Namespace:     namespace,
		PodName:       podName,
		ContainerName: containerName,
		Level:         level,
		Message:       message,
		RawLine:       rawLine,
		IsStructured:  isStructured,
		Fields:        fields,
		OwnerKind:     ownerKind,
		OwnerName:     ownerName,
	}
}

// generateLogID creates a deterministic ID from cluster+pod+rawLine+timestamp.
func generateLogID(clusterID, podName, rawLine string, timestamp int64) string {
	data := fmt.Sprintf("%s|%s|%s|%d", clusterID, podName, rawLine, timestamp)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)[:16]
}

// extractStringField looks for a value in a map trying multiple keys.
func extractStringField(m map[string]interface{}, keys ...string) (string, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s, true
			}
		}
	}
	return "", false
}

// extractTimestamp tries to parse a timestamp from common JSON log fields.
func extractTimestamp(m map[string]interface{}) (int64, bool) {
	for _, key := range []string{"timestamp", "ts", "time", "@timestamp"} {
		v, ok := m[key]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case float64:
			// Could be unix seconds or unix ms.
			if t > 1e12 {
				return int64(t), true // already ms
			}
			return int64(t * 1000), true // seconds -> ms
		case string:
			// Try RFC3339.
			if parsed, err := time.Parse(time.RFC3339, t); err == nil {
				return parsed.UnixMilli(), true
			}
			if parsed, err := time.Parse(time.RFC3339Nano, t); err == nil {
				return parsed.UnixMilli(), true
			}
		}
	}
	return 0, false
}

// normalizeLevel normalizes log level strings to uppercase canonical forms.
func normalizeLevel(level string) string {
	upper := strings.ToUpper(strings.TrimSpace(level))
	switch upper {
	case "ERROR", "ERR", "FATAL", "CRITICAL", "SEVERE":
		return "ERROR"
	case "WARN", "WARNING":
		return "WARN"
	case "INFO", "INFORMATION":
		return "INFO"
	case "DEBUG", "TRACE", "FINE", "FINER", "FINEST":
		return "DEBUG"
	default:
		return upper
	}
}

// extractLevelFromText tries to find a log level in unstructured text.
func extractLevelFromText(line string) string {
	upper := strings.ToUpper(line)
	// Check common patterns like [ERROR], level=error, ERROR:, etc.
	for _, pattern := range []struct {
		search string
		level  string
	}{
		{"ERROR", "ERROR"},
		{"FATAL", "ERROR"},
		{"WARN", "WARN"},
		{"INFO", "INFO"},
		{"DEBUG", "DEBUG"},
	} {
		if strings.Contains(upper, pattern.search) {
			return pattern.level
		}
	}
	return ""
}

// inferDeploymentFromRS tries to infer the Deployment name from a ReplicaSet
// name. ReplicaSet names follow the pattern: <deployment>-<hash>.
func inferDeploymentFromRS(rsName string) string {
	// Find the last hyphen followed by what looks like a hash (5+ alphanumeric chars).
	idx := strings.LastIndex(rsName, "-")
	if idx > 0 && len(rsName)-idx-1 >= 5 {
		return rsName[:idx]
	}
	return ""
}
