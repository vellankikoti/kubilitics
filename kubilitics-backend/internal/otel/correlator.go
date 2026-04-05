package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/kubilitics/kubilitics-backend/internal/events"
)

// Correlator links OTel spans to K8s events by matching pod name + time window.
// After spans are stored, CorrelateSpan finds K8s events for the same pod
// within +/- 5 minutes and writes matched event IDs onto the span's
// linked_event_ids column.
type Correlator struct {
	otelStore   *Store
	eventsStore *events.Store
}

// NewCorrelator creates a new Correlator.
func NewCorrelator(otelStore *Store, eventsStore *events.Store) *Correlator {
	return &Correlator{
		otelStore:   otelStore,
		eventsStore: eventsStore,
	}
}

// CorrelateSpan finds K8s events that match the span's pod within a +/- 5 minute
// window around the span's start time, and updates the span's linked_event_ids.
func (c *Correlator) CorrelateSpan(ctx context.Context, span *Span) error {
	if span.K8sPodName == "" {
		return nil // no K8s context, can't correlate
	}

	// Span times are in nanoseconds; events use milliseconds.
	startMs := span.StartTime / 1_000_000
	windowMs := int64(300_000) // 5 minutes
	from := startMs - windowMs
	to := startMs + windowMs

	matchedEvents, err := c.eventsStore.QueryEvents(ctx, events.EventQuery{
		ClusterID:    span.ClusterID,
		ResourceName: span.K8sPodName,
		Since:        &from,
		Until:        &to,
		Limit:        10,
	})
	if err != nil {
		return fmt.Errorf("correlate span %s: query events: %w", span.SpanID, err)
	}

	if len(matchedEvents) == 0 {
		return nil
	}

	eventIDs := make([]string, len(matchedEvents))
	for i, e := range matchedEvents {
		eventIDs[i] = e.EventID
	}

	idsJSON, err := json.Marshal(eventIDs)
	if err != nil {
		return fmt.Errorf("correlate span %s: marshal event IDs: %w", span.SpanID, err)
	}

	if err := c.otelStore.UpdateSpanLinkedEvents(ctx, span.SpanID, string(idsJSON)); err != nil {
		return fmt.Errorf("correlate span %s: update linked events: %w", span.SpanID, err)
	}

	slog.Debug("correlated span with K8s events",
		"span_id", span.SpanID,
		"pod", span.K8sPodName,
		"linked_events", len(matchedEvents),
	)
	return nil
}

// CorrelateSpans runs correlation for a batch of spans. Errors on individual
// spans are logged but do not abort the batch.
func (c *Correlator) CorrelateSpans(ctx context.Context, spans []Span) {
	for i := range spans {
		if err := c.CorrelateSpan(ctx, &spans[i]); err != nil {
			slog.Warn("span correlation failed",
				"span_id", spans[i].SpanID,
				"error", err,
			)
		}
	}
}
