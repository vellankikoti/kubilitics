import type { Diagnosis, WarningEvent, ReasonCode } from './types';
import { diagnosePod } from './podDiagnosis';
import { lookupReason } from './reasons';

interface JobLike {
  kind?: string;
  metadata: { name: string; namespace?: string };
  spec?: { completions?: number; backoffLimit?: number };
  status?: {
    succeeded?: number;
    active?: number;
    failed?: number;
    conditions?: Array<{ type: string; status: string; reason?: string; message?: string }>;
  };
}

/**
 * Diagnose a Job. Jobs have a rich `conditions` field that already
 * encodes the success/failure state; we combine that with the worst
 * child pod's diagnosis for full context.
 */
export function diagnoseJob(
  job: JobLike,
  childPods: Parameters<typeof diagnosePod>[0][],
  events: WarningEvent[] = []
): Diagnosis {
  const kind = job.kind ?? 'Job';
  const name = job.metadata?.name ?? 'unknown';
  const namespace = job.metadata?.namespace;

  const completeCondition = job.status?.conditions?.find(c => c.type === 'Complete' && c.status === 'True');
  const failedCondition = job.status?.conditions?.find(c => c.type === 'Failed' && c.status === 'True');

  if (completeCondition) {
    return {
      severity: 'healthy',
      headline: 'Job completed successfully',
      oneLine: `Succeeded: ${job.status?.succeeded ?? 1}${job.spec?.completions ? ` / ${job.spec.completions}` : ''}`,
      reasons: [],
      containers: [],
      conditions: (job.status?.conditions as Diagnosis['conditions']) ?? [],
      recentWarnings: events,
      computedAt: Date.now(),
      kind,
      namespace,
      name,
    };
  }

  if (failedCondition) {
    const failureReason: ReasonCode = lookupReason(failedCondition.reason);
    const childDiagnoses = childPods.map(p => diagnosePod(p, events));
    const childReasons: ReasonCode[] = childDiagnoses.flatMap(d => d.reasons);

    const allReasons = [failureReason, ...childReasons];
    const seen = new Set<string>();
    const deduped = allReasons.filter(r => {
      if (seen.has(r.code)) return false;
      seen.add(r.code);
      return true;
    });

    return {
      severity: 'broken',
      headline: failureReason.title,
      oneLine: failedCondition.message || failureReason.explanation,
      reasons: deduped,
      containers: childDiagnoses.flatMap(d => d.containers),
      conditions: (job.status?.conditions as Diagnosis['conditions']) ?? [],
      recentWarnings: events,
      computedAt: Date.now(),
      kind,
      namespace,
      name,
    };
  }

  if ((job.status?.active ?? 0) > 0) {
    return {
      severity: 'degraded',
      headline: 'Job is running',
      oneLine: `${job.status?.active} active pod${job.status?.active === 1 ? '' : 's'}.`,
      reasons: [],
      containers: [],
      conditions: (job.status?.conditions as Diagnosis['conditions']) ?? [],
      recentWarnings: events,
      computedAt: Date.now(),
      kind,
      namespace,
      name,
    };
  }

  return {
    severity: 'degraded',
    headline: 'Job is starting',
    oneLine: 'Waiting for the first pod.',
    reasons: [],
    containers: [],
    conditions: (job.status?.conditions as Diagnosis['conditions']) ?? [],
    recentWarnings: events,
    computedAt: Date.now(),
    kind,
    namespace,
    name,
  };
}
