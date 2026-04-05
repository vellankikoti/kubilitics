package events

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// CausalLink represents an inferred cause-effect relationship between two events.
type CausalLink struct {
	CausedByEventID string  `json:"caused_by_event_id"`
	Confidence      float64 `json:"confidence"`
	Rule            string  `json:"rule"`
}

// CausalityEngine infers cause-effect relationships between events using a
// set of deterministic rules. Each rule queries the store for recent events
// matching a known failure pattern.
type CausalityEngine struct {
	store *Store
}

// NewCausalityEngine creates a new causality engine.
func NewCausalityEngine(store *Store) *CausalityEngine {
	return &CausalityEngine{store: store}
}

// InferCause runs all 6 causality rules against the given event and returns
// the first match (highest confidence). Returns nil if no cause is found.
func (ce *CausalityEngine) InferCause(ctx context.Context, event *WideEvent) *CausalLink {
	// Rules ordered by specificity/confidence
	rules := []func(context.Context, *WideEvent) *CausalLink{
		ce.ruleOOMCausesCrashLoop,
		ce.ruleNodeCausesEviction,
		ce.ruleDeploymentCausesPodEvent,
		ce.ruleScaleDownCausesSPOF,
		ce.ruleConfigCausesRestart,
		ce.ruleQuotaCausesScheduling,
	}

	for _, rule := range rules {
		if link := rule(ctx, event); link != nil {
			return link
		}
	}
	return nil
}

// Rule 1: Deployment rollout causes Pod events
// If a Pod event occurs within 5 min of its owning Deployment's rollout event, link them.
func (ce *CausalityEngine) ruleDeploymentCausesPodEvent(ctx context.Context, event *WideEvent) *CausalLink {
	if event.ResourceKind != "Pod" {
		return nil
	}

	// Infer the deployment name from the pod owner
	deployName := event.OwnerName
	if deployName == "" {
		// Try to infer from pod name (strip replicaset + pod hash)
		parts := strings.Split(event.ResourceName, "-")
		if len(parts) >= 3 {
			deployName = strings.Join(parts[:len(parts)-2], "-")
		}
	}
	if deployName == "" {
		return nil
	}

	cause, err := ce.findRecentEventInWindow(ctx,
		event.ClusterID, "Deployment", deployName, event.ResourceNamespace,
		"ScalingReplicaSet", event.Timestamp, 5*time.Minute,
	)
	if err != nil || cause == nil {
		return nil
	}

	return &CausalLink{
		CausedByEventID: cause.EventID,
		Confidence:      0.90,
		Rule:            "deployment_causes_pod_event",
	}
}

// Rule 2: OOMKilled causes CrashLoopBackOff
// If CrashLoopBackOff follows OOMKilled on the same container within 2 min.
func (ce *CausalityEngine) ruleOOMCausesCrashLoop(ctx context.Context, event *WideEvent) *CausalLink {
	if event.Reason != "CrashLoopBackOff" {
		return nil
	}

	cause, err := ce.findRecentEventInWindow(ctx,
		event.ClusterID, event.ResourceKind, event.ResourceName, event.ResourceNamespace,
		"OOMKilled", event.Timestamp, 2*time.Minute,
	)
	if err != nil || cause == nil {
		return nil
	}

	return &CausalLink{
		CausedByEventID: cause.EventID,
		Confidence:      0.95,
		Rule:            "oom_causes_crashloop",
	}
}

// Rule 3: Node NotReady causes Pod eviction
// If Pod eviction follows Node NotReady within 5 min on the same node.
func (ce *CausalityEngine) ruleNodeCausesEviction(ctx context.Context, event *WideEvent) *CausalLink {
	if event.Reason != "Evicted" && event.Reason != "Preempting" {
		return nil
	}
	if event.NodeName == "" {
		return nil
	}

	// Look for a Node NotReady event on the same node
	cause, err := ce.findRecentEventInWindow(ctx,
		event.ClusterID, "Node", event.NodeName, "",
		"NodeNotReady", event.Timestamp, 5*time.Minute,
	)
	if err != nil || cause == nil {
		return nil
	}

	return &CausalLink{
		CausedByEventID: cause.EventID,
		Confidence:      0.95,
		Rule:            "node_causes_eviction",
	}
}

