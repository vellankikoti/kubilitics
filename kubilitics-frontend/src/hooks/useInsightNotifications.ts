/**
 * useInsightNotifications
 *
 * Polls active insights via useActiveInsights() and creates in-app
 * notifications for any newly detected insights so they appear in the
 * Notification Center without waiting for a page refresh.
 *
 * Should be mounted once in a top-level component (e.g. AppLayout).
 */
import { useEffect, useRef } from 'react';
import { useActiveInsights } from '@/hooks/useEventsIntelligence';
import { useNotificationStore, type NotificationSeverity } from '@/stores/notificationStore';

function mapSeverity(severity: string): NotificationSeverity {
  switch (severity) {
    case 'critical':
      return 'error';
    case 'warning':
      return 'warning';
    default:
      return 'info';
  }
}

export function useInsightNotifications() {
  const { data: insights } = useActiveInsights();
  const addNotification = useNotificationStore((s) => s.addNotification);
  // Use a ref so we don't trigger re-renders when the set changes, and the
  // set persists across renders without being a dependency.
  const seenIds = useRef<Set<string>>(new Set());

  useEffect(() => {
    if (!insights || insights.length === 0) return;

    for (const insight of insights) {
      if (seenIds.current.has(insight.insight_id)) continue;

      seenIds.current.add(insight.insight_id);

      addNotification({
        id: `insight-${insight.insight_id}`,
        title: insight.title,
        description: insight.detail,
        severity: mapSeverity(insight.severity),
        category: 'cluster',
      });
    }
  }, [insights, addNotification]);
}
