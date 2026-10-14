import { describe, it, expect } from 'vitest';
import { diagnoseController } from './controllerDiagnosis';
import {
  runningDeployment,
  brokenDeployment,
  healthyPod,
  crashLoopPod,
} from './__fixtures__/fixtures';

describe('diagnoseController', () => {
  it('healthy deployment with all ready replicas is healthy', () => {
    const d = diagnoseController(runningDeployment(), [healthyPod('web-1'), healthyPod('web-2'), healthyPod('web-3')]);
    expect(d.severity).toBe('healthy');
  });

  it('deployment with one CrashLoopBackOff pod surfaces broken', () => {
    const d = diagnoseController(runningDeployment(), [
      healthyPod('web-1'),
      crashLoopPod('web-2'),
      healthyPod('web-3'),
    ]);
    expect(d.severity).toBe('broken');
    expect(d.reasons[0].code).toBe('CrashLoopBackOff');
    expect(d.relatedPodLink?.name).toBe('web-2');
  });

  it('deployment with 0 pods despite replicas=3 is broken', () => {
    const d = diagnoseController(runningDeployment('web', 3), []);
    expect(d.severity).toBe('broken');
    expect(d.headline.toLowerCase()).toMatch(/replicas|running|no pods/);
  });

  it('deployment with readyReplicas < replicas is at least degraded', () => {
    const d = diagnoseController(brokenDeployment(), [healthyPod('web-1'), healthyPod('web-2')]);
    expect(d.severity === 'degraded' || d.severity === 'broken').toBe(true);
  });

  it('kind is propagated from the controller', () => {
    const d = diagnoseController(runningDeployment(), [healthyPod()]);
    expect(d.kind).toBe('Deployment');
  });

  it('empty pods list + status.replicas=0 + spec.replicas=0 is healthy', () => {
    const scaled = runningDeployment('scaled-down', 0);
    (scaled.status as Record<string, unknown>).replicas = 0;
    (scaled.status as Record<string, unknown>).readyReplicas = 0;
    const d = diagnoseController(scaled, []);
    expect(d.severity).toBe('healthy');
  });

  // Independent re-verification (Task 12): same shared diagnose-entry bug
  // class as podDiagnosis.test.ts — disconnected-cluster pages pass an
  // empty {} resource cast past the type system via `as unknown as`.
  it('does not crash when metadata is entirely absent (disconnected-cluster {} resource)', () => {
    expect(() => diagnoseController({} as unknown as Parameters<typeof diagnoseController>[0], [])).not.toThrow();
  });
});
