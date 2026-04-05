package otel

import (
	"context"

	"github.com/kubilitics/kubilitics-backend/internal/events"
)

// EventsStoreAdapter wraps an otel.Store to satisfy the events.OTelStore interface,
// converting otel.TraceSummary to events.TraceSummaryView to avoid import cycles.
type EventsStoreAdapter struct {
	store *Store
}

// NewEventsStoreAdapter creates a new adapter.
func NewEventsStoreAdapter(store *Store) *EventsStoreAdapter {
	return &EventsStoreAdapter{store: store}
}

// QuerySpansByLinkedEvent delegates to the otel store and converts results.
func (a *EventsStoreAdapter) QuerySpansByLinkedEvent(ctx context.Context, clusterID, eventID string, limit int) ([]events.TraceSummaryView, error) {
	traces, err := a.store.QuerySpansByLinkedEvent(ctx, clusterID, eventID, limit)
	if err != nil {
		return nil, err
	}
	return convertSummaries(traces), nil
}

// QuerySpansByResource delegates to the otel store and converts results.
func (a *EventsStoreAdapter) QuerySpansByResource(ctx context.Context, clusterID, resourceKind, resourceName, namespace string, from, to int64, limit int) ([]events.TraceSummaryView, error) {
	traces, err := a.store.QuerySpansByResource(ctx, clusterID, resourceKind, resourceName, namespace, from, to, limit)
	if err != nil {
		return nil, err
	}
	return convertSummaries(traces), nil
}

func convertSummaries(in []TraceSummary) []events.TraceSummaryView {
	out := make([]events.TraceSummaryView, len(in))
	for i, t := range in {
		out[i] = events.TraceSummaryView{
			TraceID:       t.TraceID,
			RootService:   t.RootService,
			RootOperation: t.RootOperation,
			StartTime:     t.StartTime,
			DurationNs:    t.DurationNs,
			SpanCount:     t.SpanCount,
			ErrorCount:    t.ErrorCount,
			ServiceCount:  t.ServiceCount,
			Status:        t.Status,
			ClusterID:     t.ClusterID,
			Services:      events.JSONText(t.Services),
			UpdatedAt:     t.UpdatedAt,
		}
	}
	return out
}
