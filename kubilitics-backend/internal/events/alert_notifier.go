package events

import (
	"context"
	"fmt"
	"time"

	"github.com/kubilitics/kubilitics-backend/internal/models"
)

// NotifyFunc is the signature of the addon Notifier.Notify method, accepted as
// a function value so the events package does not import the addon/notifications
// package directly.
type NotifyFunc func(ev models.NotifyEvent)

// AlertNotifierAdapter bridges the InsightsEngine to the existing webhook/Slack
// notification system.  It converts an Insight into a models.NotifyEvent and
// dispatches it through the provided NotifyFunc.
type AlertNotifierAdapter struct {
	notify NotifyFunc
}

// NewAlertNotifierAdapter creates an adapter that wraps the given notification
// dispatch function (typically addon/notifications.Notifier.Notify).
func NewAlertNotifierAdapter(notify NotifyFunc) *AlertNotifierAdapter {
	return &AlertNotifierAdapter{notify: notify}
}

// NotifyInsight converts an Insight to a NotifyEvent and fires the notification.
func (a *AlertNotifierAdapter) NotifyInsight(_ context.Context, insight *Insight) error {
	if a.notify == nil {
		return nil
	}

	ev := models.NotifyEvent{
		EventType:  insightEventType(insight.Severity),
		ClusterID:  insight.ClusterID,
		Message:    fmt.Sprintf("%s — %s", insight.Title, insight.Detail),
		OccurredAt: time.UnixMilli(insight.Timestamp).UTC().Format(time.RFC3339),
		// AddonID is not applicable for insight alerts; leave empty.
	}

	a.notify(ev)
	return nil
}

// insightEventType maps insight severity to a notification event type string.
func insightEventType(severity string) string {
	switch severity {
	case "critical":
		return "insight_critical"
	case "warning":
		return "insight_warning"
	default:
		return "insight_info"
	}
}
