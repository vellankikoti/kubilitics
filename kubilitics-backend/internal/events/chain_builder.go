package events

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Configuration & constructor
// ---------------------------------------------------------------------------

// ChainBuilderConfig holds tuning parameters for the ChainBuilder.
type ChainBuilderConfig struct {
	// ConfidenceFloor is the minimum chain confidence returned (default 0.5).
	ConfidenceFloor float64
	// MaxDepth is the maximum causal-walk depth before stopping (default 5).
	MaxDepth int
	// TimeWindow is how far back to look for owner-reference events (default 10 min).
	TimeWindow time.Duration
}

// DefaultChainBuilderConfig returns a ChainBuilderConfig populated with
// production-reasonable defaults.
func DefaultChainBuilderConfig() ChainBuilderConfig {
	return ChainBuilderConfig{
		ConfidenceFloor: 0.5,
		MaxDepth:        5,
		TimeWindow:      10 * time.Minute,
	}
}

// ChainBuilder constructs CausalChain values by recursively walking
// causal-engine inferences and owner-reference links backwards from a symptom.
type ChainBuilder struct {
	store     *Store
	causality *CausalityEngine
	config    ChainBuilderConfig
}

// NewChainBuilder creates a ChainBuilder with the given store, causality engine,
// and configuration.
func NewChainBuilder(store *Store, causality *CausalityEngine, config ChainBuilderConfig) *ChainBuilder {
	return &ChainBuilder{
		store:     store,
		causality: causality,
		config:    config,
	}
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// BuildChain builds a causal chain starting from the resource identified in
// insight.Detail. The returned chain has index 0 = root cause.
// Returns nil, nil when no chain can be constructed.
func (cb *ChainBuilder) BuildChain(ctx context.Context, insight Insight) (*CausalChain, error) {
	ns, kind, name := cb.parseInsightResource(insight)
	if kind == "" || name == "" {
		return nil, nil
	}
	return cb.BuildChainForResource(ctx, insight.ClusterID, ns, kind, name, insight.InsightID)
}

// BuildChainForResource builds a causal chain for a specific resource.
// Returns nil, nil when no causal chain can be constructed (no events found).
func (cb *ChainBuilder) BuildChainForResource(
	ctx context.Context,
	clusterID, namespace, kind, name, insightID string,
) (*CausalChain, error) {
	// Find the most recent event for the symptom resource.
	events, err := cb.store.QueryEvents(ctx, EventQuery{
		ClusterID:    clusterID,
		Namespace:    namespace,
		ResourceKind: kind,
		ResourceName: name,
		Limit:        1,
	})
	if err != nil {
		return nil, fmt.Errorf("chain builder: query symptom events: %w", err)
	}
	if len(events) == 0 {
		return nil, nil
	}

	symptomEvent := &events[0]

	visited := make(map[string]bool)
	var links []CausalLinkV2

	cb.walkCauses(ctx, symptomEvent, visited, &links, 0)

	if len(links) == 0 {
		return nil, nil
	}

	// Reverse links so index 0 is root cause → symptom direction.
	reverseLinks(links)

	// Root cause is the cause of the first link after reversal.
	rootCause := links[0].Cause

	now := time.Now().UTC()
	chain := &CausalChain{
		ID:         fmt.Sprintf("chain-%s", uuid.New().String()[:8]),
		ClusterID:  clusterID,
		InsightID:  insightID,
		RootCause:  rootCause,
		Links:      links,
		Confidence: cb.computeChainConfidence(links),
		Status:     "active",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	return chain, nil
}

// ---------------------------------------------------------------------------
// Internal recursive walk helpers
// ---------------------------------------------------------------------------

// walkCauses recursively walks backwards through causal links starting from
// event, appending discovered links to *links in effect-first order (reversed
// at the end by the caller).
func (cb *ChainBuilder) walkCauses(
	ctx context.Context,
	event *WideEvent,
	visited map[string]bool,
	links *[]CausalLinkV2,
	depth int,
) {
	if depth >= cb.config.MaxDepth {
		return
	}
	if visited[event.EventID] {
		return
	}
	visited[event.EventID] = true

	// Ask the causality engine for the direct cause of this event.
	causalLink := cb.causality.InferCause(ctx, event)
	if causalLink != nil && causalLink.CausedByEventID != event.EventID {
		// Fetch the cause event.
		causeEvent, err := cb.store.GetEvent(ctx, causalLink.CausedByEventID)
		if err == nil && causeEvent != nil {
			timeDelta := event.Timestamp - causeEvent.Timestamp

			// Apply time-proximity boost to confidence.
			boostedConf := cb.applyTimeProximityBoost(causalLink.Confidence, timeDelta)

			link := CausalLinkV2{
				Cause:       cb.eventToCausalNode(causeEvent),
				Effect:      cb.eventToCausalNode(event),
				Rule:        causalLink.Rule,
				Confidence:  boostedConf,
				TimeDeltaMs: timeDelta,
			}
			*links = append(*links, link)

			// Recurse into the cause event.
			cb.walkCauses(ctx, causeEvent, visited, links, depth+1)
			return
		}
	}

	// No direct causal link found — try owner reference walk.
	if event.OwnerKind != "" && event.OwnerName != "" {
		cb.walkOwnerCause(ctx, event, visited, links, depth)
	}
}

// walkOwnerCause looks for events on the owning resource within the time
// window and, if found, appends a link and recurses.
func (cb *ChainBuilder) walkOwnerCause(
	ctx context.Context,
	event *WideEvent,
	visited map[string]bool,
	links *[]CausalLinkV2,
	depth int,
) {
	windowMs := cb.config.TimeWindow.Milliseconds()
	since := event.Timestamp - windowMs
	until := event.Timestamp

	ownerEvents, err := cb.store.QueryEvents(ctx, EventQuery{
		ClusterID:    event.ClusterID,
		Namespace:    event.ResourceNamespace,
		ResourceKind: event.OwnerKind,
		ResourceName: event.OwnerName,
		Since:        &since,
		Until:        &until,
		Limit:        1,
	})
	if err != nil || len(ownerEvents) == 0 {
		return
	}

	ownerEvent := &ownerEvents[0]
	if visited[ownerEvent.EventID] {
		return
	}

	timeDelta := event.Timestamp - ownerEvent.Timestamp
	boostedConf := cb.applyTimeProximityBoost(0.70, timeDelta)

	link := CausalLinkV2{
		Cause:       cb.eventToCausalNode(ownerEvent),
		Effect:      cb.eventToCausalNode(event),
		Rule:        "owner_reference",
		Confidence:  boostedConf,
		TimeDeltaMs: timeDelta,
	}
	*links = append(*links, link)

	cb.walkCauses(ctx, ownerEvent, visited, links, depth+1)
}

// ---------------------------------------------------------------------------
// Scoring helpers
// ---------------------------------------------------------------------------

// applyTimeProximityBoost returns base confidence boosted by time proximity:
//   - timeDeltaMs <= 10000 → +0.10 (capped at 1.0)
//   - timeDeltaMs <= 60000 → +0.05
//   - otherwise → no boost
func (cb *ChainBuilder) applyTimeProximityBoost(base float64, timeDeltaMs int64) float64 {
	var boosted float64
	if timeDeltaMs <= 10000 {
		boosted = base + 0.10
	} else if timeDeltaMs <= 60000 {
		boosted = base + 0.05
	} else {
		boosted = base
	}
	if boosted > 1.0 {
		boosted = 1.0
	}
	return boosted
}

// computeChainConfidence multiplies all link confidences; if the product is
// below ConfidenceFloor it is clamped up to the floor.
func (cb *ChainBuilder) computeChainConfidence(links []CausalLinkV2) float64 {
	if len(links) == 0 {
		return 0
	}
	product := 1.0
	for _, l := range links {
		product *= l.Confidence
	}
	if product < cb.config.ConfidenceFloor {
		return cb.config.ConfidenceFloor
	}
	return product
}

// ---------------------------------------------------------------------------
// Conversion helpers
// ---------------------------------------------------------------------------

// eventToCausalNode converts a WideEvent into a CausalNode for chain storage.
func (cb *ChainBuilder) eventToCausalNode(e *WideEvent) CausalNode {
	hs := ""
	if e.HealthScore != nil {
		// Map numeric score to label.
		switch {
		case *e.HealthScore >= 0.8:
			hs = "healthy"
		case *e.HealthScore >= 0.5:
			hs = "degraded"
		default:
			hs = "critical"
		}
	}
	return CausalNode{
		ResourceKey:  fmt.Sprintf("%s/%s/%s", e.ResourceNamespace, e.ResourceKind, e.ResourceName),
		Kind:         e.ResourceKind,
		Namespace:    e.ResourceNamespace,
		Name:         e.ResourceName,
		EventReason:  e.Reason,
		EventMessage: e.Message,
		Timestamp:    time.UnixMilli(e.Timestamp),
		HealthStatus: hs,
	}
}

// ruleKindMap maps insight Rule names to the Kubernetes resource Kind they
// primarily affect. This is used by parseInsightResource as a fallback when the
// Detail text does not contain a Kind prefix.
var ruleKindMap = map[string]string{
	"crashLoopDetected":  "Pod",
	"imagePullFailure":   "Pod",
	"oomKillDetected":    "Pod",
	"schedulingFailures": "Pod",
	"restartStorm":       "Pod",
	"cascadingFailures":  "",  // no single resource — skip chain building
	"healthDrift":        "",  // no single resource — skip chain building
}

// parseInsightResource extracts namespace, kind, and name from an Insight so
// that BuildChain can look up events for that resource.
//
// It handles two families of formats:
//
//  1. Legacy / hand-crafted format with an explicit Kind prefix:
//     "Pod default/api-server-7f8d9 is in CrashLoopBackOff"
//     "Deployment production/web-app has insufficient replicas"
//
//  2. Actual formats produced by the built-in insight rules (insights.go):
//     "3 pod(s) in CrashLoopBackOff: default/api-server, default/pod2"
//     "2 pod(s) with image pull failures: default/api-server, prod/worker"
//     "2 pod(s) killed due to memory limits: default/crashy, prod/web"
//     "5 FailedScheduling events … affecting 2 pod(s): default/mypod — …"
//
// For formats that carry no extractable resource (restartStorm, cascadingFailures,
// healthDrift) the function returns ("", "", "") so BuildChain skips chain
// construction gracefully.
func (cb *ChainBuilder) parseInsightResource(insight Insight) (namespace, kind, name string) {
	detail := insight.Detail

	// -------------------------------------------------------------------------
	// Strategy 1: legacy format — explicit Kind prefix before "ns/name"
	// e.g. "Pod default/api-server-7f8d9 is in CrashLoopBackOff"
	// -------------------------------------------------------------------------
	knownKinds := []string{
		"Pod", "Deployment", "ReplicaSet", "StatefulSet", "DaemonSet", "Service",
		"ConfigMap", "Secret", "Job", "CronJob",
	}
	for _, k := range knownKinds {
		idx := strings.Index(detail, k+" ")
		if idx == -1 {
			continue
		}
		rest := detail[idx+len(k)+1:]
		// rest should start with "namespace/name ..."
		end := strings.IndexByte(rest, ' ')
		var resourceRef string
		if end == -1 {
			resourceRef = rest
		} else {
			resourceRef = rest[:end]
		}
		if !strings.Contains(resourceRef, "/") {
			continue // not a ns/name token — keep trying other kinds
		}
		parts := strings.SplitN(resourceRef, "/", 2)
		if len(parts) == 2 {
			return parts[0], k, parts[1]
		}
	}

	// -------------------------------------------------------------------------
	// Strategy 2: actual rule format — "…: ns/name, ns/name2, …"
	// The first resource after the last colon is used.
	// -------------------------------------------------------------------------
	colonIdx := strings.LastIndex(detail, ":")
	if colonIdx != -1 && colonIdx < len(detail)-1 {
		afterColon := strings.TrimSpace(detail[colonIdx+1:])
		// There may be a trailing " — …" clause; take the text up to " —".
		if dashIdx := strings.Index(afterColon, " —"); dashIdx != -1 {
			afterColon = strings.TrimSpace(afterColon[:dashIdx])
		}
		// The first token before a comma is the first resource reference.
		firstRef := afterColon
		if commaIdx := strings.Index(afterColon, ","); commaIdx != -1 {
			firstRef = strings.TrimSpace(afterColon[:commaIdx])
		}
		if strings.Contains(firstRef, "/") {
			parts := strings.SplitN(firstRef, "/", 2)
			if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
				// Determine the Kind from the Rule field.
				k := ruleKindMap[insight.Rule]
				if k == "" {
					return "", "", ""
				}
				return parts[0], k, parts[1]
			}
		}
	}

	return "", "", ""
}

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

// reverseLinks reverses a slice of CausalLinkV2 in-place.
func reverseLinks(links []CausalLinkV2) {
	for i, j := 0, len(links)-1; i < j; i, j = i+1, j-1 {
		links[i], links[j] = links[j], links[i]
	}
}
