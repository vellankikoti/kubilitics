package events

import (
	"context"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ClusterSize categorizes cluster by pod count.
type ClusterSize string

const (
	ClusterSizeSmall  ClusterSize = "small"  // <500 pods
	ClusterSizeMedium ClusterSize = "medium" // 500-2000 pods
	ClusterSizeLarge  ClusterSize = "large"  // 2000-5000 pods
	ClusterSizeXL     ClusterSize = "xl"     // 5000+ pods
)

// ClusterTuning holds auto-tuned settings based on cluster size.
type ClusterTuning struct {
	Size            ClusterSize
	PodCount        int
	MaxDBSizeMB     int
	MaxBatchSize    int
	BatchWindowMs   int
	LogPodsPerCycle int
	EventRetention  int // days
	LogRetention    int // days
	SpanRetention   int // days
}

// DetectClusterSize counts pods and returns tuned settings.
func DetectClusterSize(ctx context.Context, clientset kubernetes.Interface) ClusterTuning {
	// Count pods (use limit=1 with resourceVersion to just get the count)
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{Limit: 1})
	podCount := 0
	if err == nil && pods.RemainingItemCount != nil {
		podCount = int(*pods.RemainingItemCount) + len(pods.Items)
	} else if err == nil {
		// Small cluster — no remaining items
		podCount = len(pods.Items)
		// If we only got 1 and there might be more, do a count-only request
		if len(pods.Items) == 1 {
			allPods, err2 := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
				Limit:           10000,
				ResourceVersion: "0", // serve from cache, fast
			})
			if err2 == nil {
				podCount = len(allPods.Items)
			}
		}
	} else {
		log.Printf("[events/sizing] failed to count pods: %v, defaulting to small", err)
		podCount = 0
	}

	tuning := ClusterTuning{PodCount: podCount}

	switch {
	case podCount >= 5000:
		tuning.Size = ClusterSizeXL
		tuning.MaxDBSizeMB = 4096 // 4GB
		tuning.MaxBatchSize = 200
		tuning.BatchWindowMs = 200
		tuning.LogPodsPerCycle = 100
		tuning.EventRetention = 3
		tuning.LogRetention = 1
		tuning.SpanRetention = 3
	case podCount >= 2000:
		tuning.Size = ClusterSizeLarge
		tuning.MaxDBSizeMB = 2048 // 2GB
		tuning.MaxBatchSize = 100
		tuning.BatchWindowMs = 150
		tuning.LogPodsPerCycle = 75
		tuning.EventRetention = 5
		tuning.LogRetention = 2
		tuning.SpanRetention = 5
	case podCount >= 500:
		tuning.Size = ClusterSizeMedium
		tuning.MaxDBSizeMB = 1536 // 1.5GB
		tuning.MaxBatchSize = 75
		tuning.BatchWindowMs = 100
		tuning.LogPodsPerCycle = 50
		tuning.EventRetention = 7
		tuning.LogRetention = 3
		tuning.SpanRetention = 7
	default:
		tuning.Size = ClusterSizeSmall
		tuning.MaxDBSizeMB = 1024 // 1GB
		tuning.MaxBatchSize = 50
		tuning.BatchWindowMs = 100
		tuning.LogPodsPerCycle = 30
		tuning.EventRetention = 7
		tuning.LogRetention = 3
		tuning.SpanRetention = 7
	}

	log.Printf("[events/sizing] cluster has %d pods → size=%s (DB=%dMB, batch=%d, logPods=%d, retention=%dd)",
		podCount, tuning.Size, tuning.MaxDBSizeMB, tuning.MaxBatchSize, tuning.LogPodsPerCycle, tuning.EventRetention)

	return tuning
}
