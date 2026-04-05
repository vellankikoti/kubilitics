package events

import (
	"context"
	"fmt"
	"log"
	"time"
)

// AlertNotifier is an optional interface that the InsightsEngine calls whenever
// a new insight is detected.  It decouples the events package from the addon
// notifications package so there is no direct import dependency.
type AlertNotifier interface {
	NotifyInsight(ctx context.Context, insight *Insight) error
}

// InsightsEngine runs anomaly detection rules periodically.
type InsightsEngine struct {
	store     *Store
	clusterID string
	rules     []InsightRule
	notifier  AlertNotifier // optional — fires webhook/Slack/in-app alerts
	stopCh    chan struct{}
}

// SetNotifier attaches an AlertNotifier that is called for every new insight.
func (e *InsightsEngine) SetNotifier(n AlertNotifier) {
	e.notifier = n
}

// InsightRule defines a single anomaly detection rule.
type InsightRule struct {
	Name     string
	Evaluate func(ctx context.Context, store *Store, clusterID string) *Insight
}

// NewInsightsEngine creates a new InsightsEngine with all built-in rules.
func NewInsightsEngine(store *Store, clusterID string) *InsightsEngine {
	e := &InsightsEngine{
		store:     store,
		clusterID: clusterID,
		stopCh:    make(chan struct{}),
	}

	e.rules = []InsightRule{
		{Name: "oomSpike", Evaluate: oomSpike},
		{Name: "restartStorm", Evaluate: restartStorm},
		{Name: "schedulingFailures", Evaluate: schedulingFailures},
		{Name: "imagePullFailures", Evaluate: imagePullFailures},
		{Name: "cascadingFailures", Evaluate: cascadingFailures},
		{Name: "healthDrift", Evaluate: healthDrift},
	}

	return e
}

// Start runs all rules every 60 seconds in a goroutine. The provided context
// is used for all database operations; cancelling it causes the goroutine to
// exit.
func (e *InsightsEngine) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-e.stopCh:
				return
			case <-ticker.C:
				insights := e.RunRules(ctx)
				for i := range insights {
					if err := e.store.InsertInsight(ctx, &insights[i]); err != nil {
						log.Printf("events/insights: failed to store insight %s: %v", insights[i].InsightID, err)
						continue
					}
					// Fire webhook/Slack/in-app notification for the new insight.
					if e.notifier != nil {
						if err := e.notifier.NotifyInsight(ctx, &insights[i]); err != nil {
							log.Printf("events/insights: notifier error for %s: %v", insights[i].InsightID, err)
						}
					}
				}
			}
		}
	}()
}

// Stop stops the insights engine.
func (e *InsightsEngine) Stop() {
	close(e.stopCh)
}

// RunRules executes all rules and returns any new insights.
func (e *InsightsEngine) RunRules(ctx context.Context) []Insight {
	var results []Insight
	for _, rule := range e.rules {
		insight := rule.Evaluate(ctx, e.store, e.clusterID)
		if insight != nil {
			results = append(results, *insight)
		}
	}
	return results
}

// ---------------------------------------------------------------------------
// Built-in rules
// ---------------------------------------------------------------------------

