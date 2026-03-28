import { Cpu, Clock, Info, Layers } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Link } from 'react-router-dom';
import {
  GenericResourceDetail,
  SectionCard,
  DetailRow,
  LabelList,
  AnnotationList,
  type CustomTab,
  type ResourceContext,
} from '@/components/resources';
import { type KubernetesResource } from '@/hooks/useKubernetes';

interface K8sResourceSlice extends KubernetesResource {
  driver?: string;
  nodeName?: string;
  pool?: { name?: string; generation?: number; resourceSliceCount?: number };
  namedResources?: unknown;
  structuredResources?: unknown;
}

function formatCapacity(rs: K8sResourceSlice): string {
  const named = rs.namedResources as { entries?: Array<{ capacity?: Record<string, string> }> } | undefined;
  const structured = rs.structuredResources as { capacity?: Record<string, string> } | undefined;
  if (named?.entries?.length) {
    const caps = named.entries.flatMap((e) => e.capacity ? Object.values(e.capacity) : []);
    return caps.length ? caps.join(', ') : '—';
  }
  if (structured?.capacity && Object.keys(structured.capacity).length) {
    return Object.entries(structured.capacity).map(([k, v]) => `${k}: ${v}`).join(', ');
  }
  return '—';
}

function OverviewTab({ resource: rs, age }: ResourceContext<K8sResourceSlice>) {
  const driver = rs.driver ?? (rs.spec as Record<string, unknown> | undefined)?.driver as string ?? '—';
  const nodeName = rs.nodeName ?? (rs.spec as Record<string, unknown> | undefined)?.nodeName as string | undefined;
  const pool = rs.pool ?? (rs.spec as Record<string, unknown> | undefined)?.pool as K8sResourceSlice['pool'];
  const poolName = pool?.name ?? '—';
  const node = nodeName ?? poolName ?? '—';
  const capacity = formatCapacity(rs as K8sResourceSlice);

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <SectionCard icon={Cpu} title="Resource Slice" tooltip={<p className="text-xs text-muted-foreground">DRA capacity info</p>}>
        <div className="grid grid-cols-2 gap-x-8 gap-y-3">
          <DetailRow label="Driver" value={<span className="font-mono">{driver}</span>} />
          <DetailRow label="Node" value={node !== '—' ? <Link to={`/nodes/${node}`} className="text-primary hover:underline font-mono">{node}</Link> : '—'} />
          <DetailRow label="Pool" value={<span className="font-mono">{poolName}</span>} />
          {pool?.generation != null && <DetailRow label="Generation" value={<Badge variant="outline">{pool.generation}</Badge>} />}
          {pool?.resourceSliceCount != null && <DetailRow label="Slices in Pool" value={String(pool.resourceSliceCount)} />}
          <DetailRow label="Capacity" value={<span className="font-mono">{capacity}</span>} />
          <DetailRow label="Age" value={age} />
        </div>
      </SectionCard>
      <div className="lg:col-span-2">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <LabelList labels={rs?.metadata?.labels ?? {}} />
        </div>
      </div>
      <div className="lg:col-span-2">
        <AnnotationList annotations={rs?.metadata?.annotations ?? {}} />
      </div>
    </div>
  );
}

export default function ResourceSliceDetail() {
  const customTabs: CustomTab[] = [
    { id: 'overview', label: 'Overview', icon: Info, render: (ctx) => <OverviewTab {...ctx} /> },
  ];

  return (
    <GenericResourceDetail<K8sResourceSlice>
      resourceType="resourceslices"
      kind="ResourceSlice"
      pluralLabel="Resource Slices"
      listPath="/resourceslices"
      resourceIcon={Cpu}
      customTabs={customTabs}
      buildStatusCards={(ctx) => {
        const rs = ctx.resource;
        const driver = rs.driver ?? (rs.spec as Record<string, unknown> | undefined)?.driver as string ?? '—';
        const nodeName = rs.nodeName ?? (rs.spec as Record<string, unknown> | undefined)?.nodeName as string | undefined;
        const pool = rs.pool ?? (rs.spec as Record<string, unknown> | undefined)?.pool as K8sResourceSlice['pool'];
        const poolName = pool?.name ?? '—';
        const node = nodeName ?? poolName ?? '—';
        const capacity = formatCapacity(rs as K8sResourceSlice);

        return [
          { label: 'Driver', value: driver, icon: Cpu, iconColor: 'primary' as const },
          { label: 'Node', value: node, icon: Layers, iconColor: 'info' as const },
          { label: 'Pool', value: poolName, icon: Layers, iconColor: 'muted' as const },
          { label: 'Capacity', value: capacity, icon: Cpu, iconColor: 'muted' as const },
        ];
      }}
    />
  );
}
