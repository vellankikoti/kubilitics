/**
 * Kubernetes-specific tooltip content for Mi, m, QoS, Image Pull Policy, probes, tolerations, volumes.
 * Used in DetailRow and elsewhere so every K8s field can have an insightful tooltip.
 */

export const TOOLTIP_MEMORY_MIB =
  'Mebibytes (1024-based). 1 Mi ≈ 1.05 MB. Kubernetes uses Mi for memory.';

export const TOOLTIP_MEMORY_GIB =
  'Gibibytes (1024³ bytes). 1 Gi ≈ 1.07 GB.';

export const TOOLTIP_MEMORY_KIB =
  'Kibibytes (1024 bytes). 1 Ki ≈ 1.02 KB.';

export const TOOLTIP_CPU_M =
  'Millicores. 1000m = 1 CPU core. 50m = 5% of one core.';

export const TOOLTIP_QOS: Record<string, string> = {
  Guaranteed:
    'Pod has requests and limits set for all resources, and they are equal. Highest priority; last to be evicted.',
  Burstable:
    'Pod has at least one resource request or limit. Medium priority under resource pressure.',
  BestEffort:
    'Pod has no resource requests or limits. First to be evicted when node is under pressure.',
};

export const TOOLTIP_IMAGE_PULL_POLICY: Record<string, string> = {
  IfNotPresent: 'Pull image only if not present on the node. Default when :latest is not used.',
  Always: 'Always pull the image. Used when tag is :latest or you want fresh images.',
  Never: 'Use only the image already on the node. Never pull.',
};

export const TOOLTIP_PROBE_DELAY = 'Delay before the first probe after container starts.';
export const TOOLTIP_PROBE_TIMEOUT = 'Seconds to wait for a probe response.';
export const TOOLTIP_PROBE_PERIOD = 'How often to run the probe.';
export const TOOLTIP_PROBE_SUCCESS = 'Consecutive successes required to pass.';
export const TOOLTIP_PROBE_FAILURE = 'Consecutive failures before marking unhealthy.';

export const TOOLTIP_TOLERATION_EFFECT: Record<string, string> = {
  NoSchedule: 'Pods that do not tolerate this taint are not scheduled onto the node.',
  PreferNoSchedule: 'Scheduler tries to avoid placing pods that do not tolerate this taint.',
  NoExecute: 'Pods that do not tolerate are evicted (and not scheduled) after tolerationSeconds.',
};

export const TOOLTIP_TOLERATION_SECONDS =
  'Pod evicted after this many seconds if taint persists (NoExecute).';

export const TOOLTIP_VOLUME_KIND: Record<string, string> = {
  ConfigMap: 'Mount key-value data from a ConfigMap as files.',
  Secret: 'Mount sensitive data from a Secret as files.',
  PVC: 'PersistentVolumeClaim: persistent storage bound to a PersistentVolume.',
  EmptyDir: 'Temporary directory; empty when pod starts, shared between containers, lost when pod is removed.',
  Projected: 'Project multiple volume sources (e.g. downwardAPI, serviceAccountToken, configMap) into one directory.',
};

export const TOOLTIP_VOLUME_DEFAULT_MODE =
  'File permission mode (octal). e.g. 420 = 0644 (owner rw, group r, others r).';

export const TOOLTIP_RESTART_POLICY =
  'Always: restart container when it exits. OnFailure: restart only on non-zero exit. Never: never restart.';

export const TOOLTIP_DNS_POLICY =
  'ClusterFirst: cluster DNS first, fallback to upstream. ClusterFirstWithHostNet: same with hostNetwork. Default: same as ClusterFirst. None: no DNS.';

export const TOOLTIP_TERMINATION_GRACE =
  'Seconds to wait after SIGTERM before SIGKILL. Gives the container time to shut down cleanly.';

export const TOOLTIP_PRIORITY =
  '0 = default priority. Higher values = higher scheduling priority when resources are scarce.';

export const TOOLTIP_CONTAINER_ID =
  'Runtime container ID (e.g. containerd, docker).';

export const TOOLTIP_MOUNT_READONLY = 'Mount is read-only.';

export const TOOLTIP_HIGH_RESTART =
  'High restart count may indicate crashes, OOMKilled, or failing health checks.';

export const TOOLTIP_CONTAINER_CPU_USAGE_PCT =
  'Current CPU usage as a percentage of this container\'s CPU limit. From Metrics Server. Same source as Pod list.';

export const TOOLTIP_CONTAINER_MEMORY_USAGE_PCT =
  'Current memory usage as a percentage of this container\'s memory limit. From Metrics Server. Same source as Pod list.';

export const TOOLTIP_READY =
  'Container passes readiness probe and can receive traffic.';

export const TOOLTIP_LAST_EXIT =
  'Last time the container exited (reason and exit code). Check logs for details.';

export const TOOLTIP_METRICS_CPU_USAGE =
  'How much processing power this pod is using right now. Measured in "millicores" (m) — 1000m = 1 full CPU core. Example: 50m means the pod is using 5% of one CPU core.';

export const TOOLTIP_METRICS_MEMORY_USAGE =
  'How much RAM this pod is using right now. Measured in MiB (Mebibytes). Example: 128Mi means the pod is using about 128 MB of memory. If this keeps growing, the pod might get killed (OOMKilled).';

export const TOOLTIP_METRICS_LIST_VS_DETAIL =
  'These are live resource usage numbers from the Kubernetes Metrics Server running in your cluster. They update every ~60 seconds.';

export const TOOLTIP_METRICS_NETWORK_IO =
  'Total data sent and received by this pod over the network since it started, shown in whatever unit fits best (KB, MB, or GB). This is the lifetime total, not a rate — the throughput graph below shows the current send/receive speed separately. Higher values mean the pod is handling more traffic. Data comes from the kubelet running on the node.';

export const TOOLTIP_POD_USAGE_SAME_AS_LIST =
  'Current CPU and memory usage for each pod. CPU is in millicores (1000m = 1 CPU core). Memory is in MiB. These are the same values shown in the Pods list.';

export const TOOLTIP_USAGE_VS_LIMITS =
  'Shows how close this pod is to hitting its resource limits. If CPU hits 100%, the pod gets throttled (slowed down). If memory hits 100%, the pod gets killed and restarted (OOMKilled). Ideally keep both under 80%.';
