import { useState, useCallback, useMemo, useEffect } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import {
  Layers,
  Clock,
  Server,
  CheckCircle2,
  Activity,
  Scale,
  Box,
  FileText,
  Terminal,
  LayoutDashboard,
  BarChart2,
  RefreshCw,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
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
  SectionCard,
  DetailRow,
  LogViewer,
  DetailPodTable,
  type CustomTab,
  type ResourceContext,
  type ContainerInfo,
} from '@/components/resources';
import { PodTerminal } from '@/components/resources/PodTerminal';
import { useK8sResourceList, usePatchK8sResource, type KubernetesResource } from '@/hooks/useKubernetes';
import { useMutationPolling } from '@/hooks/useMutationPolling';
import { useBackendConfigStore, getEffectiveBackendBaseUrl } from '@/stores/backendConfigStore';
import { useActiveClusterId } from '@/hooks/useActiveClusterId';
import { useConnectionStatus } from '@/hooks/useConnectionStatus';
import { useQuery } from '@tanstack/react-query';
import { getReplicaSetMetrics } from '@/services/backendApiClient';

interface ReplicaSetResource extends KubernetesResource {
  spec?: {
    replicas?: number;
    selector?: { matchLabels?: Record<string, string> };
    template?: {
      spec?: {
        containers?: Array<{
          name: string;
          image: string;
          ports?: Array<{ containerPort: number; protocol: string }>;
          resources?: { requests?: { cpu?: string; memory?: string }; limits?: { cpu?: string; memory?: string } };
        }>;
      };
    };
  };
  status?: {
    replicas?: number;
    readyReplicas?: number;
    availableReplicas?: number;
    fullyLabeledReplicas?: number;
  };
}

// ---------------------------------------------------------------------------
// Custom tab components
// ---------------------------------------------------------------------------

