import { useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Scale, Clock, TrendingUp, TrendingDown, Server, Cpu, Target, Activity, Zap } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { toast } from '@/components/ui/sonner';
import {
  GenericResourceDetail,
  SectionCard,
  DetailRow,
  LabelList,
  AnnotationList,
  type CustomTab,
  type ResourceContext,
} from '@/components/resources';
import { useResourceEvents, type EventInfo } from '@/hooks/useK8sResourceDetail';
import { type KubernetesResource } from '@/hooks/useKubernetes';
import { getDetailPath } from '@/utils/resourceKindMapper';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Button } from '@/components/ui/button';

interface HPAResource extends KubernetesResource {
  spec?: {
    scaleTargetRef?: { kind?: string; name?: string; apiVersion?: string };
    minReplicas?: number;
    maxReplicas?: number;
    metrics?: Array<{
      type?: string;
      resource?: { name?: string; target?: { type?: string; averageUtilization?: number } };
    }>;
  };
  status?: {
    currentReplicas?: number;
    desiredReplicas?: number;
    currentMetrics?: Array<{
      resource?: { name?: string; current?: { averageUtilization?: number } };
    }>;
    lastScaleTime?: string;
    conditions?: Array<{ type?: string; status?: string; reason?: string; message?: string }>;
  };
}

/** Parse scaling-related events into direction, old to new, timestamp. */
function parseScalingEvents(events: EventInfo[]): Array<{ direction: 'up' | 'down'; from: number | null; to: number; time: string; reason: string; message: string }> {
  const scaling: Array<{ direction: 'up' | 'down'; from: number | null; to: number; time: string; reason: string; message: string }> = [];
  const scaleRe = /scale|replica|size|rescale/i;
  const fromToRe = /(?:from|:)\s*(\d+)\s*(?:to|->|→)\s*(\d+)/i;
  const arrowRe = /(\d+)\s*(?:->|→)\s*(\d+)/;
  const newSizeRe = /new size:\s*(\d+)/i;
  for (const e of events) {
    if (!scaleRe.test(e.reason) && !scaleRe.test(e.message)) continue;
    const fromTo = e.message.match(fromToRe) ?? e.message.match(arrowRe);
    const newSize = e.message.match(newSizeRe);
    const isUp = /up|increase|expand|above/i.test(e.reason) || /up|increase|above/i.test(e.message);
    if (fromTo) {
      const from = parseInt(fromTo[1], 10);
      const to = parseInt(fromTo[2], 10);
      scaling.push({ direction: to >= from ? 'up' : 'down', from, to, time: e.time, reason: e.reason, message: e.message });
    } else if (newSize) {
      const to = parseInt(newSize[1], 10);
      scaling.push({ direction: isUp ? 'up' : 'down', from: null, to, time: e.time, reason: e.reason, message: e.message });
    }
  }
  return scaling;
}

// ---------------------------------------------------------------------------
// Custom tab components
// ---------------------------------------------------------------------------