// oomSpike counts OOMKilled in the last 30 min vs 24h baseline.
// Triggers if >3x baseline AND >=3 events.
func oomSpike(ctx context.Context, store *Store, clusterID string) *Insight {
	thirtyMinAgo := UnixMillis() - 30*60*1000
	twentyFourHoursAgo := UnixMillis() - 24*60*60*1000

	recentEvents, err := store.QueryEvents(ctx, EventQuery{
		ClusterID: clusterID,
		Reason:    "OOMKilled",
		Since:     &thirtyMinAgo,
		Limit:     200,
	})
	if err != nil {
		return nil
	}
	recentCount := len(recentEvents)
	if recentCount < 3 {
		return nil
	}

	// Get 24h baseline (events per 30-min window).
	baselineEvents, err := store.QueryEvents(ctx, EventQuery{
		ClusterID: clusterID,
		Reason:    "OOMKilled",
		Since:     &twentyFourHoursAgo,
		Limit:     1000,
	})
	if err != nil {
		return nil
	}

	// Baseline rate = total 24h events / 48 (number of 30-min windows).
	baselineRate := float64(len(baselineEvents)) / 48.0
	if baselineRate > 0 && float64(recentCount) > 3.0*baselineRate {
		return &Insight{
			InsightID: fmt.Sprintf("ins_%d_oomSpike", time.Now().UnixNano()),
			Timestamp: UnixMillis(),
			ClusterID: clusterID,
			Rule:      "oomSpike",
			Severity:  "warning",
			Title:     "OOMKilled spike detected",
			Detail:    fmt.Sprintf("%d OOMKilled events in 30 min (%.1fx above 24h baseline)", recentCount, float64(recentCount)/baselineRate),
			Status:    "active",
		}
	}

	// If baseline is zero but we have >=3, that's also a spike.
	if baselineRate == 0 && recentCount >= 3 {
		return &Insight{
			InsightID: fmt.Sprintf("ins_%d_oomSpike", time.Now().UnixNano()),
			Timestamp: UnixMillis(),
			ClusterID: clusterID,
			Rule:      "oomSpike",
			Severity:  "warning",
			Title:     "OOMKilled spike detected",
			Detail:    fmt.Sprintf("%d OOMKilled events in 30 min (no baseline)", recentCount),
			Status:    "active",
		}
	}

	return nil
}

// restartStorm detects >10 pod restarts (reason=BackOff or Started) in the
// same namespace within 5 minutes.
func restartStorm(ctx context.Context, store *Store, clusterID string) *Insight {
	fiveMinAgo := UnixMillis() - 5*60*1000

	// Check BackOff events.
	backoffEvents, err := store.QueryEvents(ctx, EventQuery{
		ClusterID: clusterID,
		Reason:    "BackOff",
		Since:     &fiveMinAgo,
		Limit:     200,
	})
	if err != nil {
		return nil
	}

	startedEvents, err := store.QueryEvents(ctx, EventQuery{
		ClusterID: clusterID,
		Reason:    "Started",
		Since:     &fiveMinAgo,
		Limit:     200,
	})
	if err != nil {
		return nil
	}

	// Group by namespace.
	nsCounts := make(map[string]int)
	for _, e := range backoffEvents {
		nsCounts[e.ResourceNamespace]++
	}
	for _, e := range startedEvents {
		nsCounts[e.ResourceNamespace]++
	}

	for ns, count := range nsCounts {
		if count > 10 {
			return &Insight{
				InsightID: fmt.Sprintf("ins_%d_restartStorm", time.Now().UnixNano()),
				Timestamp: UnixMillis(),
				ClusterID: clusterID,
				Rule:      "restartStorm",
				Severity:  "warning",
				Title:     fmt.Sprintf("Restart storm in namespace %s", ns),
				Detail:    fmt.Sprintf("%d pod restart events in 5 minutes in namespace %s", count, ns),
				Status:    "active",
			}
		}
	}

	return nil
}

// schedulingFailures detects >5 FailedScheduling events in 10 minutes.
func schedulingFailures(ctx context.Context, store *Store, clusterID string) *Insight {
	tenMinAgo := UnixMillis() - 10*60*1000

	events, err := store.QueryEvents(ctx, EventQuery{
		ClusterID: clusterID,
		Reason:    "FailedScheduling",
		Since:     &tenMinAgo,
		Limit:     200,
	})
	if err != nil {
		return nil
	}

	if len(events) > 5 {
		return &Insight{
			InsightID: fmt.Sprintf("ins_%d_schedulingFailures", time.Now().UnixNano()),
			Timestamp: UnixMillis(),
			ClusterID: clusterID,
			Rule:      "schedulingFailures",
			Severity:  "warning",
			Title:     "Scheduling failures detected",
			Detail:    fmt.Sprintf("%d FailedScheduling events in 10 minutes — possible resource constraints", len(events)),
			Status:    "active",
		}
	}

	return nil
}

