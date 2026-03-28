import { Layers, Camera, Settings, Info } from 'lucide-react';
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

interface K8sVolumeSnapshotContent extends KubernetesResource {
  spec?: {
    source?: { volumeHandle?: string; snapshotHandle?: string };
    volumeSnapshotClassName?: string;
    deletionPolicy?: string;
    driver?: string;
    volumeSnapshotRef?: { namespace?: string; name?: string };
  };
  status?: {
    readyToUse?: boolean;
    restoreSize?: string;
    snapshotHandle?: string;
    error?: { message?: string };
  };
}

function OverviewTab({ resource: vsc, age }: ResourceContext<K8sVolumeSnapshotContent>) {
  const spec = vsc?.spec ?? {};
  const status = vsc?.status ?? {};
  const sourceSpec = spec.source ?? {};
  const vsRef = spec.volumeSnapshotRef ?? {};

  const driver = vsc?.spec?.driver ?? (vsc as unknown as Record<string, unknown>)?.driver as string ?? '—';
  const deletionPolicy = vsc?.spec?.deletionPolicy ?? (vsc as unknown as Record<string, unknown>)?.deletionPolicy as string ?? 'Delete';
  const snapshotClass = spec.volumeSnapshotClassName ?? '—';
  const restoreSize = status.restoreSize ?? '—';
  const readyToUse = status.readyToUse === true;
  const errorMsg = status.error?.message;

  let sourceLabel = '—';
  if (sourceSpec.snapshotHandle) sourceLabel = 'Pre-provisioned';
  else if (sourceSpec.volumeHandle) sourceLabel = 'Dynamic (from PVC)';

  return (
    <div className="space-y-6">
      <SectionCard icon={Layers} title="Volume Snapshot Content" tooltip={<p className="text-xs text-muted-foreground">Actual snapshot data binding</p>}>
        <div className="grid grid-cols-2 gap-x-8 gap-y-3">
          <DetailRow label="Status" value={<Badge variant={readyToUse ? 'default' : (errorMsg ? 'destructive' : 'secondary')}>{readyToUse ? 'Ready' : (errorMsg ? 'Failed' : 'Pending')}</Badge>} />
          <DetailRow label="Source" value={sourceLabel} />
          <DetailRow label="Driver" value={<span className="font-mono text-xs">{driver}</span>} />
          <DetailRow label="Snapshot Class" value={snapshotClass !== '—' ? <Link to={`/volumesnapshotclasses/${snapshotClass}`} className="text-primary hover:underline font-mono text-xs">{snapshotClass}</Link> : <span>—</span>} />
          <DetailRow label="Restore Size" value={<span className="font-mono">{restoreSize}</span>} />
          <DetailRow label="Deletion Policy" value={<Badge variant="outline">{deletionPolicy}</Badge>} />
          <DetailRow label="Age" value={age} />
        </div>
        {errorMsg && (
          <div className="mt-4 p-3 rounded-lg bg-destructive/10 text-destructive text-sm">
            <p className="font-medium">Error</p>
            <p className="font-mono text-xs mt-1">{errorMsg}</p>
          </div>
        )}
      </SectionCard>
      {vsRef.namespace && vsRef.name && (
        <SectionCard icon={Layers} title="Bound VolumeSnapshot" tooltip={<p className="text-xs text-muted-foreground">The VolumeSnapshot this content is bound to</p>}>
          <p className="text-sm text-muted-foreground mb-2">
            This VolumeSnapshotContent is bound to the following VolumeSnapshot:
          </p>
          <Link to={`/volumesnapshots/${vsRef.namespace}/${vsRef.name}`} className="text-primary hover:underline font-mono">
            {vsRef.namespace}/{vsRef.name}
          </Link>
        </SectionCard>
      )}
      <div className="lg:col-span-2">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <LabelList labels={vsc?.metadata?.labels ?? {}} />
        </div>
      </div>
      <div className="lg:col-span-2">
        <AnnotationList annotations={vsc?.metadata?.annotations ?? {}} />
      </div>
    </div>
  );
}

export default function VolumeSnapshotContentDetail() {
  const customTabs: CustomTab[] = [
    { id: 'overview', label: 'Overview', icon: Info, render: (ctx) => <OverviewTab {...ctx} /> },
  ];

  return (
    <GenericResourceDetail<K8sVolumeSnapshotContent>
      resourceType="volumesnapshotcontents"
      kind="VolumeSnapshotContent"
      pluralLabel="Volume Snapshot Contents"
      listPath="/volumesnapshotcontents"
      resourceIcon={Camera}
      loadingCardCount={5}
      customTabs={customTabs}
      deriveStatus={(vsc) => {
        const errorMsg = vsc?.status?.error?.message;
        const readyToUse = vsc?.status?.readyToUse === true;
        return errorMsg ? 'Error' : readyToUse ? 'Healthy' : 'Warning';
      }}
      buildStatusCards={(ctx) => {
        const vsc = ctx.resource;
        const spec = vsc?.spec ?? {};
        const status = vsc?.status ?? {};
        const sourceSpec = spec.source ?? {};
        const snapshotClass = spec.volumeSnapshotClassName ?? '—';
        const restoreSize = status.restoreSize ?? '—';
        const readyToUse = status.readyToUse === true;
        const errorMsg = status.error?.message;
        const deletionPolicy = vsc?.spec?.deletionPolicy ?? (vsc as unknown as Record<string, unknown>)?.deletionPolicy as string ?? 'Delete';

        let sourceLabel = '—';
        if (sourceSpec.snapshotHandle) sourceLabel = 'Pre-provisioned';
        else if (sourceSpec.volumeHandle) sourceLabel = 'Dynamic (from PVC)';

        return [
          { label: 'Status', value: readyToUse ? 'Ready' : (errorMsg ? 'Failed' : 'Pending'), icon: Camera, iconColor: 'primary' as const },
          { label: 'Source', value: sourceLabel, icon: Layers, iconColor: 'info' as const },
          { label: 'Snapshot Class', value: snapshotClass, icon: Settings, iconColor: 'muted' as const },
          { label: 'Restore Size', value: restoreSize, icon: Layers, iconColor: 'muted' as const },
          { label: 'Deletion Policy', value: deletionPolicy, icon: Settings, iconColor: 'muted' as const },
        ];
      }}
    />
  );
}
