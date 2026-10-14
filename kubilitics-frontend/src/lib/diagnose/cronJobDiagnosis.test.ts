import { describe, it, expect } from 'vitest';
import { diagnoseCronJob } from './cronJobDiagnosis';
import { suspendedCronJob, completedJob, failedJob } from './__fixtures__/fixtures';

describe('diagnoseCronJob', () => {
  it('suspended cronjob is degraded', () => {
    const d = diagnoseCronJob(suspendedCronJob(), []);
    expect(d.severity).toBe('degraded');
    expect(d.headline.toLowerCase()).toContain('suspend');
  });

  it('cronjob with a recently completed job is healthy', () => {
    const cj = {
      kind: 'CronJob',
      metadata: { name: 'nightly', namespace: 'default' },
      spec: { schedule: '0 0 * * *' },
      status: { lastSuccessfulTime: new Date().toISOString() },
    };
    const d = diagnoseCronJob(cj, [completedJob('nightly-1')]);
    expect(d.severity).toBe('healthy');
  });

  it('cronjob whose latest job failed is broken', () => {
    const cj = {
      kind: 'CronJob',
      metadata: { name: 'nightly', namespace: 'default' },
      spec: { schedule: '0 0 * * *' },
      status: {},
    };
    const d = diagnoseCronJob(cj, [failedJob('nightly-bad')]);
    expect(d.severity).toBe('broken');
  });

  // Independent re-verification (Task 12): same shared diagnose-entry bug
  // class as podDiagnosis.test.ts — disconnected-cluster pages pass an
  // empty {} resource cast past the type system via `as unknown as`.
  it('does not crash when metadata is entirely absent (disconnected-cluster {} resource)', () => {
    expect(() => diagnoseCronJob({} as unknown as Parameters<typeof diagnoseCronJob>[0], [])).not.toThrow();
  });
});