// imagePullFailures detects >3 ImagePullBackOff events in 5 minutes.
func imagePullFailures(ctx context.Context, store *Store, clusterID string) *Insight {
	fiveMinAgo := UnixMillis() - 5*60*1000

	events, err := store.QueryEvents(ctx, EventQuery{
		ClusterID: clusterID,
		Reason:    "ImagePullBackOff",
		Since:     &fiveMinAgo,
		Limit:     200,
	})
	if err != nil {
		return nil
	}

	if len(events) > 3 {
		return &Insight{
			InsightID: fmt.Sprintf("ins_%d_imagePullFailures", time.Now().UnixNano()),
			Timestamp: UnixMillis(),
			ClusterID: clusterID,
			Rule:      "imagePullFailures",
			Severity:  "warning",
			Title:     "Image pull failures detected",
			Detail:    fmt.Sprintf("%d ImagePullBackOff events in 5 minutes — check registry access and image tags", len(events)),
			Status:    "active",
		}
	}

	return nil
}

// cascadingFailures detects >3 different resources in the same correlation group.
func cascadingFailures(ctx context.Context, store *Store, clusterID string) *Insight {
	thirtyMinAgo := UnixMillis() - 30*60*1000

	events, err := store.QueryEvents(ctx, EventQuery{
		ClusterID: clusterID,
		EventType: "Warning",
		Since:     &thirtyMinAgo,
		Limit:     500,
	})
	if err != nil {
		return nil
	}

	// Group by correlation_group_id and count unique resources.
	type groupInfo struct {
		resources map[string]struct{}
	}
	groups := make(map[string]*groupInfo)

	for _, e := range events {
		if e.CorrelationGroupID == "" {
			continue
		}
		gi, ok := groups[e.CorrelationGroupID]
		if !ok {
			gi = &groupInfo{resources: make(map[string]struct{})}
			groups[e.CorrelationGroupID] = gi
		}
		resourceKey := fmt.Sprintf("%s/%s", e.ResourceKind, e.ResourceName)
		gi.resources[resourceKey] = struct{}{}
	}

	for groupID, gi := range groups {
		if len(gi.resources) > 3 {
			return &Insight{
				InsightID: fmt.Sprintf("ins_%d_cascadingFailures", time.Now().UnixNano()),
				Timestamp: UnixMillis(),
				ClusterID: clusterID,
				Rule:      "cascadingFailures",
				Severity:  "critical",
				Title:     "Cascading failure detected",
				Detail:    fmt.Sprintf("Correlation group %s has %d affected resources — failure is spreading", groupID, len(gi.resources)),
				Status:    "active",
			}
		}
	}

	return nil
}

// healthDrift detects health score trending down >5 points over 1 hour.
func healthDrift(ctx context.Context, store *Store, clusterID string) *Insight {
	now := UnixMillis()
	oneHourAgo := now - 60*60*1000

	// Get the oldest snapshot in the window.
	oldest, err := store.GetSnapshotAt(ctx, clusterID, oneHourAgo)
	if err != nil {
		return nil
	}

	// Get the newest snapshot.
	newest, err := store.GetSnapshotAt(ctx, clusterID, now)
	if err != nil {
		return nil
	}

	if oldest.SnapshotID == newest.SnapshotID {
		return nil // same snapshot, no comparison possible
	}

	drift := oldest.HealthScore - newest.HealthScore
	if drift > 5 {
		return &Insight{
			InsightID: fmt.Sprintf("ins_%d_healthDrift", time.Now().UnixNano()),
			Timestamp: UnixMillis(),
			ClusterID: clusterID,
			Rule:      "healthDrift",
			Severity:  "info",
			Title:     "Health score drifting down",
			Detail:    fmt.Sprintf("Health score dropped %.1f points in the last hour (%.1f → %.1f)", drift, oldest.HealthScore, newest.HealthScore),
			Status:    "active",
		}
	}

	return nil
}
