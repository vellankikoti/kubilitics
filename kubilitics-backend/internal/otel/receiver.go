package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/kubilitics/kubilitics-backend/internal/events"
)

// ---------------------------------------------------------------------------
// OTLP JSON structs (lightweight — no proto dependency)
// ---------------------------------------------------------------------------

// OTLPTraceRequest is the top-level OTLP/HTTP JSON trace export request.
type OTLPTraceRequest struct {
	ResourceSpans []ResourceSpans `json:"resourceSpans"`
}

// ResourceSpans groups spans by resource (service).
type ResourceSpans struct {
	Resource   Resource     `json:"resource"`
	ScopeSpans []ScopeSpans `json:"scopeSpans"`
}

// Resource holds resource-level attributes (service.name, k8s.* etc.).
type Resource struct {
	Attributes []Attribute `json:"attributes"`
}

// ScopeSpans groups spans by instrumentation scope.
type ScopeSpans struct {
	Spans []OTLPSpan `json:"spans"`
}

// OTLPSpan is a single span in the OTLP JSON format.
type OTLPSpan struct {
	TraceID           string      `json:"traceId"`
	SpanID            string      `json:"spanId"`
	ParentSpanID      string      `json:"parentSpanId"`
	Name              string      `json:"name"`
	Kind              int         `json:"kind"`
	StartTimeUnixNano string      `json:"startTimeUnixNano"`
	EndTimeUnixNano   string      `json:"endTimeUnixNano"`
	Attributes        []Attribute `json:"attributes"`
	Status            SpanStatus  `json:"status"`
	Events            []SpanEvent `json:"events"`
}

// Attribute is an OTLP key-value attribute.
type Attribute struct {
	Key   string         `json:"key"`
	Value AttributeValue `json:"value"`
}

// AttributeValue holds one of the OTLP value types.
type AttributeValue struct {
	StringValue string `json:"stringValue,omitempty"`
	IntValue    string `json:"intValue,omitempty"`
	BoolValue   bool   `json:"boolValue,omitempty"`
}

// SpanStatus holds the span status code and optional message.
type SpanStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SpanEvent is a timestamped annotation on a span.
type SpanEvent struct {
	Name              string      `json:"name"`
	TimeUnixNano      string      `json:"timeUnixNano"`
	Attributes        []Attribute `json:"attributes"`
}

// ---------------------------------------------------------------------------
// Receiver
// ---------------------------------------------------------------------------

// Receiver accepts OTLP trace data and persists it.
type Receiver struct {
	store            *Store
	defaultClusterID string // fallback for single-cluster desktop setups
}

// NewReceiver creates a new OTLP trace receiver. The defaultClusterID is used
// as a fallback when spans arrive without a kubilitics.cluster.id attribute
// (common in single-cluster desktop setups).
func NewReceiver(store *Store, defaultClusterID string) *Receiver {
	return &Receiver{store: store, defaultClusterID: defaultClusterID}
}

// SetDefaultClusterID updates the fallback cluster ID. This is called when the
// active cluster changes so that spans without an explicit cluster attribute
// are attributed to the correct cluster.
func (r *Receiver) SetDefaultClusterID(id string) {
	r.defaultClusterID = id
}

