import { describe, it, expect } from 'vitest';
import { diagnoseJob } from './jobDiagnosis';
import { completedJob, failedJob, oomKilledPod } from './__fixtures__/fixtures';

describe('diagnoseJob', () => {
  it('completed job is healthy', () => {
    const d = diagnoseJob(completedJob(), []);
    expect(d.severity).toBe('healthy');
    expect(d.headline.toLowerCase()).toContain('complet');
  });

  it('failed job with OOMKilled pod surfaces OOMKilled', () => {
    const d = diagnoseJob(failedJob(), [oomKilledPod('migration-abc')]);
    expect(d.severity).toBe('broken');
    const codes = d.reasons.map(r => r.code);
    expect(codes).toContain('OOMKilled');
  });

  it('active job with running pod is degraded (in progress)', () => {
    const job = {
      kind: 'Job',
      metadata: { name: 'in-progress', namespace: 'default' },
      spec: { completions: 1 },
      status: { succeeded: 0, active: 1, conditions: [] },
    };
    const d = diagnoseJob(job, []);
    expect(d.severity).toBe('degraded');
  });

  it('BackoffLimitExceeded surfaces with that reason', () => {
    const d = diagnoseJob(failedJob(), []);
    expect(d.severity).toBe('broken');
    expect(d.reasons[0].code).toBe('BackoffLimitExceeded');
  });

  it('job kind is propagated', () => {
    const d = diagnoseJob(completedJob(), []);
    expect(d.kind).toBe('Job');
  });

  // Independent re-verification (Task 12): same shared diagnose-entry bug
  // class as podDiagnosis.test.ts — disconnected-cluster pages pass an
  // empty {} resource cast past the type system via `as unknown as`.
  it('does not crash when metadata is entirely absent (disconnected-cluster {} resource)', () => {
    expect(() => diagnoseJob({} as unknown as Parameters<typeof diagnoseJob>[0], [])).not.toThrow();
  });
});
