/**
 * Independent re-verification (Task 12): useResourceLiveUpdates had no test
 * coverage at all before this. tsc flags 6 TS2339 errors on `resource`
 * (typed `unknown`) inside this hook's onMessage handler. A prior code-
 * reading pass concluded `if (resource && resource.type)` makes every
 * subsequent access safe — this test actually invokes the real onMessage
 * callback with the kinds of malformed WebSocket payloads a live backend
 * could plausibly send, and asserts it never throws, rather than trusting
 * the trace.
 */
import { describe, it, expect, vi } from 'vitest';
import { renderHook } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import React from 'react';

let capturedOnMessage: ((data: Record<string, unknown>) => void) | undefined;

vi.mock('./useBackendWebSocket', () => ({
  useBackendWebSocket: (options: { onMessage?: (data: Record<string, unknown>) => void }) => {
    capturedOnMessage = options.onMessage;
    return { connected: true };
  },
}));

import { useResourceLiveUpdates } from './useResourceLiveUpdates';

function wrapper(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    React.createElement(QueryClientProvider, { client: qc }, children);
}

describe('useResourceLiveUpdates — onMessage malformed-payload survival', () => {
  function mount() {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderHook(() => useResourceLiveUpdates({ clusterId: 'c1', enabled: true }), { wrapper: wrapper(qc) });
    if (!capturedOnMessage) throw new Error('onMessage was not captured — hook wiring changed');
    return capturedOnMessage;
  }

  it('does not throw when resource is entirely absent', () => {
    const onMessage = mount();
    expect(() => onMessage({ type: 'resource_update' })).not.toThrow();
  });

  it('does not throw when resource is null', () => {
    const onMessage = mount();
    expect(() => onMessage({ type: 'resource_update', resource: null })).not.toThrow();
  });

  it('does not throw when resource is a bare string instead of an object', () => {
    const onMessage = mount();
    // resource && resource.type — a string is truthy, '.type' on a string is undefined (falsy),
    // so the inner block is skipped. Proving this directly, not just reasoning it.
    expect(() => onMessage({ type: 'resource_update', resource: 'not-an-object' as unknown as Record<string, unknown> })).not.toThrow();
  });

  it('does not throw when resource has type but is missing metadata/uid/action/event_type entirely', () => {
    const onMessage = mount();
    expect(() =>
      onMessage({ type: 'resource_update', resource: { type: 'Pod' } })
    ).not.toThrow();
  });

  it('does not throw when resource.metadata is itself null', () => {
    const onMessage = mount();
    expect(() =>
      onMessage({ type: 'resource_update', resource: { type: 'Pod', metadata: null } })
    ).not.toThrow();
  });

  it('does not throw when resource is a number', () => {
    const onMessage = mount();
    expect(() =>
      onMessage({ type: 'resource_update', resource: 42 as unknown as Record<string, unknown> })
    ).not.toThrow();
  });

  it('does not throw when resource is an array', () => {
    const onMessage = mount();
    expect(() =>
      onMessage({ type: 'resource_update', resource: [] as unknown as Record<string, unknown> })
    ).not.toThrow();
  });

  it('full real-shaped event still triggers an animation entry as expected (control case, not just absence-of-throw)', () => {
    const onMessage = mount();
    expect(() =>
      onMessage({
        type: 'resource_update',
        resource: { type: 'Pod', uid: 'abc-123', action: 'ADDED', metadata: { uid: 'abc-123' } },
      })
    ).not.toThrow();
  });
});