// ProcessTraces parses an OTLP JSON request and stores the spans.
// The clusterIDHint is an optional fallback extracted from the
// X-Kubilitics-Cluster-Id HTTP header; it is used when spans lack a
// kubilitics.cluster.id resource attribute.
func (r *Receiver) ProcessTraces(ctx context.Context, req *OTLPTraceRequest, clusterIDHint string) error {
	var allSpans []Span

	// Track trace-level aggregates for summary upsert.
	type traceAgg struct {
		rootService   string
		rootOperation string
		startTime     int64
		endTime       int64
		spanCount     int
		errorCount    int
		services      map[string]bool
		clusterID     string
		hasRoot       bool
	}
	traceMap := make(map[string]*traceAgg)

	for _, rs := range req.ResourceSpans {
		// Extract resource-level attributes
		resAttrs := attributeMap(rs.Resource.Attributes)

		serviceName := resAttrs["service.name"]
		k8sPod := resAttrs["k8s.pod.name"]
		k8sNamespace := resAttrs["k8s.namespace.name"]
		k8sNode := resAttrs["k8s.node.name"]
		k8sContainer := resAttrs["k8s.container.name"]
		k8sDeployment := resAttrs["k8s.deployment.name"]

		for _, ss := range rs.ScopeSpans {
			for _, os := range ss.Spans {
				startNs := parseNano(os.StartTimeUnixNano)
				endNs := parseNano(os.EndTimeUnixNano)
				durationNs := endNs - startNs
				if durationNs < 0 {
					durationNs = 0
				}

				// Extract span-level attributes
				spanAttrs := attributeMap(os.Attributes)

				// Merge resource + span attributes for full attribute JSON
				allAttrs := make(map[string]string, len(resAttrs)+len(spanAttrs))
				for k, v := range resAttrs {
					allAttrs[k] = v
				}
				for k, v := range spanAttrs {
					allAttrs[k] = v
				}
				attrsJSON, _ := json.Marshal(allAttrs)

				// Serialize span events
				eventsJSON, _ := json.Marshal(os.Events)

				// Extract HTTP fields
				httpMethod := firstNonEmpty(spanAttrs["http.method"], spanAttrs["http.request.method"])
				httpURL := firstNonEmpty(spanAttrs["http.url"], spanAttrs["url.full"])
				httpRoute := spanAttrs["http.route"]
				var httpStatusCode *int
				if sc := firstNonEmpty(spanAttrs["http.status_code"], spanAttrs["http.response.status_code"]); sc != "" {
					if v, err := strconv.Atoi(sc); err == nil {
						httpStatusCode = &v
					}
				}

				// Extract DB fields
				dbSystem := spanAttrs["db.system"]
				dbStatement := spanAttrs["db.statement"]

				// Extract user
				userID := firstNonEmpty(spanAttrs["enduser.id"], spanAttrs["user.id"])

				// Determine cluster ID from resource attributes (convention: kubilitics.cluster.id)
				clusterID := firstNonEmpty(resAttrs["kubilitics.cluster.id"], resAttrs["k8s.cluster.uid"])

				// Fallback chain: header hint → default cluster → "unknown"
				if clusterID == "" && clusterIDHint != "" {
					clusterID = clusterIDHint
				}
				if clusterID == "" && r.defaultClusterID != "" {
					clusterID = r.defaultClusterID
				}
				if clusterID == "" {
					log.Printf("[otel/receiver] span %s has no cluster ID, using 'unknown'", os.SpanID)
					clusterID = "unknown"
				}

				span := Span{
					SpanID:         os.SpanID,
					TraceID:        os.TraceID,
					ParentSpanID:   os.ParentSpanID,
					ServiceName:    serviceName,
					OperationName:  os.Name,
					SpanKind:       spanKindString(os.Kind),
					StartTime:      startNs,
					EndTime:        endNs,
					DurationNs:     durationNs,
					StatusCode:     statusCodeString(os.Status.Code),
					StatusMessage:  os.Status.Message,
					HTTPMethod:     httpMethod,
					HTTPURL:        httpURL,
					HTTPStatusCode: httpStatusCode,
					HTTPRoute:      httpRoute,
					DBSystem:       dbSystem,
					DBStatement:    dbStatement,
					K8sPodName:     k8sPod,
					K8sNamespace:   k8sNamespace,
					K8sNodeName:    k8sNode,
					K8sContainer:   k8sContainer,
					K8sDeployment:  k8sDeployment,
					UserID:         userID,
					ClusterID:      clusterID,
					Attributes:     events.JSONText(attrsJSON),
					Events:         events.JSONText(eventsJSON),
					LinkedEventIDs: events.JSONText("[]"),
				}
				allSpans = append(allSpans, span)

				// Aggregate trace summary
				agg, ok := traceMap[os.TraceID]
				if !ok {
					agg = &traceAgg{
						startTime: startNs,
						endTime:   endNs,
						services:  make(map[string]bool),
						clusterID: clusterID,
					}
					traceMap[os.TraceID] = agg
				}
				agg.spanCount++
				if span.StatusCode == "ERROR" {
					agg.errorCount++
				}
				if serviceName != "" {
					agg.services[serviceName] = true
				}
				if startNs < agg.startTime {
					agg.startTime = startNs
				}
				if endNs > agg.endTime {
					agg.endTime = endNs
				}
				// Root span: no parent
				if os.ParentSpanID == "" {
					agg.rootService = serviceName
					agg.rootOperation = os.Name
					agg.hasRoot = true
				}
				if clusterID != "" {
					agg.clusterID = clusterID
				}
			}
		}
	}

	// Persist spans
	if err := r.store.InsertSpans(ctx, allSpans); err != nil {
		return fmt.Errorf("insert spans: %w", err)
	}

	// Upsert trace summaries
	now := time.Now().UnixNano()
	for traceID, agg := range traceMap {
		serviceList := make([]string, 0, len(agg.services))
		for svc := range agg.services {
			serviceList = append(serviceList, svc)
		}
		servicesJSON, _ := json.Marshal(serviceList)

		status := "OK"
		if agg.errorCount > 0 {
			status = "ERROR"
		}

		summary := &TraceSummary{
			TraceID:       traceID,
			RootService:   agg.rootService,
			RootOperation: agg.rootOperation,
			StartTime:     agg.startTime,
			DurationNs:    agg.endTime - agg.startTime,
			SpanCount:     agg.spanCount,
			ErrorCount:    agg.errorCount,
			ServiceCount:  len(agg.services),
			Status:        status,
			ClusterID:     agg.clusterID,
			Services:      events.JSONText(servicesJSON),
			UpdatedAt:     now,
		}
		if err := r.store.InsertTraceSummary(ctx, summary); err != nil {
			return fmt.Errorf("upsert trace summary %s: %w", traceID, err)
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// attributeMap converts a slice of OTLP Attributes to a string map.
func attributeMap(attrs []Attribute) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		if a.Value.StringValue != "" {
			m[a.Key] = a.Value.StringValue
		} else if a.Value.IntValue != "" {
			m[a.Key] = a.Value.IntValue
		} else if a.Value.BoolValue {
			m[a.Key] = "true"
		}
	}
	return m
}

// parseNano parses a nanosecond timestamp string to int64.
func parseNano(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// spanKindString maps OTLP SpanKind int to human string.
func spanKindString(kind int) string {
	switch kind {
	case 1:
		return "internal"
	case 2:
		return "server"
	case 3:
		return "client"
	case 4:
		return "producer"
	case 5:
		return "consumer"
	default:
		return ""
	}
}

// statusCodeString maps OTLP StatusCode int to human string.
func statusCodeString(code int) string {
	switch code {
	case 0:
		return "UNSET"
	case 1:
		return "OK"
	case 2:
		return "ERROR"
	default:
		return "UNSET"
	}
}

// firstNonEmpty returns the first non-empty string from the arguments.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
