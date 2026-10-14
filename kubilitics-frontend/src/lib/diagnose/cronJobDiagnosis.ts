import type { Diagnosis, WarningEvent } from './types';
import { diagnoseJob } from './jobDiagnosis';

interface CronJobLike {
  kind?: string;
  metadata: { name: string; namespace?: string };
  spec?: { schedule?: string; suspend?: boolean };
  status?: {
    lastSuccessfulTime?: string;
    lastScheduleTime?: string;
    active?: Array<unknown>;
  };
}

/**
 * Diagnose a CronJob. A CronJob's health is primarily derived from its
 * most recent Job execution. We also respect the suspend flag.
 */
export function diagnoseCronJob(
  cronjob: CronJobLike,
  recentJobs: Parameters<typeof diagnoseJob>[0][],
  events: WarningEvent[] = []
): Diagnosis {
  const kind = cronjob.kind ?? 'CronJob';
  const name = cronjob.metadata?.name ?? 'unknown';
  const namespace = cronjob.metadata?.namespace;

  if (cronjob.spec?.suspend === true) {
    return {
      severity: 'degraded',
      headline: 'CronJob is suspended',
      oneLine: 'spec.suspend is true — no jobs will be created until this is cleared.',
      reasons: [],
      containers: [],
      conditions: [],
      recentWarnings: events,
      computedAt: Date.now(),
      kind,
      namespace,
      name,
    };
  }

  if (recentJobs.length > 0) {
    const latest = recentJobs[0];
    const jobDiagnosis = diagnoseJob(latest, [], events);
    return {
      ...jobDiagnosis,
      kind,
      namespace,
      name,
      headline:
        jobDiagnosis.severity === 'healthy'
          ? 'CronJob is healthy'
          : jobDiagnosis.severity === 'broken'
          ? `Latest run failed: ${jobDiagnosis.headline.toLowerCase()}`
          : jobDiagnosis.headline,
    };
  }

  if (cronjob.status?.lastSuccessfulTime) {
    return {
      severity: 'healthy',
      headline: 'CronJob is healthy',
      oneLine: `Last successful run: ${cronjob.status.lastSuccessfulTime}`,
      reasons: [],
      containers: [],
      conditions: [],
      recentWarnings: events,
      computedAt: Date.now(),
      kind,
      namespace,
      name,
    };
  }

  return {
    severity: 'unknown',
    headline: 'CronJob has not run yet',
    oneLine: `Schedule: ${cronjob.spec?.schedule ?? 'unknown'}`,
    reasons: [],
    containers: [],
    conditions: [],
    recentWarnings: events,
    computedAt: Date.now(),
    kind,
    namespace,
    name,
  };
}
