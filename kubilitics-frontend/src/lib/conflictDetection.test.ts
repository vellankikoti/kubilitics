/**
 * Unit tests for src/lib/conflictDetection.ts
 *
 * Covers: isConflictError with BackendApiError, generic Error messages,
 * non-conflict errors, and non-Error values.
 */
import { describe, it, expect, vi } from 'vitest';

// Use vi.hoisted so the class is available when vi.mock factory runs (hoisted).
const { MockBackendApiError } = vi.hoisted(() => {
  class MockBackendApiError extends Error {
    status: number;
    body?: string;
    requestId?: string;
    constructor(message: string, status: number, body?: string, requestId?: string) {
      super(message);
      this.name = 'BackendApiError';
      this.status = status;
      this.body = body;
      this.requestId = requestId;
    }
  }
  return { MockBackendApiError };
});

vi.mock('@/services/backendApiClient', () => ({
  BackendApiError: MockBackendApiError,
}));

import { isConflictError } from './conflictDetection';

describe('isConflictError', () => {
  // ── BackendApiError with status 409 ────────────────────────────────────────

  it('returns true for BackendApiError with status 409', () => {
    const err = new MockBackendApiError('Conflict', 409, '{"reason":"Conflict"}');
    expect(isConflictError(err)).toBe(true);
  });

  it('returns false for BackendApiError with status 400', () => {
    const err = new MockBackendApiError('Bad Request', 400);
    expect(isConflictError(err)).toBe(false);
  });

  it('returns false for BackendApiError with status 500', () => {
    const err = new MockBackendApiError('Internal Server Error', 500);
    expect(isConflictError(err)).toBe(false);
  });

  it('returns false for BackendApiError with status 404', () => {
    const err = new MockBackendApiError('Not Found', 404);
    expect(isConflictError(err)).toBe(false);
  });

  // ── Generic Error with conflict message strings ────────────────────────────

  it('returns true for Error containing "409" in message', () => {
    const err = new Error('Kubernetes API error: 409 - conflict');
    expect(isConflictError(err)).toBe(true);
  });

  it('returns true for Error containing "conflict" in message (case-insensitive)', () => {
    const err = new Error('Resource Conflict detected');
    expect(isConflictError(err)).toBe(true);
  });

  it('returns true for Error containing "the object has been modified"', () => {
    const err = new Error('the object has been modified; please apply your changes to the latest version');
    expect(isConflictError(err)).toBe(true);
  });

  it('returns true for Error with uppercase "THE OBJECT HAS BEEN MODIFIED"', () => {
    const err = new Error('THE OBJECT HAS BEEN MODIFIED');
    expect(isConflictError(err)).toBe(true);
  });

  // ── Non-conflict errors ────────────────────────────────────────────────────

  it('returns false for generic Error without conflict indicators', () => {
    const err = new Error('Something went wrong');
    expect(isConflictError(err)).toBe(false);
  });

  it('returns false for Error with 404 status message (no 409)', () => {
    const err = new Error('Not Found: 404');
    expect(isConflictError(err)).toBe(false);
  });

  it('returns false for Error with 500 status message', () => {
    const err = new Error('Kubernetes API error: 500 - Internal Server Error');
    expect(isConflictError(err)).toBe(false);
  });

  // ── Non-Error values ───────────────────────────────────────────────────────

  it('returns false for null', () => {
    expect(isConflictError(null)).toBe(false);
  });

  it('returns false for undefined', () => {
    expect(isConflictError(undefined)).toBe(false);
  });

  it('returns false for a string', () => {
    expect(isConflictError('conflict')).toBe(false);
  });

  it('returns false for a number', () => {
    expect(isConflictError(409)).toBe(false);
  });

  it('returns false for a plain object', () => {
    expect(isConflictError({ status: 409, message: 'conflict' })).toBe(false);
  });
});
