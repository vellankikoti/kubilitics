/**
 * Independent re-verification (Task 12): diagnoseWorkload is the exact
 * shared entry point all 7 affected pages call —
 * `diagnoseWorkload({ ...resource, kind: X } as unknown as ..., options)`.
 * When a cluster is disconnected, useResourceDetail's real runtime fallback
 * (`resource ?? ({} as T)`) means `resource` is `{}` — never null/undefined,
 * but also never carrying `.metadata`. The page-level `as unknown as` cast
 * suppresses the TS2339 error this would otherwise raise, which is exactly
 * why a tsc-error-based sweep missed it: the bug lives one function call
 * deeper than any flagged line, behind an explicit type-system bypass.
 *
 * Previously: every one of these 7 calls threw
 * "TypeError: Cannot read properties of undefined (reading 'name')" from
 * inside diagnosePod/diagnoseController/diagnoseJob/diagnoseCronJob's
 * unguarded `resource.metadata.name` access. Fixed by optional-chaining
 * `.metadata?.name`/`.metadata?.namespace` (and podDiagnosis's additional
 * `.metadata?.deletionTimestamp`) in all 4 functions.
 */
import { describe, it, expect } from 'vitest';
import { diagnoseWorkload } from './index';

describe('diagnoseWorkload — the real page call-site shape, for every affected kind', () => {
  const kinds = ['Pod', 'Deployment', 'StatefulSet', 'DaemonSet', 'ReplicaSet', 'Job', 'CronJob'];

  for (const kind of kinds) {
    it(`does not crash for kind=${kind} when the resource is {} (disconnected-cluster fallback)`, () => {
      const resource = { ...{}, kind } as unknown as Parameters<typeof diagnoseWorkload>[0];
      expect(() => diagnoseWorkload(resource, {})).not.toThrow();
    });
  }

  it('unsupported kind still falls through to the safe default branch', () => {
    const resource = { kind: 'SomethingUnknown' } as unknown as Parameters<typeof diagnoseWorkload>[0];
    const d = diagnoseWorkload(resource, {});
    expect(d.severity).toBe('unknown');
    expect(d.name).toBe('unknown');
  });
});
