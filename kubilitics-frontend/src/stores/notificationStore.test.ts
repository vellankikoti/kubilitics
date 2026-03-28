/**
 * Unit tests for src/stores/notificationStore.ts
 *
 * Covers: addNotification, markAsRead, markAllAsRead, dismiss, clearAll,
 * 100-notification cap, unreadCount, byCategory, panel toggle.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest';

// Mock safeLocalStorage before importing the store
vi.mock('@/lib/safeStorage', () => ({
  safeLocalStorage: {
    getItem: (_name: string) => null,
    setItem: (_name: string, _value: string) => {},
    removeItem: (_name: string) => {},
  },
}));

import { useNotificationStore } from './notificationStore';
import type { NotificationCategory, NotificationSeverity } from './notificationStore';

/** Helper to create a minimal notification input. */
function makeNotification(overrides: {
  id?: string;
  title?: string;
  severity?: NotificationSeverity;
  category?: NotificationCategory;
} = {}) {
  return {
    id: overrides.id,
    title: overrides.title ?? 'Test notification',
    severity: overrides.severity ?? ('info' as const),
    category: overrides.category ?? ('cluster' as const),
  };
}

describe('notificationStore', () => {
  beforeEach(() => {
    useNotificationStore.setState({
      notifications: [],
      isPanelOpen: false,
    });
  });

  // ── addNotification ────────────────────────────────────────────────────────

  it('addNotification adds a notification to the array', () => {
    useNotificationStore.getState().addNotification(makeNotification({ title: 'Pod crash' }));
    const { notifications } = useNotificationStore.getState();
    expect(notifications).toHaveLength(1);
    expect(notifications[0].title).toBe('Pod crash');
    expect(notifications[0].read).toBe(false);
  });

  it('addNotification prepends new notifications (newest first)', () => {
    useNotificationStore.getState().addNotification(makeNotification({ title: 'First' }));
    useNotificationStore.getState().addNotification(makeNotification({ title: 'Second' }));
    const { notifications } = useNotificationStore.getState();
    expect(notifications[0].title).toBe('Second');
    expect(notifications[1].title).toBe('First');
  });

  it('addNotification generates an id if not provided', () => {
    useNotificationStore.getState().addNotification(makeNotification());
    const { notifications } = useNotificationStore.getState();
    expect(notifications[0].id).toBeTruthy();
    expect(typeof notifications[0].id).toBe('string');
  });

  it('addNotification uses provided id when given', () => {
    useNotificationStore.getState().addNotification(makeNotification({ id: 'custom-id' }));
    expect(useNotificationStore.getState().notifications[0].id).toBe('custom-id');
  });

  it('addNotification sets timestamp as ISO string', () => {
    useNotificationStore.getState().addNotification(makeNotification());
    const ts = useNotificationStore.getState().notifications[0].timestamp;
    expect(() => new Date(ts).toISOString()).not.toThrow();
  });

  it('addNotification sets read = false by default', () => {
    useNotificationStore.getState().addNotification(makeNotification());
    expect(useNotificationStore.getState().notifications[0].read).toBe(false);
  });

  // ── 100-notification cap ───────────────────────────────────────────────────

  it('enforces maxNotifications cap (100)', () => {
    const store = useNotificationStore.getState();
    // Add 101 notifications
    for (let i = 0; i < 101; i++) {
      store.addNotification(makeNotification({ id: `n-${i}`, title: `Notification ${i}` }));
    }
    const { notifications } = useNotificationStore.getState();
    expect(notifications.length).toBe(100);
    // The newest should be first (n-100), oldest (n-0) should be dropped
    expect(notifications[0].id).toBe('n-100');
  });

  it('cap is exactly 100 notifications', () => {
    const store = useNotificationStore.getState();
    for (let i = 0; i < 100; i++) {
      store.addNotification(makeNotification({ id: `n-${i}` }));
    }
    expect(useNotificationStore.getState().notifications.length).toBe(100);
    // Adding one more should still be 100
    store.addNotification(makeNotification({ id: 'n-100' }));
    expect(useNotificationStore.getState().notifications.length).toBe(100);
  });

  // ── markAsRead ─────────────────────────────────────────────────────────────

  it('markAsRead marks a specific notification as read', () => {
    useNotificationStore.getState().addNotification(makeNotification({ id: 'a' }));
    useNotificationStore.getState().addNotification(makeNotification({ id: 'b' }));
    useNotificationStore.getState().markAsRead('a');

    const { notifications } = useNotificationStore.getState();
    const notifA = notifications.find((n) => n.id === 'a');
    const notifB = notifications.find((n) => n.id === 'b');
    expect(notifA?.read).toBe(true);
    expect(notifB?.read).toBe(false);
  });

  it('markAsRead is a no-op for non-existent id', () => {
    useNotificationStore.getState().addNotification(makeNotification({ id: 'a' }));
    useNotificationStore.getState().markAsRead('nonexistent');
    expect(useNotificationStore.getState().notifications[0].read).toBe(false);
  });

  // ── markAllAsRead ──────────────────────────────────────────────────────────

  it('markAllAsRead marks all notifications as read', () => {
    useNotificationStore.getState().addNotification(makeNotification({ id: 'a' }));
    useNotificationStore.getState().addNotification(makeNotification({ id: 'b' }));
    useNotificationStore.getState().addNotification(makeNotification({ id: 'c' }));

    useNotificationStore.getState().markAllAsRead();

    const { notifications } = useNotificationStore.getState();
    expect(notifications.every((n) => n.read)).toBe(true);
  });

  it('markAllAsRead works on empty array', () => {
    useNotificationStore.getState().markAllAsRead();
    expect(useNotificationStore.getState().notifications).toEqual([]);
  });

  // ── dismiss ────────────────────────────────────────────────────────────────

  it('dismiss removes a specific notification', () => {
    useNotificationStore.getState().addNotification(makeNotification({ id: 'a' }));
    useNotificationStore.getState().addNotification(makeNotification({ id: 'b' }));

    useNotificationStore.getState().dismiss('a');

    const { notifications } = useNotificationStore.getState();
    expect(notifications).toHaveLength(1);
    expect(notifications[0].id).toBe('b');
  });

  it('dismiss is a no-op for non-existent id', () => {
    useNotificationStore.getState().addNotification(makeNotification({ id: 'a' }));
    useNotificationStore.getState().dismiss('nonexistent');
    expect(useNotificationStore.getState().notifications).toHaveLength(1);
  });

  // ── clearAll ───────────────────────────────────────────────────────────────

  it('clearAll empties all notifications', () => {
    useNotificationStore.getState().addNotification(makeNotification({ id: 'a' }));
    useNotificationStore.getState().addNotification(makeNotification({ id: 'b' }));

    useNotificationStore.getState().clearAll();
    expect(useNotificationStore.getState().notifications).toEqual([]);
  });

  it('clearAll on empty array is a no-op', () => {
    useNotificationStore.getState().clearAll();
    expect(useNotificationStore.getState().notifications).toEqual([]);
  });

  // ── unreadCount ────────────────────────────────────────────────────────────

  it('unreadCount returns correct count with no notifications', () => {
    expect(useNotificationStore.getState().unreadCount()).toBe(0);
  });

  it('unreadCount returns correct count with mixed read/unread', () => {
    useNotificationStore.getState().addNotification(makeNotification({ id: 'a' }));
    useNotificationStore.getState().addNotification(makeNotification({ id: 'b' }));
    useNotificationStore.getState().addNotification(makeNotification({ id: 'c' }));
    useNotificationStore.getState().markAsRead('b');

    expect(useNotificationStore.getState().unreadCount()).toBe(2);
  });

  it('unreadCount returns 0 after markAllAsRead', () => {
    useNotificationStore.getState().addNotification(makeNotification({ id: 'a' }));
    useNotificationStore.getState().addNotification(makeNotification({ id: 'b' }));
    useNotificationStore.getState().markAllAsRead();

    expect(useNotificationStore.getState().unreadCount()).toBe(0);
  });

  // ── byCategory ─────────────────────────────────────────────────────────────

  it('byCategory filters notifications by category', () => {
    useNotificationStore.getState().addNotification(makeNotification({ id: 'a', category: 'cluster' }));
    useNotificationStore.getState().addNotification(makeNotification({ id: 'b', category: 'ai' }));
    useNotificationStore.getState().addNotification(makeNotification({ id: 'c', category: 'cluster' }));

    const clusterNotifs = useNotificationStore.getState().byCategory('cluster');
    expect(clusterNotifs).toHaveLength(2);
    expect(clusterNotifs.every((n) => n.category === 'cluster')).toBe(true);

    const aiNotifs = useNotificationStore.getState().byCategory('ai');
    expect(aiNotifs).toHaveLength(1);

    const systemNotifs = useNotificationStore.getState().byCategory('system');
    expect(systemNotifs).toHaveLength(0);
  });

  // ── Panel toggle ───────────────────────────────────────────────────────────

  it('togglePanel toggles isPanelOpen', () => {
    expect(useNotificationStore.getState().isPanelOpen).toBe(false);
    useNotificationStore.getState().togglePanel();
    expect(useNotificationStore.getState().isPanelOpen).toBe(true);
    useNotificationStore.getState().togglePanel();
    expect(useNotificationStore.getState().isPanelOpen).toBe(false);
  });

  it('setPanelOpen sets isPanelOpen directly', () => {
    useNotificationStore.getState().setPanelOpen(true);
    expect(useNotificationStore.getState().isPanelOpen).toBe(true);
    useNotificationStore.getState().setPanelOpen(false);
    expect(useNotificationStore.getState().isPanelOpen).toBe(false);
  });
});