function OverviewTab({ resource }: ResourceContext<HPAResource>) {
  const navigate = useNavigate();
  const ref = resource?.spec?.scaleTargetRef;
  const targetKind = ref?.kind ?? '–';
  const targetName = ref?.name ?? '–';
  const hpaNamespace = resource?.metadata?.namespace ?? '';
  const minReplicas = resource?.spec?.minReplicas ?? 1;
  const maxReplicas = resource?.spec?.maxReplicas ?? 1;
  const currentReplicas = resource?.status?.currentReplicas ?? 0;
  const desiredReplicas = resource?.status?.desiredReplicas ?? currentReplicas;
  const conditions = resource?.status?.conditions ?? [];
  const labels = resource?.metadata?.labels ?? {};
  const annotations = resource?.metadata?.annotations ?? {};

  const currentMetricsWithTarget = useMemo(() => {
    const metrics = resource?.spec?.metrics ?? [];
    const currentMetrics = resource?.status?.currentMetrics ?? [];
    return currentMetrics.map((cm) => {
      const name = cm.resource?.name ?? 'resource';
      const current = cm.resource?.current?.averageUtilization;
      const target = metrics.find((m) => m.resource?.name === name)?.resource?.target?.averageUtilization;
      return { name, current, target };
    });
  }, [resource?.spec?.metrics, resource?.status?.currentMetrics]);

  const targetLink = () => getDetailPath(targetKind, targetName, hpaNamespace) ?? '#';

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <SectionCard icon={Target} title="Scale Target">
          <div className="grid grid-cols-2 gap-x-8 gap-y-3">
            <DetailRow label="Reference" value={
              targetName !== '–' ? (
                <Button variant="link" className="p-0 h-auto font-mono text-primary" onClick={() => navigate(targetLink())}>{targetKind}/{targetName}</Button>
              ) : '–'
            } />
            <DetailRow label="Min Replicas" value={<span className="font-mono">{minReplicas}</span>} />
            <DetailRow label="Max Replicas" value={<span className="font-mono">{maxReplicas}</span>} />
            <DetailRow label="Desired Replicas" value={<span className="font-mono">{desiredReplicas}</span>} />
          </div>
          <div className="mt-4 space-y-3">
            <div className="flex items-center justify-between text-xs text-muted-foreground">
              <span className="font-mono">Min {minReplicas}</span>
              <span className="font-mono font-medium text-foreground">Current {currentReplicas}</span>
              <span className="font-mono">Max {maxReplicas}</span>
            </div>
            <div className="relative h-8 rounded-full bg-muted overflow-visible" role="slider" aria-valuemin={minReplicas} aria-valuemax={maxReplicas} aria-valuenow={currentReplicas}>
              <div
                className="absolute top-1/2 h-10 w-1 rounded-full bg-primary shadow-md border-2 border-background"
                style={{
                  left: maxReplicas > minReplicas
                    ? `${Math.min(100, Math.max(0, ((currentReplicas - minReplicas) / (maxReplicas - minReplicas)) * 100))}%`
                    : '50%',
                  transform: 'translateY(-50%) translateX(-50%)',
                }}
              />
            </div>
          </div>
      </SectionCard>
      <SectionCard icon={Cpu} title="Current Metrics">
          {currentMetricsWithTarget.length === 0 ? (
            <p className="text-muted-foreground text-sm">No current metrics reported yet.</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className="bg-muted/40">
                  <TableHead className="font-medium">Metric</TableHead>
                  <TableHead className="font-medium">Current</TableHead>
                  <TableHead className="font-medium">Target</TableHead>
                  <TableHead className="font-medium min-w-[120px]">Usage</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {currentMetricsWithTarget.map((row, i) => (
                  <TableRow key={i}>
                    <TableCell className="font-medium capitalize">{row.name}</TableCell>
                    <TableCell className="font-mono">{row.current != null ? `${row.current}%` : '–'}</TableCell>
                    <TableCell className="font-mono">{row.target != null ? `${row.target}%` : '–'}</TableCell>
                    <TableCell>
                      {row.target != null && row.current != null ? (
                        <Progress value={Math.min(Math.round((row.current / row.target) * 100), 100)} className="h-2" />
                      ) : (
                        <span className="text-muted-foreground text-sm">–</span>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
      </SectionCard>
      <SectionCard icon={Activity} title="Conditions" className="lg:col-span-2">
          {conditions.length === 0 ? (
            <p className="text-muted-foreground text-sm">No conditions.</p>
          ) : (
            <div className="space-y-2">
              {conditions.map((c, i) => (
                <div key={i} className="flex items-center justify-between p-3 rounded-lg bg-muted/50">
                  <span className="font-medium">{c.type}</span>
                  <div className="flex items-center gap-2">
                    <Badge variant={c.status === 'True' ? 'default' : 'secondary'}>{c.status}</Badge>
                    {c.reason && <span className="text-sm text-muted-foreground">{c.reason}</span>}
                  </div>
                </div>
              ))}
            </div>
          )}
      </SectionCard>
      <div className="lg:col-span-2">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <LabelList labels={labels} />
        </div>
      </div>
      <div className="lg:col-span-2">
        <AnnotationList annotations={annotations} />
      </div>
    </div>
  );
}

function ScalingHistoryTab({ resource, namespace }: ResourceContext<HPAResource>) {
  const hpaName = resource?.metadata?.name ?? '';
  const ns = resource?.metadata?.namespace ?? namespace;
  const { events } = useResourceEvents('HorizontalPodAutoscaler', ns ?? undefined, hpaName ?? undefined);
  const scalingEvents = useMemo(() => parseScalingEvents(events), [events]);

  if (scalingEvents.length === 0) {
    return <p className="text-muted-foreground text-sm">No scaling events recorded.</p>;
  }

  return (
    <SectionCard icon={TrendingUp} title="Scaling Events">
        <div className="space-y-3">
          {scalingEvents.map((ev, i) => (
            <div key={i} className="flex items-center gap-4 p-3 rounded-lg bg-muted/50">
              {ev.direction === 'up' ? (
                <TrendingUp className="h-5 w-5 text-emerald-600 shrink-0" aria-hidden />
              ) : (
                <TrendingDown className="h-5 w-5 text-amber-600 shrink-0" aria-hidden />
              )}
              <div className="flex-1 min-w-0">
                <span className="font-mono text-sm font-medium">
                  {ev.from != null ? `${ev.from} → ${ev.to}` : `→ ${ev.to}`}
                </span>
                <span className="text-muted-foreground text-sm ml-2">{ev.time}</span>
                {ev.reason && <Badge variant="secondary" className="ml-2 text-xs">{ev.reason}</Badge>}
              </div>
            </div>
          ))}
        </div>
    </SectionCard>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export default function HorizontalPodAutoscalerDetail() {
  const customTabs: CustomTab[] = [
    { id: 'overview', label: 'Overview', render: (ctx) => <OverviewTab {...ctx} /> },
    { id: 'scaling-events', label: 'Scaling History', render: (ctx) => <ScalingHistoryTab {...ctx} /> },
  ];

  return (
    <GenericResourceDetail<HPAResource>
      resourceType="horizontalpodautoscalers"
      kind="HorizontalPodAutoscaler"
      pluralLabel="HPAs"
      listPath="/horizontalpodautoscalers"
      resourceIcon={Scale}
      loadingCardCount={4}
      customTabs={customTabs}
      deriveStatus={(resource) => {
        const currentReplicas = resource?.status?.currentReplicas ?? 0;
        const desiredReplicas = resource?.status?.desiredReplicas ?? currentReplicas;
        return currentReplicas === desiredReplicas ? 'Healthy' : 'Warning';
      }}
      headerMetadata={(ctx) => (
        <span className="flex items-center gap-1.5 ml-2 text-sm text-muted-foreground">
          <Clock className="h-3.5 w-3.5" />Created {ctx.age}
        </span>
      )}
      extraHeaderActions={(ctx) => [
        { label: 'Edit', icon: TrendingUp, variant: 'outline', onClick: () => toast.info('Edit not implemented') },
      ]}
      extraActionItems={() => [
        { icon: TrendingUp, label: 'Edit Scaling', description: 'Modify min/max replicas and metrics', onClick: () => toast.info('Edit not implemented') },
      ]}
      buildStatusCards={(ctx) => {
        const resource = ctx.resource;
        const minReplicas = resource?.spec?.minReplicas ?? 1;
        const maxReplicas = resource?.spec?.maxReplicas ?? 1;
        const currentReplicas = resource?.status?.currentReplicas ?? 0;
        const desiredReplicas = resource?.status?.desiredReplicas ?? currentReplicas;
        const metrics = resource?.spec?.metrics ?? [];
        const currentMetrics = resource?.status?.currentMetrics ?? [];
        const cpuTarget = metrics.find((m) => m.resource?.name === 'cpu')?.resource?.target?.averageUtilization;
        const cpuCurrent = currentMetrics.find((m) => m.resource?.name === 'cpu')?.resource?.current?.averageUtilization;

        return [
          { label: 'Current / Desired', value: `${currentReplicas} / ${desiredReplicas}`, icon: Server, iconColor: 'primary' as const },
          { label: 'Min / Max', value: `${minReplicas} / ${maxReplicas}`, icon: Scale, iconColor: 'muted' as const },
          { label: 'CPU', value: cpuCurrent != null && cpuTarget != null ? `${cpuCurrent}% / ${cpuTarget}%` : '–', icon: Cpu, iconColor: 'info' as const },
          { label: 'Age', value: ctx.age, icon: Clock, iconColor: 'muted' as const },
        ];
      }}
    />
  );
}
