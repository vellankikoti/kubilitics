package events

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kubilitics/kubilitics-backend/internal/service"
	"k8s.io/client-go/kubernetes"
)

// ---------------------------------------------------------------------------
// SQLite Disk Guard
// ---------------------------------------------------------------------------

const (
	// DefaultMaxDBSizeMB is the maximum SQLite database size (1 GB) before aggressive pruning kicks in.
	DefaultMaxDBSizeMB = 1024
	// DBWarningThreshold is the fraction of DefaultMaxDBSizeMB at which a warning is logged.
	DBWarningThreshold = 0.8
)

// Pipeline orchestrates all Events Intelligence components: collection,
// enrichment, causality inference, incident detection, relationship building,
// insight generation, and state snapshots. It is the single entry point that
// wires everything together.
type Pipeline struct {
	store         *Store
	collector     *Collector
	enricher      *Enricher
	causality     *CausalityEngine
	changes       *ChangeDetector
	incidents     *IncidentDetector
	relationships *RelationshipBuilder
	insights      *InsightsEngine
	snapshots     *SnapshotManager

	logCollector  *LogCollector     // optional — log persistence & cross-pod search
	alertNotifier AlertNotifier     // optional — wired to insights engine on Start()
	subscribers   []chan *WideEvent // SSE subscribers
	mu            sync.RWMutex
	ctx           context.Context    // cancelled on Stop — propagated to all goroutines
	cancel        context.CancelFunc
	stopCh        chan struct{}
	clusterID     string             // set during Start()
	startedAt     time.Time          // set during Start(), used for uptime calculation
}

// NewPipeline creates a new Pipeline and all sub-components.
func NewPipeline(db *sqlx.DB) *Pipeline {
	store := NewStore(db)
	return &Pipeline{
		store:         store,
		collector:     NewCollector(store),
		enricher:      NewEnricher(store, nil),
		causality:     NewCausalityEngine(store),
		changes:       NewChangeDetector(store),
		incidents:     NewIncidentDetector(store),
		relationships: NewRelationshipBuilder(store),
		stopCh:        make(chan struct{}),
	}
}

// Store returns the underlying event store (used by the API handler).
func (p *Pipeline) Store() *Store {
	return p.store
}

// SetAlertNotifier attaches an AlertNotifier to the insights engine so that
// newly detected insights trigger webhook/Slack/in-app notifications.
// Must be called before Start(); if the insights engine is already running the
// notifier is attached immediately.
func (p *Pipeline) SetAlertNotifier(n AlertNotifier) {
	p.alertNotifier = n
	if p.insights != nil {
		p.insights.SetNotifier(n)
	}
}

// SetMetricsProvider configures the enricher to fetch real-time CPU/memory
// metrics for events as they flow through the pipeline.
func (p *Pipeline) SetMetricsProvider(mp MetricsProvider) {
	p.enricher.metrics = mp
}

// SetLogCollector attaches a LogCollector to the pipeline so that pod logs are
// periodically collected and persisted for cross-pod search.
func (p *Pipeline) SetLogCollector(lc *LogCollector) {
	p.logCollector = lc
}

// StartLogCollector creates and starts the log collector if one hasn't been set.
// It should be called after Start() once a LogsService is available.
func (p *Pipeline) StartLogCollector(logsService service.LogsService, clientset kubernetes.Interface, clusterID string) {
	if p.logCollector == nil {
		p.logCollector = NewLogCollector(p.store, clusterID)
	}
	p.logCollector.Start(logsService, clientset)
}

// Start begins the pipeline: collector, insights engine, snapshot manager,
// and the main event processing goroutine.
func (p *Pipeline) Start(clientset kubernetes.Interface, clusterID string) error {
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.clusterID = clusterID
	p.startedAt = time.Now()

	// Start the K8s event collector.
	if err := p.collector.Start(clientset, clusterID); err != nil {
		p.cancel()
		return fmt.Errorf("start collector: %w", err)
	}
	p.collector.WatchResourceChanges(clientset, clusterID)

	// Create per-cluster components that need clusterID.
	p.insights = NewInsightsEngine(p.store, clusterID)
	if p.alertNotifier != nil {
		p.insights.SetNotifier(p.alertNotifier)
	}
	p.insights.Start(p.ctx)

	p.snapshots = NewSnapshotManager(p.store, clusterID)
	p.snapshots.Start(p.ctx, 5*time.Minute)

	// Main event processing goroutine.
	go p.processEvents()

	// Background: resolve stale incidents every 60s.
	go p.incidentResolutionLoop()

	// Background: retention pruning every hour.
	go p.retentionLoop()

	log.Printf("[events/pipeline] started for cluster %s", clusterID)
	return nil
}