// Rule 4: ConfigMap/Secret change causes Pod restart
// If Pod restart follows ConfigMap/Secret change within 2 min in the same namespace.
func (ce *CausalityEngine) ruleConfigCausesRestart(ctx context.Context, event *WideEvent) *CausalLink {
	if event.ResourceKind != "Pod" {
		return nil
	}
	if event.Reason != "Killing" && event.Reason != "Started" && event.Reason != "Pulled" {
		return nil
	}

	// Look for ConfigMap changes in the same namespace
	cause, err := ce.findRecentConfigChange(ctx,
		event.ClusterID, event.ResourceNamespace, event.Timestamp, 2*time.Minute,
	)
	if err != nil || cause == nil {
		return nil
	}

	return &CausalLink{
		CausedByEventID: cause.EventID,
		Confidence:      0.85,
		Rule:            "config_causes_restart",
	}
}

// Rule 5: Scale-down causes SPOF condition
// If a resource has replica_count=1 after a scaling event.
func (ce *CausalityEngine) ruleScaleDownCausesSPOF(ctx context.Context, event *WideEvent) *CausalLink {
	if event.Reason != "ScalingReplicaSet" {
		return nil
	}
	// Check if the message indicates scaling down to 1
	if !strings.Contains(event.Message, "to 1") && !strings.Contains(event.Message, "Scaled down") {
		return nil
	}

	// The scaling event itself is the cause — flag it
	return &CausalLink{
		CausedByEventID: event.EventID,
		Confidence:      0.90,
		Rule:            "scaledown_causes_spof",
	}
}

// Rule 6: ResourceQuota exceeded causes FailedScheduling
// If FailedScheduling follows ResourceQuota exceeded in the same namespace.
func (ce *CausalityEngine) ruleQuotaCausesScheduling(ctx context.Context, event *WideEvent) *CausalLink {
	if event.Reason != "FailedScheduling" {
		return nil
	}

	// Look for quota exceeded events in the same namespace
	cause, err := ce.findRecentEventInWindow(ctx,
		event.ClusterID, "ResourceQuota", "", event.ResourceNamespace,
		"FailedCreate", event.Timestamp, 5*time.Minute,
	)
	if err != nil || cause == nil {
		return nil
	}

	return &CausalLink{
		CausedByEventID: cause.EventID,
		Confidence:      0.85,
		Rule:            "quota_causes_scheduling",
	}
}

// ---------------------------------------------------------------------------
// Query helpers
// ---------------------------------------------------------------------------

// findRecentEventInWindow searches for a recent event within a time window
// before the given timestamp. It returns nil (not an error) if no match is found.
func (ce *CausalityEngine) findRecentEventInWindow(
	ctx context.Context,
	clusterID, resourceKind, resourceName, namespace, reason string,
	timestamp int64, window time.Duration,
) (*WideEvent, error) {
	since := timestamp - window.Milliseconds()
	q := EventQuery{
		ClusterID:    clusterID,
		ResourceKind: resourceKind,
		ResourceName: resourceName,
		Namespace:    namespace,
		Reason:       reason,
		Since:        &since,
		Until:        &timestamp,
		Limit:        1,
	}

	events, err := ce.store.QueryEvents(ctx, q)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if len(events) == 0 {
		return nil, nil
	}
	return &events[0], nil
}

// findRecentConfigChange looks for ConfigMap or Secret change events
// in the given namespace within the time window.
func (ce *CausalityEngine) findRecentConfigChange(
	ctx context.Context,
	clusterID, namespace string,
	timestamp int64, window time.Duration,
) (*WideEvent, error) {
	// Try ConfigMap first
	cause, err := ce.findRecentEventInWindow(ctx,
		clusterID, "ConfigMap", "", namespace,
		"ConfigChanged", timestamp, window,
	)
	if err != nil {
		return nil, err
	}
	if cause != nil {
		return cause, nil
	}

	// Try Secret
	return ce.findRecentEventInWindow(ctx,
		clusterID, "Secret", "", namespace,
		"SecretChanged", timestamp, window,
	)
}
