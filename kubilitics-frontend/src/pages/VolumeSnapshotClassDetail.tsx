import { Layers, Camera, Server, Settings, Star, Info } from 'lucide-react';
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

interface K8sVolumeSnapshotClass extends KubernetesResource {
  driver?: string;
  deletionPolicy?: string;
}

function OverviewTab({ resource: vsc, age }: ResourceContext<K8sVolumeSnapshotClass>) {
  const vscName = vsc?.metadata?.name ?? '';
  const driver = (vsc as K8sVolumeSnapshotClass & { spec?: { driver?: string } })?.driver ?? (vsc.spec as Record<string, unknown> | undefined)?.driver as string | undefined ?? '—';
  const deletionPolicy = (vsc as K8sVolumeSnapshotClass & { spec?: { deletionPolicy?: string } })?.deletionPolicy ?? (vsc.spec as Record<string, unknown> | undefined)?.deletionPolicy as string | undefined ?? 'Delete';
  const isDefault = vsc?.metadata?.annotations?.['snapshot.storage.kubernetes.io/is-default-class'] === 'true';

  return (
    <div className="space-y-6">
      <SectionCard icon={Layers} title="Volume Snapshot Class" tooltip={<p className="text-xs text-muted-foreground">CSI snapshot parameters</p>}>
        <div className="grid grid-cols-2 gap-x-8 gap-y-3">
          <DetailRow label="Driver" value={<span className="font-mono text-xs">{driver}</span>} />
          <DetailRow label="Deletion Policy" value={<Badge variant="outline">{deletionPolicy}</Badge>} />
          <DetailRow label="Default Class" value={<Badge variant={isDefault ? 'default' : 'secondary'}>{isDefault ? 'Yes' : 'No'}</Badge>} />
          <DetailRow label="Age" value={age} />
        </div>
      </SectionCard>
      <SectionCard icon={Layers} title="Usage" tooltip={<p className="text-xs text-muted-foreground">VolumeSnapshots using this class</p>}>
        <p className="text-sm text-muted-foreground">
          VolumeSnapshots reference this class via <code className="bg-muted px-1 rounded">spec.volumeSnapshotClassName: {vscName}</code>.
          View <Link to="/volumesnapshots" className="text-primary hover:underline">Volume Snapshots</Link> and filter by snapshot class to see usage.
        </p>
      </SectionCard>
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

export default function VolumeSnapshotClassDetail() {
  const customTabs: CustomTab[] = [
    { id: 'overview', label: 'Overview', icon: Info, render: (ctx) => <OverviewTab {...ctx} /> },
  ];

  return (
    <GenericResourceDetail<K8sVolumeSnapshotClass>
      resourceType="volumesnapshotclasses"
      kind="VolumeSnapshotClass"
      pluralLabel="Volume Snapshot Classes"
      listPath="/volumesnapshotclasses"
      resourceIcon={Camera}
      loadingCardCount={3}
      customTabs={customTabs}
      deriveStatus={() => 'Healthy'}
      buildStatusCards={(ctx) => {
        const vsc = ctx.resource;
        const driver = (vsc as K8sVolumeSnapshotClass & { spec?: { driver?: string } })?.driver ?? (vsc.spec as Record<string, unknown> | undefined)?.driver as string | undefined ?? '—';
        const deletionPolicy = (vsc as K8sVolumeSnapshotClass & { spec?: { deletionPolicy?: string } })?.deletionPolicy ?? (vsc.spec as Record<string, unknown> | undefined)?.deletionPolicy as string | undefined ?? 'Delete';
        const isDefault = vsc?.metadata?.annotations?.['snapshot.storage.kubernetes.io/is-default-class'] === 'true';

        return [
          { label: 'Driver', value: driver, icon: Server, iconColor: 'primary' as const },
          { label: 'Deletion Policy', value: deletionPolicy, icon: Settings, iconColor: 'info' as const },
          { label: 'Default Class', value: isDefault ? 'Yes' : 'No', icon: Star, iconColor: 'muted' as const },
        ];
      }}
    />
  );
}
