import type { Diagnosis, WarningEvent } from './types';
import { diagnosePod } from './podDiagnosis';

interface ControllerLike {
  kind?: string;
  metadata: { name: string; namespace?: string };
  spec?: {
    replicas?: number;
    selector?: { matchLabels?: Record<string, string> };
  };
  status?: {
    replicas?: number;
    readyReplicas?: number;
    availableReplicas?: number;
    updatedReplicas?: number;
  };
}

/**
 * Diagnose a Deployment / StatefulSet / DaemonSet / ReplicaSet by walking
 * its child pods and picking the worst one. Controllers don't have their
 * own "state" — their health is derived from (a) controller-level status
 * fields like readyReplicas and (b) the worst child pod.
 */
export function diagnoseController(
  controller: ControllerLike,
  childPods: Parameters<typeof diagnosePod>[0][],
  events: WarningEvent[] = []
): Diagnosis {
  const kind = controller.kind ?? 'Deployment';
  const name = controller.metadata?.name ?? 'unknown';
  const namespace = controller.metadata?.namespace;

  const desiredReplicas = controller.spec?.replicas ?? 0;
  const statusReplicas = controller.status?.replicas ?? 0;
  const readyReplicas = controller.status?.readyReplicas ?? 0;

  // Controller-level zero-pods check: desired > 0 but no pods provided or no status replicas
  if (desiredReplicas > 0 && (childPods.length === 0 || statusReplicas === 0)) {
    return {
      severity: 'broken',
      headline: 'No replicas are running',
      oneLine: `Desired: ${desiredReplicas} replicas, running: 0.`,
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

  // Diagnose each child pod, pick the worst
  if (childPods.length > 0) {
    const podDiagnoses = childPods.map(p => diagnosePod(p, events));
    const worst = podDiagnoses
      .slice()
      .sort((a, b) => severityRank(b.severity) - severityRank(a.severity))[0];

    // Healthy pods but not enough ready replicas → degraded
    if (worst.severity === 'healthy' && readyReplicas < desiredReplicas) {
      return {
        ...worst,
        severity: 'degraded',
        headline: `${readyReplicas}/${desiredReplicas} replicas ready`,
        oneLine: `${desiredReplicas - readyReplicas} replica${desiredReplicas - readyReplicas === 1 ? '' : 's'} not yet ready.`,
        kind,
        namespace,
        name,
      };
    }

    return {
      ...worst,
      kind,
      namespace,
      name,
      relatedPodLink: worst.name
        ? { namespace: worst.namespace ?? namespace ?? 'default', name: worst.name }
        : undefined,
    };
  }

  // No pods and no broken controller status — healthy
  return {
    severity: 'healthy',
    headline: 'Healthy',
    oneLine:
      desiredReplicas === 0
        ? 'Scaled to zero.'
        : `${readyReplicas}/${desiredReplicas} replicas ready`,
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

function severityRank(s: Diagnosis['severity']): number {
  return { healthy: 0, degraded: 1, unknown: 2, broken: 3 }[s];
}
