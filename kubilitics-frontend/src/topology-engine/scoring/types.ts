import type { KubernetesKind } from '../types/topology.types';

/** Individual strategy scores for a single node */
export interface StrategyScores {
  pageRank: number;
  blastRadius: number;
  trafficBetweenness: number;
  structuralRisk: number;
}

/** Final criticality result for a single node */
export interface CriticalityResult {
  nodeId: string;
  score: number; // 0-100 unified score
  tier: CriticalityTier;
  strategies: StrategyScores;
}

export type CriticalityTier = 'critical' | 'high' | 'moderate' | 'low';

/** Weights for the unified ensemble */
export interface EnsembleWeights {
  pageRank: number;
  blastRadius: number;
  trafficBetweenness: number;
  structuralRisk: number;
}

export const DEFAULT_ENSEMBLE_WEIGHTS: EnsembleWeights = {
  pageRank: 0.25,
  blastRadius: 0.30,
  trafficBetweenness: 0.20,
  structuralRisk: 0.25,
};

/** Resource type base weights for structural risk scoring */
export const RESOURCE_TYPE_WEIGHTS: Partial<Record<KubernetesKind, number>> = {
  Node: 35,
  Ingress: 30,
  Service: 24,
  StatefulSet: 25,
  PersistentVolume: 22,
  Deployment: 18,
  DaemonSet: 15,
  Secret: 12,
  CronJob: 10,
  ConfigMap: 8,
  ReplicaSet: 6,
  Pod: 5,
  PodGroup: 5,
  Job: 8,
  Container: 3,
  Namespace: 10,
  PersistentVolumeClaim: 15,
  StorageClass: 12,
  ServiceAccount: 8,
  Role: 6,
  ClusterRole: 10,
  RoleBinding: 5,
  ClusterRoleBinding: 8,
  NetworkPolicy: 7,
  ResourceQuota: 5,
  LimitRange: 4,
  HorizontalPodAutoscaler: 10,
  Endpoints: 5,
  EndpointSlice: 5,
};

/** Relationship weights for PageRank edge weighting */
export const RELATIONSHIP_WEIGHTS: Record<string, number> = {
  owns: 1.0,
  selects: 0.9,
  scheduled_on: 0.9,
  routes: 0.85,
  runs: 0.85,
  exposes: 0.8,
  backed_by: 0.75,
  references: 0.7,
  mounts: 0.6,
  configures: 0.6,
  contains: 0.3, // dampened — Namespace→everything would dominate
  stores: 0.5,
  permits: 0.5,
  limits: 0.5,
  manages: 0.5,
};
