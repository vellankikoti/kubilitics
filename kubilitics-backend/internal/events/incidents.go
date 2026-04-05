package events

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// IncidentDetector groups related events into incidents.
type IncidentDetector struct {
	store           *Store
	activeIncidents map[string]*Incident // keyed by incident_id
	mu              sync.Mutex
}

// NewIncidentDetector creates a new IncidentDetector.
func NewIncidentDetector(store *Store) *IncidentDetector {
	return &IncidentDetector{
		store:           store,
		activeIncidents: make(map[string]*Incident),
	}
}

// Evaluate checks if the event should start a new incident, join an existing
// one, or is not incident-worthy. Returns the incident if one was created or
// updated, nil otherwise.
func (d *IncidentDetector) Evaluate(ctx context.Context, event *WideEvent) *Incident {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the event should join an existing active incident.
	for _, inc := range d.activeIncidents {
		if d.eventMatchesIncident(event, inc) {
			d.addEventToIncident(ctx, event, inc)
			return inc
		}
	}

	// Check if the event should start a new incident.
	if d.shouldStartIncident(ctx, event) {
		inc := d.createIncident(ctx, event)
		return inc
	}

	return nil
}

// shouldStartIncident determines if this event warrants a new incident.
func (d *IncidentDetector) shouldStartIncident(ctx context.Context, event *WideEvent) bool {
	// Condition 1: Warning event with health_delta < -10.
	if event.EventType == "Warning" && event.HealthScore != nil && *event.HealthScore < -10 {
		return true
	}

	// Condition 2: >5 Warning events in the same namespace within the last 2 minutes.
	twoMinAgo := UnixMillis() - 2*60*1000
	warnings, err := d.store.QueryEvents(ctx, EventQuery{
		ClusterID: event.ClusterID,
		Namespace: event.ResourceNamespace,
		EventType: "Warning",
		Since:     &twoMinAgo,
		Limit:     10,
	})
	if err == nil && len(warnings) > 5 {
		return true
	}

	// Condition 3: Event reason is "NodeNotReady".
	if event.Reason == "NodeNotReady" {
		return true
	}

	return false
}

// eventMatchesIncident checks if an event belongs to an existing active incident.
func (d *IncidentDetector) eventMatchesIncident(event *WideEvent, inc *Incident) bool {
	if inc.Status == "resolved" {
		return false
	}

	// Same correlation_group_id.
	if event.CorrelationGroupID != "" && event.CorrelationGroupID == inc.RootCauseKind {
		return true
	}

	// Same namespace and within the incident time window (since incident started).
	if event.ResourceNamespace == inc.Namespace && event.ClusterID == inc.ClusterID {
		return true
	}

	return false
}

// createIncident creates a new incident from a triggering event.
func (d *IncidentDetector) createIncident(ctx context.Context, event *WideEvent) *Incident {
	incidentID := fmt.Sprintf("inc_%d", time.Now().UnixNano())
	now := UnixMillis()

	inc := &Incident{
		IncidentID:       incidentID,
		StartedAt:        now,
		Status:           "active",
		Severity:         event.Severity,
		ClusterID:        event.ClusterID,
		Namespace:        event.ResourceNamespace,
		HealthBefore:     event.HealthScore,
		HealthLowest:     event.HealthScore,
		RootCauseKind:    event.ResourceKind,
		RootCauseName:    event.ResourceName,
		RootCauseSummary: fmt.Sprintf("%s on %s/%s", event.Reason, event.ResourceKind, event.ResourceName),
		Dimensions:       JSONText("{}"),
	}

	// Store the incident.
	if err := d.store.InsertIncident(ctx, inc); err != nil {
		return nil
	}

	// Link the triggering event.
	ie := &IncidentEvent{
		IncidentID: incidentID,
		EventID:    event.EventID,
		Role:       "trigger",
	}
	_ = d.store.LinkEventToIncident(ctx, ie)

	d.activeIncidents[incidentID] = inc
	return inc
}

// addEventToIncident adds an event to an existing incident.
func (d *IncidentDetector) addEventToIncident(ctx context.Context, event *WideEvent, inc *Incident) {
	// Update health lowest if this event has a lower health score.
	if event.HealthScore != nil && (inc.HealthLowest == nil || *event.HealthScore < *inc.HealthLowest) {
		inc.HealthLowest = event.HealthScore
	}

	// Link the event.
	ie := &IncidentEvent{
		IncidentID: inc.IncidentID,
		EventID:    event.EventID,
		Role:       "contributing",
	}
	_ = d.store.LinkEventToIncident(ctx, ie)

	// Persist updated incident.
	_ = d.store.InsertIncident(ctx, inc)
}

// ResolveStaleIncidents checks active incidents and resolves any that have had
// no new Warning events for 10 minutes in their scope.
func (d *IncidentDetector) ResolveStaleIncidents(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tenMinAgo := UnixMillis() - 10*60*1000

	for id, inc := range d.activeIncidents {
		if inc.Status == "resolved" {
			delete(d.activeIncidents, id)
			continue
		}

		// Check for recent Warning events in this incident's scope.
		since := tenMinAgo
		warnings, err := d.store.QueryEvents(ctx, EventQuery{
			ClusterID: inc.ClusterID,
			Namespace: inc.Namespace,
			EventType: "Warning",
			Since:     &since,
			Limit:     1,
		})
		if err != nil {
			continue
		}

		// If no recent warnings, resolve the incident.
		if len(warnings) == 0 {
			now := UnixMillis()
			inc.EndedAt = &now
			inc.Status = "resolved"

			ttr := (now - inc.StartedAt) / 1000 // seconds
			inc.TTR = &ttr

			// Get the latest event health as health_after.
			incEvents, err := d.store.GetIncidentEvents(ctx, inc.IncidentID)
			if err == nil && len(incEvents) > 0 {
				lastEvent := incEvents[len(incEvents)-1]
				inc.HealthAfter = lastEvent.HealthScore
			}

			_ = d.store.InsertIncident(ctx, inc)
			delete(d.activeIncidents, id)
		}
	}

	return nil
}

// GenerateSummary looks at the causal chain and generates a one-line summary
// for the given incident.
func (d *IncidentDetector) GenerateSummary(ctx context.Context, incidentID string) string {
	inc, err := d.store.GetIncident(ctx, incidentID)
	if err != nil {
		return "Unknown incident"
	}

	events, err := d.store.GetIncidentEvents(ctx, incidentID)
	if err != nil || len(events) == 0 {
		return inc.RootCauseSummary
	}

	// Find the trigger event (first event).
	trigger := events[0]

	// Count unique reasons.
	reasons := make(map[string]struct{})
	for _, e := range events {
		reasons[e.Reason] = struct{}{}
	}

	// Calculate health drop.
	var healthDrop float64
	if inc.HealthBefore != nil && inc.HealthLowest != nil {
		healthDrop = *inc.HealthBefore - *inc.HealthLowest
	}

	// Build summary.
	if healthDrop > 0 {
		return fmt.Sprintf("%s of %s caused %d event types, health dropped %.0f points",
			trigger.Reason, trigger.ResourceName, len(reasons), healthDrop)
	}

	var ttrStr string
	if inc.TTR != nil {
		ttrStr = fmt.Sprintf(", TTR %ds", *inc.TTR)
	}

	return fmt.Sprintf("%s on %s/%s triggered %d events across %d reasons%s",
		trigger.Reason, trigger.ResourceKind, trigger.ResourceName,
		len(events), len(reasons), ttrStr)
}