function OverviewTab({ resource: replicaSet }: ResourceContext<ReplicaSetResource>) {
  const navigate = useNavigate();
  const { namespace } = useParams();
  const desired = replicaSet.spec?.replicas || 0;
  const ready = replicaSet.status?.readyReplicas || 0;
  const available = replicaSet.status?.availableReplicas || 0;
  const fullyLabeled = replicaSet.status?.fullyLabeledReplicas || 0;
  const ownerRef = replicaSet.metadata?.ownerReferences?.[0];

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <SectionCard icon={Layers} title="ReplicaSet Information" tooltip={<p className="text-xs text-muted-foreground">Configuration and ownership details</p>}>
          <div className="grid grid-cols-2 gap-x-8 gap-y-3">
            <DetailRow label="Desired Replicas" value={String(desired)} />
            <DetailRow label="Current Replicas" value={String(replicaSet.status?.replicas || 0)} />
            {ownerRef && (
              <DetailRow
                label="Owner"
                className="col-span-2"
                value={
                  <Button
                    variant="link"
                    className="h-auto p-0 font-semibold text-sm justify-start"
                    onClick={() => navigate(`/deployments/${namespace}/${ownerRef.name}`)}
                  >
                    {ownerRef.kind}: {ownerRef.name}
                  </Button>
                }
              />
            )}
          </div>
        </SectionCard>

        <SectionCard icon={Activity} title="Replica Status">
          <div className="space-y-3">
            <div className="flex justify-between items-center">
              <span className="text-sm">Ready</span>
              <div className="flex items-center gap-2">
                <Progress value={desired > 0 ? (ready / desired) * 100 : 0} className="w-32 h-2" />
                <span className="font-mono text-sm w-12">{ready}/{desired}</span>
              </div>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-sm">Available</span>
              <div className="flex items-center gap-2">
                <Progress value={desired > 0 ? (available / desired) * 100 : 0} className="w-32 h-2" />
                <span className="font-mono text-sm w-12">{available}/{desired}</span>
              </div>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-sm">Fully Labeled</span>
              <div className="flex items-center gap-2">
                <Progress value={desired > 0 ? (fullyLabeled / desired) * 100 : 0} className="w-32 h-2" />
                <span className="font-mono text-sm w-12">{fullyLabeled}/{desired}</span>
              </div>
            </div>
          </div>
        </SectionCard>
      </div>

      {/* Metadata */}
      <div className="lg:col-span-2">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <LabelList labels={replicaSet.metadata?.labels ?? {}} />
          <LabelList labels={replicaSet.spec?.selector?.matchLabels ?? {}} title="Selector" />
        </div>
      </div>
      <div className="lg:col-span-2">
        <AnnotationList annotations={replicaSet.metadata?.annotations ?? {}} />
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export default function ReplicaSetDetail() {
  const { namespace, name } = useParams();
  const clusterId = useActiveClusterId();
  const [, setSearchParams] = useSearchParams();
  const { isConnected } = useConnectionStatus();

  const [showScaleDialog, setShowScaleDialog] = useState(false);
  const [selectedLogPod, setSelectedLogPod] = useState<string>('');
  const [selectedLogContainer, setSelectedLogContainer] = useState<string>('');
  const [selectedTerminalPod, setSelectedTerminalPod] = useState<string>('');
  const [selectedTerminalContainer, setSelectedTerminalContainer] = useState<string>('');

  const { refetchInterval: fastPollInterval, isFastPolling, triggerFastPolling } = useMutationPolling({
    fastInterval: 2000,
    fastDuration: 30000,
    normalInterval: 60000,
  });

  const backendBaseUrl = getEffectiveBackendBaseUrl(useBackendConfigStore((s) => s.backendBaseUrl));
  const isBackendConfigured = useBackendConfigStore((s) => s.isBackendConfigured)();
  const patchReplicaSet = usePatchK8sResource('replicasets');

  // Pods list for tabs
  const { data: podsList } = useK8sResourceList<KubernetesResource & { metadata?: { name?: string; labels?: Record<string, string> }; status?: { phase?: string }; spec?: { nodeName?: string } }>(
    'pods',
    namespace ?? undefined,
    { enabled: !!namespace, refetchInterval: fastPollInterval, staleTime: isFastPolling ? 1000 : 30000 }
  );

  const handleScale = useCallback(async (replicas: number) => {
    if (!isConnected || !name || !namespace) {
      toast.error('Connect cluster to scale ReplicaSet');
      return;
    }
    if (!isBackendConfigured()) {
      toast.error('Connect to Kubilitics backend in Settings to scale.');
      return;
    }
    if (!clusterId) {
      toast.error('Select a cluster from the cluster list to perform this action.');
      return;
    }
    try {
      await patchReplicaSet.mutateAsync({ name, namespace, patch: { spec: { replicas } } });
      toast.success(`Scaled ${name} to ${replicas} replicas`);
      triggerFastPolling();
      setSearchParams({ tab: 'pods' });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to scale');
      throw err;
    }
  }, [isConnected, name, namespace, clusterId, patchReplicaSet, triggerFastPolling, setSearchParams, isBackendConfigured]);

  const customTabs: CustomTab[] = [
    {
      id: 'overview',
      label: 'Overview',
      icon: LayoutDashboard,
      render: (ctx) => <OverviewTab {...ctx} />,
    },
    {
      id: 'pods',
      label: 'Pods',
      icon: Box,
      render: (ctx) => {
        const rs = ctx.resource;
        const rsMatchLabels = rs.spec?.selector?.matchLabels ?? {};
        const rsPods = (podsList?.items ?? []).filter((pod) => {
          const labels = pod.metadata?.labels ?? {};
          return Object.entries(rsMatchLabels).every(([k, v]) => labels[k] === v);
        });
        return (
          <SectionCard icon={Box} title="Pods" tooltip={<p className="text-xs text-muted-foreground">Pods managed by this ReplicaSet</p>}>
            <DetailPodTable pods={rsPods} namespace={namespace ?? ''} />
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
          name: c.name,
          image: c.image,
          ready: true,
          restartCount: 0,
          state: 'running',
          ports: c.ports || [],
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
        const rs = ctx.resource;
        const rsMatchLabels = rs.spec?.selector?.matchLabels ?? {};
        const rsPods = (podsList?.items ?? []).filter((pod) => {
          const labels = pod.metadata?.labels ?? {};
          return Object.entries(rsMatchLabels).every(([k, v]) => labels[k] === v);
        });
        const containers: ContainerInfo[] = (rs.spec?.template?.spec?.containers || []).map(c => ({
          name: c.name, image: c.image, ready: true, restartCount: 0, state: 'running', ports: c.ports || [], resources: c.resources || {},
        }));
        const firstPodName = rsPods[0]?.metadata?.name ?? '';
        const logPod = selectedLogPod || firstPodName;
        const logPodContainers = (rsPods.find((p) => p.metadata?.name === logPod) as { spec?: { containers?: Array<{ name: string }> } } | undefined)?.spec?.containers?.map((c) => c.name) ?? containers.map((c) => c.name);

        return (
          <SectionCard icon={FileText} title="Logs" tooltip={<p className="text-xs text-muted-foreground">Stream logs from ReplicaSet pods</p>}>
            {rsPods.length === 0 ? (
              <p className="text-sm text-muted-foreground">No pods available to view logs.</p>
            ) : (
              <div className="space-y-4">
                <div className="flex flex-wrap gap-4 items-end">
                  <div className="space-y-2">
                    <Label>Pod</Label>
                    <Select value={logPod} onValueChange={setSelectedLogPod}>
                      <SelectTrigger className="w-[280px]">
                        <SelectValue placeholder="Select pod" />
                      </SelectTrigger>
                      <SelectContent>
                        {rsPods.map((p) => (
                          <SelectItem key={p.metadata?.name} value={p.metadata?.name ?? ''}>
                            {p.metadata?.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>Container</Label>
                    <Select value={selectedLogContainer || logPodContainers[0]} onValueChange={setSelectedLogContainer}>
                      <SelectTrigger className="w-[180px]">
                        <SelectValue placeholder="Select container" />
                      </SelectTrigger>
                      <SelectContent>
                        {logPodContainers.map((c) => (
                          <SelectItem key={c} value={c}>{c}</SelectItem>
                        ))}
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
        const rs = ctx.resource;
        const rsMatchLabels = rs.spec?.selector?.matchLabels ?? {};
        const rsPods = (podsList?.items ?? []).filter((pod) => {
          const labels = pod.metadata?.labels ?? {};
          return Object.entries(rsMatchLabels).every(([k, v]) => labels[k] === v);
        });
        const containers: ContainerInfo[] = (rs.spec?.template?.spec?.containers || []).map(c => ({
          name: c.name, image: c.image, ready: true, restartCount: 0, state: 'running', ports: c.ports || [], resources: c.resources || {},
        }));
        const firstPodName = rsPods[0]?.metadata?.name ?? '';
        const terminalPod = selectedTerminalPod || firstPodName;
        const terminalPodContainers = (rsPods.find((p) => p.metadata?.name === terminalPod) as { spec?: { containers?: Array<{ name: string }> } } | undefined)?.spec?.containers?.map((c) => c.name) ?? containers.map((c) => c.name);

        return (
          <SectionCard icon={Terminal} title="Terminal" tooltip={<p className="text-xs text-muted-foreground">Exec into ReplicaSet pods</p>}>
            {rsPods.length === 0 ? (
              <p className="text-sm text-muted-foreground">No pods available to open a terminal.</p>
            ) : (
              <div className="space-y-4">
                <div className="flex flex-wrap gap-4 items-end">
                  <div className="space-y-2">
                    <Label>Pod</Label>
                    <Select value={terminalPod} onValueChange={setSelectedTerminalPod}>
                      <SelectTrigger className="w-[280px]">
                        <SelectValue placeholder="Select pod" />
                      </SelectTrigger>
                      <SelectContent>
                        {rsPods.map((p) => (
                          <SelectItem key={p.metadata?.name} value={p.metadata?.name ?? ''}>
                            {p.metadata?.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>Container</Label>
                    <Select value={selectedTerminalContainer || terminalPodContainers[0]} onValueChange={setSelectedTerminalContainer}>
                      <SelectTrigger className="w-[180px]">
                        <SelectValue placeholder="Select container" />
                      </SelectTrigger>
                      <SelectContent>
                        {terminalPodContainers.map((c) => (
                          <SelectItem key={c} value={c}>{c}</SelectItem>
                        ))}
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
      render: () => <MetricsDashboard resourceType="replicaset" resourceName={name} namespace={namespace} clusterId={clusterId} />,
    },
  ];

  return (
    <>
      <GenericResourceDetail<ReplicaSetResource>
        resourceType="replicasets"
        kind="ReplicaSet"
        pluralLabel="ReplicaSets"
        listPath="/replicasets"
        resourceIcon={Layers}
        loadingCardCount={4}
        detailOptions={{ refetchInterval: fastPollInterval, staleTime: isFastPolling ? 1000 : 5000 }}
        deriveStatus={(rs) => rs.status?.readyReplicas === rs.spec?.replicas ? 'Running' : rs.status?.readyReplicas ? 'Pending' : 'Failed'}
        customTabs={customTabs}
        buildStatusCards={(ctx) => {
          const rs = ctx.resource;
          const desired = rs.spec?.replicas || 0;
          const ready = rs.status?.readyReplicas || 0;
          const available = rs.status?.availableReplicas || 0;
          const fullyLabeled = rs.status?.fullyLabeledReplicas || 0;
          return [
            { label: 'Ready', value: `${ready}/${desired}`, icon: Server, iconColor: ready === desired ? 'success' as const : 'warning' as const },
            { label: 'Available', value: available, icon: CheckCircle2, iconColor: 'success' as const },
            { label: 'Fully Labeled', value: fullyLabeled, icon: Activity, iconColor: 'info' as const },
            { label: 'Age', value: ctx.age, icon: Clock, iconColor: 'primary' as const },
          ];
        }}
        headerMetadata={(ctx) => {
          const ownerRef = ctx.resource.metadata?.ownerReferences?.[0];
          return (
            <span className="flex items-center gap-1.5 ml-2 text-sm text-muted-foreground">
              <Clock className="h-3.5 w-3.5" />
              Created {ctx.age}
              {ownerRef && (
                <>
                  <span className="mx-2">&bull;</span>
                  Owner: {ownerRef.kind}
                </>
              )}
              {ctx.isConnected && <Badge variant="outline" className="ml-2 text-xs">Live</Badge>}
              {isFastPolling && (
                <Badge className="ml-2 text-xs bg-blue-500/20 text-blue-600 dark:text-blue-400 border-blue-500/30 animate-pulse gap-1">
                  <RefreshCw className="h-3 w-3 animate-spin" />
                  Syncing
                </Badge>
              )}
            </span>
          );
        }}
        extraHeaderActions={() => [
          { label: 'Scale', icon: Scale, variant: 'outline', onClick: () => setShowScaleDialog(true), className: 'press-effect' },
        ]}
        extraActionItems={() => [
          { icon: Scale, label: 'Scale ReplicaSet', description: 'Adjust the number of replicas', onClick: () => setShowScaleDialog(true), className: 'press-effect' },
        ]}
        extraDialogs={(ctx) => (
          <ScaleDialog
            open={showScaleDialog}
            onOpenChange={setShowScaleDialog}
            resourceType="ReplicaSet"
            resourceName={ctx.name}
            namespace={ctx.namespace}
            currentReplicas={ctx.resource.spec?.replicas || 0}
            onScale={handleScale}
          />
        )}
      />
    </>
  );
}