// Stop shuts down all pipeline components gracefully.
func (p *Pipeline) Stop() {
	select {
	case <-p.stopCh:
		return // already stopped
	default:
		close(p.stopCh)
	}

	// Cancel the pipeline context — all goroutines using p.ctx will exit.
	if p.cancel != nil {
		p.cancel()
	}

	p.collector.Stop()
	if p.logCollector != nil {
		p.logCollector.Stop()
	}
	if p.insights != nil {
		p.insights.Stop()
	}
	if p.snapshots != nil {
		p.snapshots.Stop()
	}

	// Close all subscriber channels.
	p.mu.Lock()
	for _, ch := range p.subscribers {
		close(ch)
	}
	p.subscribers = nil
	p.mu.Unlock()

	log.Printf("[events/pipeline] stopped")
}

// Health returns a health snapshot for this pipeline.
func (p *Pipeline) Health(ctx context.Context) *PipelineHealth {
	health := &PipelineHealth{
		ClusterID:   p.clusterID,
		CollectorOK: p.collector.IsHealthy(),
		InsightsOK:  p.insights != nil,
		SnapshotsOK: p.snapshots != nil,
	}

	// Set last event time.
	lastEvt := p.collector.LastEventTime()
	if !lastEvt.IsZero() {
		health.LastEventTime = lastEvt.UnixMilli()
	}

	// Count events in last 5 minutes.
	fiveMinAgo := time.Now().Add(-5 * time.Minute).UnixMilli()
	if count, err := p.store.CountEventsSince(ctx, p.clusterID, fiveMinAgo); err == nil {
		health.EventsLast5Min = count
	}

	// Uptime.
	if !p.startedAt.IsZero() {
		health.Uptime = int64(time.Since(p.startedAt).Seconds())
	}

	// Determine status.
	if health.CollectorOK && health.InsightsOK && health.SnapshotsOK {
		health.Status = "healthy"
	} else if health.CollectorOK {
		health.Status = "degraded"
	} else {
		health.Status = "down"
	}

	return health
}

// Subscribe registers an SSE subscriber and returns a read-only channel.
func (p *Pipeline) Subscribe() <-chan *WideEvent {
	ch := make(chan *WideEvent, 64)
	p.mu.Lock()
	p.subscribers = append(p.subscribers, ch)
	p.mu.Unlock()
	return ch
}

// Unsubscribe removes an SSE subscriber.
func (p *Pipeline) Unsubscribe(ch <-chan *WideEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, sub := range p.subscribers {
		if sub == ch {
			close(sub)
			p.subscribers = append(p.subscribers[:i], p.subscribers[i+1:]...)
			return
		}
	}
}

// processEvents is the main goroutine that reads from the collector,
// enriches, infers causality, evaluates incidents, stores, builds
// relationships, and broadcasts to SSE subscribers.
func (p *Pipeline) processEvents() {
	for {
		select {
		case event, ok := <-p.collector.Events():
			if !ok {
				return
			}

			// 1. Enrich with Kubilitics context.
			event = p.enricher.Enrich(p.ctx, event)

			// 2. Infer causality.
			if link := p.causality.InferCause(p.ctx, event); link != nil {
				event.CausedByEventID = &link.CausedByEventID
				// Inherit or create correlation group.
				if parent, _ := p.store.GetEvent(p.ctx, link.CausedByEventID); parent != nil && parent.CorrelationGroupID != "" {
					event.CorrelationGroupID = parent.CorrelationGroupID
				} else {
					event.CorrelationGroupID = fmt.Sprintf("grp_%d", time.Now().UnixNano())
				}
			}

			// 3. Check for incident.
			p.incidents.Evaluate(p.ctx, event)

			// 4. Store the event.
			if err := p.store.InsertEvent(p.ctx, event); err != nil {
				log.Printf("[events/pipeline] failed to store event %s: %v", event.EventID, err)
				continue
			}

			// 5. Build relationships.
			if err := p.relationships.BuildRelationships(p.ctx, event); err != nil {
				log.Printf("[events/pipeline] failed to build relationships for %s: %v", event.EventID, err)
			}

			// 6. Broadcast to SSE subscribers.
			p.broadcast(event)

		case <-p.ctx.Done():
			return
		}
	}
}

