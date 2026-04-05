package events

import (
	"context"
	"fmt"
	"time"
)

// RelationshipBuilder creates typed links between events.
type RelationshipBuilder struct {
	store *Store
}

// NewRelationshipBuilder creates a new RelationshipBuilder.
func NewRelationshipBuilder(store *Store) *RelationshipBuilder {
	return &RelationshipBuilder{store: store}
}

// BuildRelationships creates relationship records for a newly stored event.
// It evaluates multiple relationship types: caused_by, follows, resolves,
// and co_occurs.
func (rb *RelationshipBuilder) BuildRelationships(ctx context.Context, event *WideEvent) error {
	if err := rb.buildCausedBy(ctx, event); err != nil {
		return fmt.Errorf("caused_by: %w", err)
	}
	if err := rb.buildFollows(ctx, event); err != nil {
		return fmt.Errorf("follows: %w", err)
	}
	if err := rb.buildResolves(ctx, event); err != nil {
		return fmt.Errorf("resolves: %w", err)
	}
	if err := rb.buildCoOccurs(ctx, event); err != nil {
		return fmt.Errorf("co_occurs: %w", err)
	}
	return nil
}

// buildCausedBy creates a "caused_by" relationship if event.CausedByEventID is set.
func (rb *RelationshipBuilder) buildCausedBy(ctx context.Context, event *WideEvent) error {
	if event.CausedByEventID == nil || *event.CausedByEventID == "" {
		return nil
	}

	rel := &EventRelationship{
		SourceEventID:    event.EventID,
		TargetEventID:    *event.CausedByEventID,
		RelationshipType: "caused_by",
		Confidence:       1.0,
		Metadata:         JSONText(`{"source":"explicit"}`),
	}
	return rb.store.InsertRelationship(ctx, rel)
}

// buildFollows finds the most recent event for the same resource_uid within
// 5 minutes and creates a "follows" link.
func (rb *RelationshipBuilder) buildFollows(ctx context.Context, event *WideEvent) error {
	if event.ResourceUID == "" {
		return nil
	}

	fiveMinAgo := event.Timestamp - 5*60*1000
	candidates, err := rb.store.QueryEvents(ctx, EventQuery{
		ClusterID: event.ClusterID,
		Since:     &fiveMinAgo,
		Limit:     50,
	})
	if err != nil {
		return nil // non-fatal
	}

	// Find the most recent event for the same resource_uid that precedes this event.
	var predecessor *WideEvent
	for i := range candidates {
		c := &candidates[i]
		if c.EventID == event.EventID {
			continue
		}
		if c.ResourceUID != event.ResourceUID {
			continue
		}
		if c.Timestamp > event.Timestamp {
			continue
		}
		if predecessor == nil || c.Timestamp > predecessor.Timestamp {
			predecessor = c
		}
	}

	if predecessor == nil {
		return nil
	}

	rel := &EventRelationship{
		SourceEventID:    event.EventID,
		TargetEventID:    predecessor.EventID,
		RelationshipType: "follows",
		Confidence:       0.9,
		Metadata:         JSONText(fmt.Sprintf(`{"gap_ms":%d}`, event.Timestamp-predecessor.Timestamp)),
	}
	return rb.store.InsertRelationship(ctx, rel)
}

// buildResolves checks if this is a Normal event that resolves a recent Warning
// for the same resource and reason within 10 minutes.
func (rb *RelationshipBuilder) buildResolves(ctx context.Context, event *WideEvent) error {
	if event.EventType != "Normal" {
		return nil
	}

	// Look for a recent Warning event for the same resource and reason.
	tenMinAgo := event.Timestamp - 10*60*1000
	candidates, err := rb.store.QueryEvents(ctx, EventQuery{
		ClusterID: event.ClusterID,
		Namespace: event.ResourceNamespace,
		EventType: "Warning",
		Reason:    event.Reason,
		Since:     &tenMinAgo,
		Limit:     10,
	})
	if err != nil {
		return nil // non-fatal
	}

	for i := range candidates {
		c := &candidates[i]
		if c.ResourceName == event.ResourceName && c.ResourceKind == event.ResourceKind {
			rel := &EventRelationship{
				SourceEventID:    event.EventID,
				TargetEventID:    c.EventID,
				RelationshipType: "resolves",
				Confidence:       0.85,
				Metadata:         JSONText(fmt.Sprintf(`{"resolution_ms":%d}`, event.Timestamp-c.Timestamp)),
			}
			if err := rb.store.InsertRelationship(ctx, rel); err != nil {
				return err
			}
			break // resolve the most recent matching warning
		}
	}

	return nil
}

// buildCoOccurs finds events in the same 30-second window in different resources
// but the same namespace. If >3 co-occurrences from the same resource pair in
// 24 hours, creates "co_occurs" links.
func (rb *RelationshipBuilder) buildCoOccurs(ctx context.Context, event *WideEvent) error {
	windowMs := int64(30 * 1000)
	windowStart := event.Timestamp - windowMs
	windowEnd := event.Timestamp + windowMs

	// Find events in the same 30-second window, same namespace.
	coEvents, err := rb.store.QueryEvents(ctx, EventQuery{
		ClusterID: event.ClusterID,
		Namespace: event.ResourceNamespace,
		Since:     &windowStart,
		Limit:     50,
	})
	if err != nil {
		return nil // non-fatal
	}

	// Filter: different resource, within window.
	var windowPeers []WideEvent
	for _, ce := range coEvents {
		if ce.EventID == event.EventID {
			continue
		}
		if ce.ResourceUID == event.ResourceUID {
			continue
		}
		if ce.Timestamp > windowEnd {
			continue
		}
		windowPeers = append(windowPeers, ce)
	}

	if len(windowPeers) == 0 {
		return nil
	}

	// Check 24h co-occurrence frequency for each peer resource.
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour).UnixMilli()
	for _, peer := range windowPeers {
		// Count co-occurrences in 24h by checking events from both resources.
		peerEvents, err := rb.store.QueryEvents(ctx, EventQuery{
			ClusterID:    event.ClusterID,
			Namespace:    event.ResourceNamespace,
			ResourceName: peer.ResourceName,
			Since:        &twentyFourHoursAgo,
			Limit:        100,
		})
		if err != nil {
			continue
		}

		thisEvents, err := rb.store.QueryEvents(ctx, EventQuery{
			ClusterID:    event.ClusterID,
			Namespace:    event.ResourceNamespace,
			ResourceName: event.ResourceName,
			Since:        &twentyFourHoursAgo,
			Limit:        100,
		})
		if err != nil {
			continue
		}

		// Count how many times events from both resources occur within 30s of each other.
		coCount := 0
		for _, te := range thisEvents {
			for _, pe := range peerEvents {
				if abs(te.Timestamp-pe.Timestamp) <= windowMs {
					coCount++
				}
			}
		}

		if coCount > 3 {
			rel := &EventRelationship{
				SourceEventID:    event.EventID,
				TargetEventID:    peer.EventID,
				RelationshipType: "co_occurs",
				Confidence:       float64(coCount) / 100.0,
				Metadata:         JSONText(fmt.Sprintf(`{"co_occurrence_count_24h":%d}`, coCount)),
			}
			_ = rb.store.InsertRelationship(ctx, rel)
		}
	}

	return nil
}

// abs returns the absolute value of an int64.
func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
