import { useState, useCallback, useMemo } from 'react';
import { useParams, useNavigate, Link, useSearchParams } from 'react-router-dom';
import {
  Database,
  Clock,
  Server,
  RotateCcw,
  CheckCircle2,
  XCircle,
  Activity,
  Scale,
  HardDrive,
  Box,
  FileText,
  Terminal,
  LayoutDashboard,
  Layers,
  BarChart2,
  Settings,
  Globe,
  SlidersHorizontal,
  Hash,
  RefreshCw,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import { Progress } from '@/components/ui/progress';
import { toast } from '@/components/ui/sonner';
import {
  GenericResourceDetail,
  ContainersSection,
  LabelList,
  AnnotationList,
  MetricsDashboard,
  ScaleDialog,
  RolloutActionsDialog,
  SectionCard,
  DetailRow,
  LogViewer,
  DetailPodTable,
  type CustomTab,
  type ResourceContext,
  type ContainerInfo,
} from '@/components/resources';
import { PodTerminal } from '@/components/resources/PodTerminal';
import { useK8sResourceList, usePatchK8sResource, calculateAge, type KubernetesResource } from '@/hooks/useKubernetes';
import { useConnectionStatus } from '@/hooks/useConnectionStatus';
import { useMutationPolling } from '@/hooks/useMutationPolling';
import { useBackendConfigStore, getEffectiveBackendBaseUrl } from '@/stores/backendConfigStore';
import { useActiveClusterId } from '@/hooks/useActiveClusterId';

interface StatefulSetResource extends KubernetesResource {
  spec?: {
    replicas?: number;
    serviceName?: string;
    podManagementPolicy?: string;
    revisionHistoryLimit?: number;
    minReadySeconds?: number;
    updateStrategy?: { type?: string; rollingUpdate?: { partition?: number } };
    selector?: { matchLabels?: Record<string, string> };
    template?: {
      spec?: {
        containers?: Array<{
          name: string;
          image: string;
          ports?: Array<{ containerPort: number; protocol?: string }>;
          resources?: { requests?: { cpu?: string; memory?: string }; limits?: { cpu?: string; memory?: string } };
        }>;
      };
    };
    volumeClaimTemplates?: Array<{
      metadata?: { name: string };
      spec?: { storageClassName?: string; resources?: { requests?: { storage?: string } } };
    }>;
  };
  status?: {
    replicas?: number;
    readyReplicas?: number;
    currentReplicas?: number;
    updatedReplicas?: number;
    conditions?: Array<{ type: string; status: string; lastTransitionTime?: string; reason?: string; message?: string }>;
  };
}

// ---------------------------------------------------------------------------
// Overview tab
// ---------------------------------------------------------------------------

function OverviewTab({ resource: statefulSet }: ResourceContext<StatefulSetResource>) {
  const navigate = useNavigate();
  const { namespace } = useParams();
  const desired = statefulSet.spec?.replicas || 0;
  const ready = statefulSet.status?.readyReplicas || 0;
  const current = statefulSet.status?.currentReplicas || 0;
  const updated = statefulSet.status?.updatedReplicas || 0;
  const volumeClaimTemplates = statefulSet.spec?.volumeClaimTemplates || [];

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <SectionCard icon={Database} title="StatefulSet Information" tooltip={<p className="text-xs text-muted-foreground">Configuration and update strategy</p>}>
            <div className="grid grid-cols-2 gap-x-8 gap-y-3">
              <DetailRow
                label="Service Name"
                value={
                  <Button
                    variant="link"
                    className="h-auto p-0 font-mono text-sm font-semibold justify-start"
                    onClick={() => navigate(`/services/${namespace}/${statefulSet.spec?.serviceName}`)}
                  >
                    {statefulSet.spec?.serviceName || '-'}
                  </Button>
                }
              />
              <DetailRow label="Pod Management" value={<Badge variant="outline">{statefulSet.spec?.podManagementPolicy || 'OrderedReady'}</Badge>} />
              <DetailRow label="Update Strategy" value={<Badge variant="outline">{statefulSet.spec?.updateStrategy?.type || 'RollingUpdate'}</Badge>} />
              <DetailRow label="Partition" value={String(statefulSet.spec?.updateStrategy?.rollingUpdate?.partition ?? 0)} />
              <DetailRow label="Revision History Limit" value={String(statefulSet.spec?.revisionHistoryLimit ?? 10)} />
              <DetailRow label="Min Ready Seconds" value={`${statefulSet.spec?.minReadySeconds ?? 0}s`} />
            </div>
        </SectionCard>

        <SectionCard icon={Activity} title="Replica Status">
            <div className="space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-sm">Ready</span>
                <div className="flex items-center gap-2">
                  <Progress value={(ready / desired) * 100} className="w-32 h-2" />
                  <span className="font-mono text-sm w-12">{ready}/{desired}</span>
                </div>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm">Current</span>
                <div className="flex items-center gap-2">
                  <Progress value={(current / desired) * 100} className="w-32 h-2" />
                  <span className="font-mono text-sm w-12">{current}/{desired}</span>
                </div>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm">Updated</span>
                <div className="flex items-center gap-2">
                  <Progress value={(updated / desired) * 100} className="w-32 h-2" />
                  <span className="font-mono text-sm w-12">{updated}/{desired}</span>
                </div>
              </div>
            </div>
        </SectionCard>
      </div>

      {volumeClaimTemplates.length > 0 && (
        <SectionCard icon={HardDrive} title="Volume Claim Templates">
            <div className="space-y-3">
              {volumeClaimTemplates.map((vct, i) => (
                <div key={i} className="flex items-center justify-between p-3 rounded-lg bg-muted/50">
                  <div>
                    <p className="font-medium text-sm">{vct.metadata?.name || `volume-${i}`}</p>
                    <p className="text-xs text-muted-foreground">
                      Storage Class: {vct.spec?.storageClassName || 'default'}
                    </p>
                  </div>
                  <Badge variant="outline" className="font-mono">
                    {vct.spec?.resources?.requests?.storage || 'N/A'}
                  </Badge>
                </div>
              ))}
            </div>
        </SectionCard>
      )}

      {(statefulSet.status?.conditions?.length ?? 0) > 0 && (
        <SectionCard icon={Activity} title="Conditions">
            <div className="space-y-3">
              {statefulSet.status?.conditions?.map((c) => (
                <div key={c.type} className="flex items-center justify-between p-3 rounded-lg bg-muted/50">
                  <div className="flex items-center gap-3">
                    {c.status === 'True' ? <CheckCircle2 className="h-5 w-5 text-emerald-600" /> : <XCircle className="h-5 w-5 text-rose-600" />}
                    <div>
                      <p className="font-medium text-sm">{c.type}</p>
                      {c.reason && <p className="text-xs text-muted-foreground">{c.reason}</p>}
                    </div>
                  </div>
                  {c.lastTransitionTime && <span className="text-xs text-muted-foreground">{new Date(c.lastTransitionTime).toLocaleString()}</span>}
                </div>
              ))}
            </div>
        </SectionCard>
      )}

      {/* Metadata */}
      <div className="lg:col-span-2">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <LabelList labels={statefulSet.metadata?.labels || {}} />
          <LabelList labels={statefulSet.spec?.selector?.matchLabels || {}} title="Selector" />
        </div>
      </div>
      <div className="lg:col-span-2">
        <AnnotationList annotations={statefulSet.metadata?.annotations || {}} />
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export default function StatefulSetDetail() {
  const { namespace, name } = useParams();
  const clusterId = useActiveClusterId();
  const navigate = useNavigate();
  const { isConnected } = useConnectionStatus();
  const [, setSearchParams] = useSearchParams();

  const [showScaleDialog, setShowScaleDialog] = useState(false);
  const [showRolloutDialog, setShowRolloutDialog] = useState(false);
  const [selectedLogPod, setSelectedLogPod] = useState<string>('');
  const [selectedLogContainer, setSelectedLogContainer] = useState<string>('');
  const [selectedTerminalPod, setSelectedTerminalPod] = useState<string>('');
  const [selectedTerminalContainer, setSelectedTerminalContainer] = useState<string>('');
  const [partitionInput, setPartitionInput] = useState<string>('');

  const { refetchInterval: fastPollInterval, isFastPolling, triggerFastPolling } = useMutationPolling({
    fastInterval: 2000,
    fastDuration: 30000,
    normalInterval: 60000,
  });

  const backendBaseUrl = getEffectiveBackendBaseUrl(useBackendConfigStore((s) => s.backendBaseUrl));
  const isBackendConfigured = useBackendConfigStore((s) => s.isBackendConfigured());
  const patchStatefulSet = usePatchK8sResource('statefulsets');

  // Pods & PVC lists
  const { data: podsList } = useK8sResourceList<KubernetesResource & { metadata?: { name?: string; labels?: Record<string, string> }; status?: { phase?: string; podIP?: string }; spec?: { nodeName?: string } }>(
    'pods',
    namespace ?? undefined,
    { enabled: !!namespace, limit: 5000, refetchInterval: fastPollInterval, staleTime: isFastPolling ? 1000 : 30000 }
  );
  const { data: pvcList } = useK8sResourceList<KubernetesResource & { metadata?: { name?: string; creationTimestamp?: string }; status?: { phase?: string; capacity?: { storage?: string } }; spec?: { storageClassName?: string; resources?: { requests?: { storage?: string } } } }>(
    'persistentvolumeclaims',
    namespace ?? undefined,
    { enabled: !!namespace && !!name, limit: 5000 }
  );

  const handleScale = useCallback(async (replicas: number) => {
    if (!isConnected || !name || !namespace) { toast.error('Connect cluster to scale StatefulSet'); return; }
    try {
      await patchStatefulSet.mutateAsync({ name, namespace, patch: { spec: { replicas } } });
      toast.success(`Scaled ${name} to ${replicas} replicas`);
      triggerFastPolling();
      setSearchParams({ tab: 'pods-ordinals' });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to scale');
      throw err;
    }
  }, [isConnected, name, namespace, patchStatefulSet, triggerFastPolling, setSearchParams]);

  const handleRestart = useCallback(async () => {
    if (!isConnected || !name || !namespace) { toast.error('Connect cluster to restart StatefulSet'); return; }
    try {
      await patchStatefulSet.mutateAsync({
        name,
        namespace,
        patch: { spec: { template: { metadata: { annotations: { 'kubectl.kubernetes.io/restartedAt': new Date().toISOString() } } } } },
      });
      toast.success(`Rollout restart initiated for ${name}`);
      triggerFastPolling();
      setSearchParams({ tab: 'pods-ordinals' });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to restart');
      throw err;
    }
  }, [isConnected, name, namespace, patchStatefulSet, triggerFastPolling, setSearchParams]);

  const handleRollback = useCallback(async (_revision: number) => {
    toast.info('Rollback for StatefulSet is revision-specific; use detail when supported.');
  }, []);

  const customTabs: CustomTab[] = [
    {
      id: 'overview',
      label: 'Overview',
      icon: LayoutDashboard,
      render: (ctx) => <OverviewTab {...ctx} />,
    },
    {
      id: 'pods-ordinals',
      label: 'Pods & Ordinals',
      icon: Hash,
      render: (ctx) => {
        const sts = ctx.resource;
        const stsMatchLabels = sts.spec?.selector?.matchLabels ?? {};
        const stsPodsRaw = (podsList?.items ?? []).filter((pod) => {
          const labels = pod.metadata?.labels ?? {};
          return Object.entries(stsMatchLabels).every(([k, v]) => labels[k] === v);
        });
        const stsName = sts.metadata?.name ?? '';
        const stsPods = [...stsPodsRaw].sort((a, b) => {
          const ordA = parseInt(a.metadata?.name?.replace(new RegExp(`^${stsName}-`), '') ?? '-1', 10);
          const ordB = parseInt(b.metadata?.name?.replace(new RegExp(`^${stsName}-`), '') ?? '-1', 10);
          return ordA - ordB;
        });
        return (
          <SectionCard icon={Box} title="Pods & Ordinals" tooltip={<p className="text-xs text-muted-foreground">Ordered pods managed by this StatefulSet</p>}>
            <DetailPodTable
              pods={stsPods}
              namespace={namespace ?? ''}
              extraColumns={[
                {
                  header: 'Ordinal',
                  render: (pod) => {
                    const podName = pod.metadata?.name ?? '';
                    const parts = podName.split('-');
                    const ordinal = parseInt(parts[parts.length - 1], 10);
                    return <span className="font-mono text-xs">{isNaN(ordinal) ? '–' : ordinal}</span>;
                  },
                },
              ]}
            />
          </SectionCard>
        );
      },
    },
    {
      id: 'pvc',
      label: 'PersistentVolumeClaims',
      icon: HardDrive,
      render: (ctx) => {
        const sts = ctx.resource;
        const stsName = sts.metadata?.name ?? '';
        const stsMatchLabels = sts.spec?.selector?.matchLabels ?? {};
        const stsPodsRaw = (podsList?.items ?? []).filter((pod) => {
          const labels = pod.metadata?.labels ?? {};
          return Object.entries(stsMatchLabels).every(([k, v]) => labels[k] === v);
        });
        const volumeClaimTemplates = sts.spec?.volumeClaimTemplates || [];

        const templatePattern = new RegExp(`^[a-z0-9-]+-${stsName}-\\d+$`);
        const podPvcNames = new Set<string>();
        for (const pod of stsPodsRaw) {
          const volumes = (pod as { spec?: { volumes?: Array<{ persistentVolumeClaim?: { claimName?: string } }> } }).spec?.volumes ?? [];
          for (const vol of volumes) {
            if (vol.persistentVolumeClaim?.claimName) podPvcNames.add(vol.persistentVolumeClaim.claimName);
          }
        }
        const stsPvcs = (pvcList?.items ?? []).filter((pvc) => {
          const pvcName = pvc.metadata?.name ?? '';
          return templatePattern.test(pvcName) || podPvcNames.has(pvcName);
        });
        const stsPvcsWithOrdinal = stsPvcs.map((pvc) => {
          const pvcName = pvc.metadata?.name ?? '';
          const match = pvcName.match(new RegExp(`-${stsName}-(\\d+)$`));
          const ordinal = match ? parseInt(match[1], 10) : null;
          return { pvc, ordinal };
        }).sort((a, b) => (a.ordinal ?? -1) - (b.ordinal ?? -1));
        const pvcTotalStorage = (() => {
          let totalGi = 0;
          for (const { pvc } of stsPvcsWithOrdinal) {
            const raw = (pvc.status as { capacity?: { storage?: string } })?.capacity?.storage ?? '';
            const m = raw.match(/^(\d+(?:\.\d+)?)\s*Gi?$/i);
            if (m) totalGi += parseFloat(m[1]);
            else {
              const mMi = raw.match(/^(\d+(?:\.\d+)?)\s*Mi?$/i);
              if (mMi) totalGi += parseFloat(mMi[1]) / 1024;
            }
          }
          return totalGi > 0 ? `${totalGi.toFixed(2)} Gi` : null;
        })();

        return (
          <SectionCard icon={HardDrive} title="PersistentVolumeClaims" tooltip={<p className="text-xs text-muted-foreground">PVCs used by this StatefulSet, per pod ordinal</p>}>
            {stsPvcs.length === 0 && volumeClaimTemplates.length === 0 ? (
              <p className="text-sm text-muted-foreground">No PVCs associated with this StatefulSet.</p>
            ) : stsPvcs.length === 0 ? (
              <p className="text-sm text-muted-foreground">No PVCs found for this StatefulSet yet.</p>
            ) : (
              <div className="space-y-3">
                <div className="rounded-lg border overflow-x-auto">
                  <table className="w-full text-sm min-w-[600px]">
                    <thead className="bg-muted/50">
                      <tr>
                        <th className="text-left p-3 font-medium">PVC Name</th>
                        <th className="text-left p-3 font-medium">Bound Pod (ordinal)</th>
                        <th className="text-left p-3 font-medium">Status</th>
                        <th className="text-left p-3 font-medium">Capacity</th>
                        <th className="text-left p-3 font-medium">StorageClass</th>
                        <th className="text-left p-3 font-medium">Age</th>
                      </tr>
                    </thead>
                    <tbody>
                      {stsPvcsWithOrdinal.map(({ pvc, ordinal }) => {
                        const pvcName = pvc.metadata?.name ?? '';
                        const phase = (pvc.status as { phase?: string })?.phase ?? 'Unknown';
                        const storageClass = (pvc.spec as { storageClassName?: string })?.storageClassName ?? 'default';
                        const capacity = (pvc.status as { capacity?: { storage?: string } })?.capacity?.storage ?? '—';
                        const pvcAge = pvc.metadata?.creationTimestamp ? calculateAge(pvc.metadata.creationTimestamp) : '—';
                        const podLabel = ordinal !== null ? `${stsName}-${ordinal}` : '—';
                        const statusVariant = phase === 'Bound' ? 'default' : phase === 'Pending' ? 'secondary' : 'destructive';
                        const statusClassName = phase === 'Bound' ? 'bg-emerald-600/90 text-white border-0' : phase === 'Pending' ? 'bg-amber-500/90 text-white border-0' : phase === 'Lost' ? 'bg-destructive/90 text-destructive-foreground border-0' : '';
                        return (
                          <tr key={pvcName} className="border-t">
                            <td className="p-3">
                              <Link to={`/persistentvolumeclaims/${namespace}/${pvcName}`} className="text-primary hover:underline font-mono text-xs">
                                {pvcName}
                              </Link>
                            </td>
                            <td className="p-3 font-mono text-xs">
                              {ordinal !== null ? <Link to={`/pods/${namespace}/${podLabel}`} className="text-primary hover:underline">{podLabel}</Link> : '—'}
                            </td>
                            <td className="p-3">
                              <Badge variant={statusVariant} className={statusClassName || undefined}>{phase}</Badge>
                            </td>
                            <td className="p-3 font-mono text-xs">{capacity}</td>
                            <td className="p-3 font-mono text-xs">{storageClass}</td>
                            <td className="p-3 text-muted-foreground">{pvcAge}</td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
                {pvcTotalStorage && (
                  <p className="text-sm font-medium text-muted-foreground">
                    Total storage: <span className="font-mono text-foreground">{pvcTotalStorage}</span>
                  </p>
                )}
              </div>
            )}
          </SectionCard>
        );
      },
    },
    {
      id: 'headless-service',
      label: 'Headless Service',
      icon: Globe,
      render: (ctx) => {
        const sts = ctx.resource;
        const desired = sts.spec?.replicas || 0;
        const stsName = sts.metadata?.name ?? '';
        return (
          <SectionCard icon={Globe} title="Headless Service" tooltip={<p className="text-xs text-muted-foreground">Service and DNS for StatefulSet pods</p>}>
            {!sts.spec?.serviceName ? (
              <p className="text-sm text-muted-foreground">No service name configured.</p>
            ) : (
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <p className="text-muted-foreground mb-1">Service</p>
                    <Link to={`/services/${namespace}/${sts.spec.serviceName}`} className="font-mono text-primary hover:underline">
                      {sts.spec.serviceName}
                    </Link>
                  </div>
                  <div>
                    <p className="text-muted-foreground mb-1">Cluster IP</p>
                    <p className="font-mono">None (headless)</p>
                  </div>
                </div>
                <div>
                  <p className="text-muted-foreground mb-2">DNS names (pod-0, pod-1, ...)</p>
                  <div className="rounded-lg bg-muted/50 p-3 font-mono text-xs space-y-1">
                    {Array.from({ length: desired }, (_, i) => (
                      <div key={i}>{stsName}-{i}.{sts.spec!.serviceName}.{namespace}.svc.cluster.local</div>
                    ))}
                  </div>
                </div>
              </div>
            )}
          </SectionCard>
        );
      },
    },
    {
      id: 'update-strategy',
      label: 'Update Strategy',
      icon: SlidersHorizontal,
      render: (ctx) => {
        const sts = ctx.resource;
        const updateStrategyType = sts.spec?.updateStrategy?.type ?? 'RollingUpdate';
        const partition = sts.spec?.updateStrategy?.rollingUpdate?.partition ?? 0;
        const desired = sts.spec?.replicas || 0;
        const updated = sts.status?.updatedReplicas || 0;

        return (
          <SectionCard icon={SlidersHorizontal} title="Update Strategy" tooltip={<p className="text-xs text-muted-foreground">Strategy and partition control</p>}>
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <p className="text-muted-foreground mb-1">Type</p>
                  <Badge variant="outline">{updateStrategyType}</Badge>
                </div>
                <div>
                  <p className="text-muted-foreground mb-1">Partition</p>
                  <p className="font-mono">Pods with ordinal &gt;= partition receive updates.</p>
                </div>
              </div>
              {updateStrategyType === 'RollingUpdate' && (
                <>
                  <div className="flex items-center gap-4 flex-wrap">
                    <Label className="w-24">Partition</Label>
                    <Input
                      type="number"
                      min={0}
                      max={desired}
                      value={partitionInput !== '' ? partitionInput : partition}
                      onChange={(e) => setPartitionInput(e.target.value)}
                      className="w-24 font-mono"
                    />
                    <Button
                      size="sm"
                      onClick={async () => {
                        const val = partitionInput !== '' ? parseInt(partitionInput, 10) : partition;
                        if (Number.isNaN(val) || val < 0) return;
                        try {
                          await patchStatefulSet.mutateAsync({
                            name: name!,
                            namespace: namespace!,
                            patch: { spec: { updateStrategy: { type: 'RollingUpdate', rollingUpdate: { partition: val } } } },
                          });
                          toast.success(`Partition set to ${val}`);
                          setPartitionInput('');
                          ctx.refetch();
                        } catch (err) {
                          toast.error(err instanceof Error ? err.message : 'Failed to update partition');
                        }
                      }}
                    >
                      Apply
                    </Button>
                  </div>
                  <div>
                    <p className="text-muted-foreground text-sm mb-1">Update progress</p>
                    <p className="text-sm">Updated replicas: <span className="font-mono">{updated}</span> of <span className="font-mono">{desired}</span></p>
                  </div>
                </>
              )}
            </div>
          </SectionCard>
        );
      },
    },
    {
      id: 'containers',
      label: 'Containers',
      icon: Layers,
      render: (ctx) => {
        const containers: ContainerInfo[] = (ctx.resource.spec?.template?.spec?.containers || []).map(c => ({
          name: c.name, image: c.image, ready: true, restartCount: 0, state: 'running',
          ports: c.ports?.map(p => ({ containerPort: p.containerPort, protocol: p.protocol || 'TCP' })) || [],
          resources: c.resources || {},
        }));
        return <ContainersSection containers={containers} />;
      },
    },
    {
      id: 'logs',
      label: 'Logs',
      icon: FileText,
      render: (ctx) => {
        const sts = ctx.resource;
        const stsMatchLabels = sts.spec?.selector?.matchLabels ?? {};
        const stsPods = (podsList?.items ?? []).filter((pod) => {
          const labels = pod.metadata?.labels ?? {};
          return Object.entries(stsMatchLabels).every(([k, v]) => labels[k] === v);
        });
        const containers: ContainerInfo[] = (sts.spec?.template?.spec?.containers || []).map(c => ({
          name: c.name, image: c.image, ready: true, restartCount: 0, state: 'running',
          ports: c.ports?.map(p => ({ containerPort: p.containerPort, protocol: p.protocol || 'TCP' })) || [],
          resources: c.resources || {},
        }));
        const firstPodName = stsPods[0]?.metadata?.name ?? '';
        const logPod = selectedLogPod || firstPodName;
        const logPodContainers = (stsPods.find((p) => p.metadata?.name === logPod) as { spec?: { containers?: Array<{ name: string }> } } | undefined)?.spec?.containers?.map((c) => c.name) ?? containers.map((c) => c.name);

        return (
          <SectionCard icon={FileText} title="Logs" tooltip={<p className="text-xs text-muted-foreground">Stream logs from StatefulSet pods</p>}>
            {stsPods.length === 0 ? (
              <p className="text-sm text-muted-foreground">No pods available to view logs.</p>
            ) : (
              <div className="space-y-4">
                <div className="flex flex-wrap gap-4 items-end">
                  <div className="space-y-2">
                    <Label>Pod</Label>
                    <Select value={logPod} onValueChange={setSelectedLogPod}>
                      <SelectTrigger className="w-[280px]"><SelectValue placeholder="Select pod" /></SelectTrigger>
                      <SelectContent>
                        {stsPods.map((p) => (<SelectItem key={p.metadata?.name} value={p.metadata?.name ?? ''}>{p.metadata?.name}</SelectItem>))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>Container</Label>
                    <Select value={selectedLogContainer || logPodContainers[0]} onValueChange={setSelectedLogContainer}>
                      <SelectTrigger className="w-[180px]"><SelectValue placeholder="Select container" /></SelectTrigger>
                      <SelectContent>
                        {logPodContainers.map((c) => (<SelectItem key={c} value={c}>{c}</SelectItem>))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                <LogViewer podName={logPod} namespace={namespace ?? undefined} containerName={selectedLogContainer || logPodContainers[0]} containers={logPodContainers} onContainerChange={setSelectedLogContainer} />
              </div>
            )}
          </SectionCard>
        );
      },
    },
    {
      id: 'terminal',
      label: 'Terminal',
      icon: Terminal,
      render: (ctx) => {
        const sts = ctx.resource;
        const stsMatchLabels = sts.spec?.selector?.matchLabels ?? {};
        const stsPods = (podsList?.items ?? []).filter((pod) => {
          const labels = pod.metadata?.labels ?? {};
          return Object.entries(stsMatchLabels).every(([k, v]) => labels[k] === v);
        });
        const containers: ContainerInfo[] = (sts.spec?.template?.spec?.containers || []).map(c => ({
          name: c.name, image: c.image, ready: true, restartCount: 0, state: 'running',
          ports: c.ports?.map(p => ({ containerPort: p.containerPort, protocol: p.protocol || 'TCP' })) || [],
          resources: c.resources || {},
        }));
        const firstPodName = stsPods[0]?.metadata?.name ?? '';
        const terminalPod = selectedTerminalPod || firstPodName;
        const terminalPodContainers = (stsPods.find((p) => p.metadata?.name === terminalPod) as { spec?: { containers?: Array<{ name: string }> } } | undefined)?.spec?.containers?.map((c) => c.name) ?? containers.map((c) => c.name);

        return (
          <SectionCard icon={Terminal} title="Terminal" tooltip={<p className="text-xs text-muted-foreground">Exec into StatefulSet pods</p>}>
            {stsPods.length === 0 ? (
              <p className="text-sm text-muted-foreground">No pods available for terminal.</p>
            ) : (
              <div className="space-y-4">
                <div className="flex flex-wrap gap-4 items-end">
                  <div className="space-y-2">
                    <Label>Pod</Label>
                    <Select value={terminalPod} onValueChange={setSelectedTerminalPod}>
                      <SelectTrigger className="w-[280px]"><SelectValue placeholder="Select pod" /></SelectTrigger>
                      <SelectContent>
                        {stsPods.map((p) => (<SelectItem key={p.metadata?.name} value={p.metadata?.name ?? ''}>{p.metadata?.name}</SelectItem>))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>Container</Label>
                    <Select value={selectedTerminalContainer || terminalPodContainers[0]} onValueChange={setSelectedTerminalContainer}>
                      <SelectTrigger className="w-[180px]"><SelectValue placeholder="Select container" /></SelectTrigger>
                      <SelectContent>
                        {terminalPodContainers.map((c) => (<SelectItem key={c} value={c}>{c}</SelectItem>))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                <PodTerminal key={`${terminalPod}-${selectedTerminalContainer || terminalPodContainers[0]}`} podName={terminalPod} namespace={namespace ?? undefined} containerName={selectedTerminalContainer || terminalPodContainers[0]} containers={terminalPodContainers} onContainerChange={setSelectedTerminalContainer} />
              </div>
            )}
          </SectionCard>
        );
      },
    },
    {
      id: 'metrics',
      label: 'Metrics',
      icon: BarChart2,
      render: () => <MetricsDashboard resourceType="statefulset" resourceName={name} namespace={namespace} clusterId={clusterId} />,
    },
  ];

  return (
    <>
      <GenericResourceDetail<StatefulSetResource>
        resourceType="statefulsets"
        kind="StatefulSet"
        pluralLabel="StatefulSets"
        listPath="/statefulsets"
        resourceIcon={Database}
        loadingCardCount={6}
        detailOptions={{ refetchInterval: fastPollInterval, staleTime: isFastPolling ? 1000 : 5000 }}
        deriveStatus={(sts) => sts.status?.readyReplicas === sts.spec?.replicas ? 'Running' : sts.status?.readyReplicas ? 'Pending' : 'Failed'}
        customTabs={customTabs}
        buildStatusCards={(ctx) => {
          const sts = ctx.resource;
          const desired = sts.spec?.replicas || 0;
          const ready = sts.status?.readyReplicas || 0;
          const updateStrategyType = sts.spec?.updateStrategy?.type ?? 'RollingUpdate';
          const partition = sts.spec?.updateStrategy?.rollingUpdate?.partition ?? 0;
          const volumeClaimTemplates = sts.spec?.volumeClaimTemplates || [];

          // PVC count calculation
          const stsMatchLabels = sts.spec?.selector?.matchLabels ?? {};
          const stsPodsRaw = (podsList?.items ?? []).filter((pod) => {
            const labels = pod.metadata?.labels ?? {};
            return Object.entries(stsMatchLabels).every(([k, v]) => labels[k] === v);
          });
          const stsName = sts.metadata?.name ?? '';
          const templatePattern = new RegExp(`^[a-z0-9-]+-${stsName}-\\d+$`);
          const podPvcNames = new Set<string>();
          for (const pod of stsPodsRaw) {
            const volumes = (pod as { spec?: { volumes?: Array<{ persistentVolumeClaim?: { claimName?: string } }> } }).spec?.volumes ?? [];
            for (const vol of volumes) {
              if (vol.persistentVolumeClaim?.claimName) podPvcNames.add(vol.persistentVolumeClaim.claimName);
            }
          }
          const stsPvcs = (pvcList?.items ?? []).filter((pvc) => {
            const pvcName = pvc.metadata?.name ?? '';
            return templatePattern.test(pvcName) || podPvcNames.has(pvcName);
          });
          const pvcCount = stsPvcs.length || (volumeClaimTemplates.length * desired);

          return [
            { label: 'Ready', value: `${ready}/${desired}`, icon: Server, iconColor: ready === desired ? 'success' as const : 'warning' as const },
            { label: 'Replicas', value: desired, icon: Activity, iconColor: 'info' as const },
            { label: 'Update Strategy', value: updateStrategyType, icon: SlidersHorizontal, iconColor: 'primary' as const },
            { label: 'Partition', value: partition, icon: Hash, iconColor: 'primary' as const },
            { label: 'Service', value: sts.spec?.serviceName || '—', icon: Globe, iconColor: 'primary' as const },
            { label: 'PVCs', value: pvcCount, icon: HardDrive, iconColor: 'primary' as const },
          ];
        }}
        headerMetadata={(ctx) => (
          <span className="flex items-center gap-1.5 ml-2 text-sm text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            Created {ctx.age}
            <span className="mx-2">&bull;</span>
            <Server className="h-3.5 w-3.5" />
            {ctx.resource.spec?.serviceName}
            {ctx.isConnected && <Badge variant="outline" className="ml-2 text-xs">Live</Badge>}
            {isFastPolling && (
              <Badge className="ml-2 text-xs bg-blue-500/20 text-blue-600 dark:text-blue-400 border-blue-500/30 animate-pulse gap-1">
                <RefreshCw className="h-3 w-3 animate-spin" />
                Syncing
              </Badge>
            )}
          </span>
        )}
        extraHeaderActions={() => [
          { label: 'Scale', icon: Scale, variant: 'outline', onClick: () => setShowScaleDialog(true), className: 'press-effect' },
          { label: 'Restart', icon: RotateCcw, variant: 'outline', onClick: () => setShowRolloutDialog(true), className: 'press-effect' },
        ]}
        extraActionItems={() => [
          { icon: Scale, label: 'Scale StatefulSet', description: 'Adjust the number of replicas', onClick: () => setShowScaleDialog(true), className: 'press-effect' },
          { icon: RotateCcw, label: 'Rollout Restart', description: 'Trigger a rolling restart', onClick: () => setShowRolloutDialog(true), className: 'press-effect' },
        ]}
        extraDialogs={(ctx) => (
          <>
            <ScaleDialog
              open={showScaleDialog}
              onOpenChange={setShowScaleDialog}
              resourceType="StatefulSet"
              resourceName={ctx.name}
              namespace={ctx.namespace}
              currentReplicas={ctx.resource.spec?.replicas || 0}
              onScale={handleScale}
            />
            <RolloutActionsDialog
              open={showRolloutDialog}
              onOpenChange={setShowRolloutDialog}
              resourceType="StatefulSet"
              resourceName={ctx.name}
              namespace={ctx.namespace}
              revisions={[]}
              onRestart={handleRestart}
              onRollback={handleRollback}
            />
          </>
        )}
      />
    </>
  );
}