// broadcast sends an event to all SSE subscribers (non-blocking).
func (p *Pipeline) broadcast(event *WideEvent) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, ch := range p.subscribers {
		select {
		case ch <- event:
		default:
			// Subscriber is slow; drop the event to avoid blocking.
		}
	}
}

// incidentResolutionLoop checks for stale incidents every 60 seconds.
func (p *Pipeline) incidentResolutionLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := p.incidents.ResolveStaleIncidents(p.ctx); err != nil {
				log.Printf("[events/pipeline] incident resolution error: %v", err)
			}
		case <-p.ctx.Done():
			return
		}
	}
}

// retentionLoop prunes old events (>7 days) and snapshots (>30 days) every hour,
// and runs the SQLite disk guard check.
func (p *Pipeline) retentionLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour).UnixMilli()
			if n, err := p.store.PruneOldEvents(p.ctx, sevenDaysAgo); err != nil {
				log.Printf("[events/pipeline] event pruning error: %v", err)
			} else if n > 0 {
				log.Printf("[events/pipeline] pruned %d old events", n)
			}

			thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour).UnixMilli()
			if n, err := p.store.PruneOldSnapshots(p.ctx, thirtyDaysAgo); err != nil {
				log.Printf("[events/pipeline] snapshot pruning error: %v", err)
			} else if n > 0 {
				log.Printf("[events/pipeline] pruned %d old snapshots", n)
			}

			// Prune stored logs older than 3 days.
			if n, err := p.store.PruneLogs(p.ctx, 3); err != nil {
				log.Printf("[events/pipeline] log pruning error: %v", err)
			} else if n > 0 {
				log.Printf("[events/pipeline] pruned %d old stored logs", n)
			}

			// SQLite disk guard: aggressive prune if DB size exceeds limit.
			p.checkDBSize(p.ctx)
		case <-p.ctx.Done():
			return
		}
	}
}

// checkDBSize implements the SQLite disk guard. It logs a warning when the DB
// approaches the size limit and performs aggressive pruning when it exceeds the limit.
func (p *Pipeline) checkDBSize(ctx context.Context) {
	sizeBytes, err := p.store.GetDBSizeBytes()
	if err != nil {
		log.Printf("[events/pipeline] disk guard: failed to read DB size: %v", err)
		return
	}

	sizeMB := float64(sizeBytes) / (1024 * 1024)
	maxMB := float64(DefaultMaxDBSizeMB)

	if sizeMB > maxMB*DBWarningThreshold {
		log.Printf("[events/pipeline] WARNING: DB size %.1fMB approaching limit %.0fMB", sizeMB, maxMB)
	}

	if sizeMB <= maxMB {
		return
	}

	log.Printf("[events/pipeline] DB size %.1fMB exceeds limit %.0fMB — aggressive pruning", sizeMB, maxMB)

	// Aggressive prune: reduce retention to 3 days for events, 1 day for logs.
	threeDaysAgo := time.Now().Add(-3 * 24 * time.Hour).UnixMilli()
	if n, err := p.store.PruneOldEvents(ctx, threeDaysAgo); err != nil {
		log.Printf("[events/pipeline] aggressive event prune error: %v", err)
	} else if n > 0 {
		log.Printf("[events/pipeline] aggressive prune: deleted %d events (>3 days)", n)
	}
	if n, err := p.store.PruneLogs(ctx, 1); err != nil {
		log.Printf("[events/pipeline] aggressive log prune error: %v", err)
	} else if n > 0 {
		log.Printf("[events/pipeline] aggressive prune: deleted %d logs (>1 day)", n)
	}

	// If still over limit, emergency prune to 1 day events, delete all logs.
	sizeAfter, _ := p.store.GetDBSizeBytes()
	if float64(sizeAfter)/(1024*1024) > maxMB {
		oneDayAgo := time.Now().Add(-1 * 24 * time.Hour).UnixMilli()
		p.store.PruneOldEvents(ctx, oneDayAgo)
		p.store.PruneLogs(ctx, 0) // delete all logs
		log.Printf("[events/pipeline] emergency prune complete (1-day events, all logs deleted)")
	}

	// Reclaim space from deleted rows.
	if _, err := p.store.db.Exec("VACUUM"); err != nil {
		log.Printf("[events/pipeline] VACUUM error: %v", err)
	} else {
		finalSize, _ := p.store.GetDBSizeBytes()
		log.Printf("[events/pipeline] VACUUM complete, DB size now %.1fMB", float64(finalSize)/(1024*1024))
	}
}
