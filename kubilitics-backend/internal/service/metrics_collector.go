// Package service: MetricsCollector continuously collects metrics for all pods
// across all connected clusters and persists them to SQLite for historical charts.
// One API call per cluster fetches all pod metrics — no per-pod overhead.
package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/kubilitics/kubilitics-backend/internal/k8s"
	"github.com/kubilitics/kubilitics-backend/internal/metrics"
	"github.com/kubilitics/kubilitics-backend/internal/repository"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	collectInterval = 30 * time.Second
	purgeInterval   = 1 * time.Hour
	maxHistoryAge   = 7 * 24 * time.Hour // keep 7 days
	// maxConcurrentNodeStatsCalls bounds how many node kubelet /stats/summary
	// calls fetchNetworkStats fans out at once. Each node call has its own
	// 5s timeout; serially that was O(nodes)*5s worst case on a slow/down
	// node. Matches the bound buildClusterSummary already uses for its own
	// per-resource-kind fan-out (handler.go's maxConcurrentSummaryListCalls).
	maxConcurrentNodeStatsCalls = 10
)

// MetricsCollector runs a background goroutine that fetches all pod metrics
// from every connected cluster and stores them in SQLite.
type MetricsCollector struct {
	clusterService ClusterService
	provider       *metrics.MetricsServerProvider
	repo           *repository.SQLiteRepository
}

// NewMetricsCollector creates a new collector.
func NewMetricsCollector(
	clusterService ClusterService,
	provider *metrics.MetricsServerProvider,
	repo *repository.SQLiteRepository,
) *MetricsCollector {
	return &MetricsCollector{
		clusterService: clusterService,
		provider:       provider,
		repo:           repo,
	}
}

// Start begins the collection loop. Call this at server startup.
func (mc *MetricsCollector) Start(ctx context.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Default().Error("panic in metrics collector loop goroutine", "error", r)
			}
		}()
		// Initial collection after a short delay (let clusters connect first)
		time.Sleep(5 * time.Second)
		mc.collectAll(ctx)

		ticker := time.NewTicker(collectInterval)
		defer ticker.Stop()
		purgeTicker := time.NewTicker(purgeInterval)
		defer purgeTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mc.collectAll(ctx)
			case <-purgeTicker.C:
				mc.purgeOld(ctx)
			}
		}
	}()
	slog.Info("metrics collector started", "interval", collectInterval.String(), "retention", maxHistoryAge.String())
}

func (mc *MetricsCollector) collectAll(ctx context.Context) {
	clusters, err := mc.clusterService.ListClusters(ctx)
	if err != nil {
		slog.Warn("metrics collector: failed to list clusters", "error", err)
		return
	}

	now := time.Now().Unix()
	totalPods := 0

	for _, cluster := range clusters {
		if cluster.Status != "connected" {
			continue
		}
		client, err := mc.clusterService.GetClient(cluster.ID)
		if err != nil {
			continue // skip clusters without active client
		}

		fetchCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		podMetrics, err := mc.provider.GetAllPodMetrics(fetchCtx, client)
		cancel()

		if err != nil {
			slog.Debug("metrics collector: skip cluster", "cluster", cluster.Name, "error", err)
			continue
		}

		if len(podMetrics) == 0 {
			continue
		}

		// Best-effort: fetch per-node network stats from kubelet
		netStats := mc.fetchNetworkStats(ctx, client)

		rows := make([]repository.MetricsHistoryRow, 0, len(podMetrics))
		for _, pm := range podMetrics {
			row := repository.MetricsHistoryRow{
				ClusterID: cluster.ID,
				Namespace: pm.Namespace,
				PodName:   pm.Name,
				Timestamp: now,
				CPUMilli:  pm.CPUMilli,
				MemoryMiB: pm.MemoryMiB,
			}
			// Attach network stats if available
			key := pm.Namespace + "/" + pm.Name
			if ns, ok := netStats[key]; ok {
				row.NetworkRx = ns.rx
				row.NetworkTx = ns.tx
			}
			rows = append(rows, row)
		}

		if err := mc.repo.InsertMetricsHistory(ctx, rows); err != nil {
			slog.Warn("metrics collector: failed to insert", "cluster", cluster.Name, "error", err)
			continue
		}
		totalPods += len(rows)
	}

	if totalPods > 0 {
		slog.Debug("metrics collected", "pods", totalPods)
	}
}

type podNetStats struct {
	rx, tx int64
}

// parseNodeStatsSummary decodes a single kubelet /stats/summary response
// into a namespace/name-keyed map of network rx/tx. Pulled out of
// fetchNetworkStats as its own pure function so the parsing/fallback logic
// (interface sum when the top-level rx/tx are both zero) is unit-testable
// without a real kubelet or a fake clientset's RESTClient (which is nil,
// not a usable stub).
func parseNodeStatsSummary(raw []byte) (map[string]podNetStats, error) {
	var summary struct {
		Pods []struct {
			PodRef struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"podRef"`
			Network struct {
				RxBytes    int64 `json:"rxBytes"`
				TxBytes    int64 `json:"txBytes"`
				Interfaces []struct {
					RxBytes int64 `json:"rxBytes"`
					TxBytes int64 `json:"txBytes"`
				} `json:"interfaces"`
			} `json:"network"`
		} `json:"pods"`
	}
	if err := json.Unmarshal(raw, &summary); err != nil {
		return nil, err
	}
	result := make(map[string]podNetStats, len(summary.Pods))
	for _, pod := range summary.Pods {
		rx := pod.Network.RxBytes
		tx := pod.Network.TxBytes
		if rx == 0 && tx == 0 {
			for _, iface := range pod.Network.Interfaces {
				rx += iface.RxBytes
				tx += iface.TxBytes
			}
		}
		result[pod.PodRef.Namespace+"/"+pod.PodRef.Name] = podNetStats{rx: rx, tx: tx}
	}
	return result, nil
}

// fetchNetworkStats gets network rx/tx for all pods by calling kubelet stats/summary per node.
func (mc *MetricsCollector) fetchNetworkStats(ctx context.Context, client *k8s.Client) map[string]podNetStats {
	result := make(map[string]podNetStats)
	nodes, err := client.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return result
	}

	// Fan out across nodes (bounded) instead of one node at a time — each
	// node call has its own 5s timeout, so serially this was O(nodes)*5s
	// worst case (a single slow/unreachable kubelet delayed every node
	// after it). A write-mutex merges each node's results into the shared
	// map since map writes aren't safe from concurrent goroutines.
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrentNodeStatsCalls)

	for _, node := range nodes.Items {
		nodeName := node.Name
		g.Go(func() error {
			statsCtx, cancel := context.WithTimeout(gctx, 5*time.Second)
			raw, err := client.Clientset.CoreV1().RESTClient().Get().
				AbsPath("/api/v1/nodes/" + nodeName + "/proxy/stats/summary").
				DoRaw(statsCtx)
			cancel()
			if err != nil {
				return nil
			}
			nodeStats, err := parseNodeStatsSummary(raw)
			if err != nil {
				return nil
			}
			mu.Lock()
			for key, stats := range nodeStats {
				result[key] = stats
			}
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait() // every g.Go above always returns nil; errors are per-node skips, not fatal

	return result
}

func (mc *MetricsCollector) purgeOld(ctx context.Context) {
	deleted, err := mc.repo.PurgeOldMetrics(ctx, maxHistoryAge)
	if err != nil {
		slog.Warn("metrics purge failed", "error", err)
		return
	}
	if deleted > 0 {
		slog.Info("metrics purged", "deleted_rows", deleted)
	}
}
