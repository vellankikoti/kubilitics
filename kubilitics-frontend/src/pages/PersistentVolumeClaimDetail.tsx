import { useState, useEffect, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Database, HardDrive, Server, Expand, Info, FolderOpen, AlertCircle, Loader2 } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { toast } from '@/components/ui/sonner';
import {
  GenericResourceDetail,
  SectionCard,
  DetailRow,
  LabelList,
  AnnotationList,
  PVCFileBrowser,
  type CustomTab,
  type ResourceContext,
  type ResourceStatus,
} from '@/components/resources';
import { useK8sResourceList, type KubernetesResource } from '@/hooks/useKubernetes';

interface K8sPVC extends KubernetesResource {
  spec?: {
    volumeName?: string;
    storageClassName?: string;
    accessModes?: string[];
    volumeMode?: string;
    resources?: { requests?: { storage?: string } };
  };
  status?: {
    phase?: string;
    capacity?: { storage?: string };
    accessModes?: string[];
  };
}

interface K8sPod extends KubernetesResource {
  spec?: {
    containers?: Array<{
      name: string;
      volumeMounts?: Array<{ name: string; mountPath: string }>;
    }>;
    initContainers?: Array<{
      name: string;
      volumeMounts?: Array<{ name: string; mountPath: string }>;
    }>;
    volumes?: Array<{
      name: string;
      persistentVolumeClaim?: { claimName: string };
    }>;
  };
  status?: {
    phase?: string;
  };
}

// ---------------------------------------------------------------------------
// Helpers: find a running pod that mounts a given PVC
// ---------------------------------------------------------------------------

interface MountInfo {
  podName: string;
  namespace: string;
  containerName: string;
  mountPath: string;
}

function findPVCMount(pods: K8sPod[], pvcName: string, pvcNamespace: string): MountInfo | null {
  for (const pod of pods) {
    if (pod.metadata?.namespace !== pvcNamespace) continue;
    if (pod.status?.phase !== 'Running') continue;

    const volumes = pod.spec?.volumes ?? [];
    const pvcVolume = volumes.find((v) => v.persistentVolumeClaim?.claimName === pvcName);
    if (!pvcVolume) continue;

    // Find the container + mountPath that uses this volume
    const allContainers = [...(pod.spec?.containers ?? []), ...(pod.spec?.initContainers ?? [])];
    for (const container of allContainers) {
      const mount = container.volumeMounts?.find((vm) => vm.name === pvcVolume.name);
      if (mount) {
        return {
          podName: pod.metadata?.name ?? '',
          namespace: pod.metadata?.namespace ?? '',
          containerName: container.name,
          mountPath: mount.mountPath,
        };
      }
    }
  }
  return null;
}

// ---------------------------------------------------------------------------
// Overview Tab
// ---------------------------------------------------------------------------

function OverviewTab({ resource: pvc, age }: ResourceContext<K8sPVC>) {
  const navigate = useNavigate();
  const capacity = pvc?.status?.capacity?.storage ?? pvc?.spec?.resources?.requests?.storage ?? '—';
  const accessModes = pvc?.spec?.accessModes ?? [];
  const storageClass = pvc?.spec?.storageClassName ?? '—';
  const volumeMode = pvc?.spec?.volumeMode ?? 'Filesystem';
  const volumeName = pvc?.spec?.volumeName ?? '—';
  const labels = pvc?.metadata?.labels ?? {};

  return (
    <div className="space-y-6">
      <SectionCard icon={Database} title="PVC information" tooltip={<p className="text-xs text-muted-foreground">Capacity, storage class, and access</p>}>
        <div className="grid grid-cols-2 gap-x-8 gap-y-3">
          <DetailRow label="Status" value={<Badge variant="outline">{pvc?.status?.phase ?? '—'}</Badge>} />
          <DetailRow label="Capacity" value={<Badge variant="secondary" className="font-mono">{capacity}</Badge>} />
          <DetailRow label="Volume Mode" value={volumeMode} />
          <DetailRow label="Storage Class" value={<Badge variant="outline">{storageClass}</Badge>} />
          <DetailRow label="Access Modes" value={<span className="font-mono">{accessModes.join(', ') || '—'}</span>} />
          <DetailRow label="Age" value={age} />
          {volumeName !== '—' && (
            <DetailRow
              label="Bound Volume"
              value={
                <Button
                  variant="link"
                  className="h-auto p-0 font-mono text-left break-all"
                  onClick={() => navigate(`/persistentvolumes/${volumeName}`)}
                >
                  {volumeName}
                </Button>
              }
            />
          )}
        </div>
      </SectionCard>
      <div className="lg:col-span-2">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <LabelList labels={labels} />
        </div>
      </div>
      <div className="lg:col-span-2">
        <AnnotationList annotations={pvc?.metadata?.annotations || {}} />
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Browse Files Tab
// ---------------------------------------------------------------------------

function BrowseFilesTab({ resource: pvc, namespace, backendBaseUrl, clusterId, isBackendConfigured }: ResourceContext<K8sPVC>) {
  const pvcName = pvc?.metadata?.name ?? '';
  const pvcNamespace = namespace || pvc?.metadata?.namespace || '';

  // Fetch pods in the same namespace to find one that mounts this PVC
  const { data: podsData, isLoading: podsLoading } = useK8sResourceList<K8sPod>(
    'pods',
    pvcNamespace,
    { enabled: !!pvcName && !!pvcNamespace }
  );

  const pods = useMemo(() => (podsData as K8sPod[] | undefined) ?? [], [podsData]);

  const mountInfo = useMemo(
    () => (pvcName ? findPVCMount(pods, pvcName, pvcNamespace) : null),
    [pods, pvcName, pvcNamespace]
  );

  if (!isBackendConfigured || !clusterId) {
    return (
      <div className="flex flex-col items-center justify-center py-16 gap-3">
        <div className="h-12 w-12 rounded-xl bg-slate-100 dark:bg-slate-800 flex items-center justify-center">
          <AlertCircle className="h-6 w-6 text-slate-400" />
        </div>
        <p className="text-sm font-medium text-slate-600 dark:text-slate-300">Backend not configured</p>
        <p className="text-xs text-muted-foreground">File browsing requires a connected backend.</p>
      </div>
    );
  }

  if (podsLoading) {
    return (
      <div className="flex flex-col items-center justify-center py-16 gap-2 text-muted-foreground">
        <Loader2 className="h-6 w-6 animate-spin text-primary/60" />
        <span className="text-sm">Searching for pods mounting this PVC...</span>
      </div>
    );
  }

  if (!mountInfo) {
    return (
      <div className="flex flex-col items-center justify-center py-16 gap-3">
        <div className="h-12 w-12 rounded-xl bg-amber-50 dark:bg-amber-900/20 flex items-center justify-center">
          <AlertCircle className="h-6 w-6 text-amber-500" />
        </div>
        <p className="text-sm font-medium text-slate-600 dark:text-slate-300">No running pod found</p>
        <p className="text-xs text-muted-foreground text-center max-w-sm">
          File browsing requires a running pod that mounts this PVC.
          Deploy a workload that uses <span className="font-mono">{pvcName}</span> to browse its contents.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {/* Info bar: which pod/container/path */}
      <div className="flex items-center gap-2 px-1 text-xs text-muted-foreground">
        <span>
          Browsing via pod{' '}
          <span className="font-mono font-medium text-slate-700 dark:text-slate-300">{mountInfo.podName}</span>
          {' / '}
          <span className="font-mono text-slate-600 dark:text-slate-400">{mountInfo.containerName}</span>
          {' at '}
          <span className="font-mono text-slate-600 dark:text-slate-400">{mountInfo.mountPath}</span>
        </span>
      </div>

      <div className="border border-slate-200/60 dark:border-slate-700/60 rounded-lg overflow-hidden bg-white dark:bg-slate-900">
        <PVCFileBrowser
          podName={mountInfo.podName}
          namespace={mountInfo.namespace}
          containerName={mountInfo.containerName}
          mountPath={mountInfo.mountPath}
          baseUrl={backendBaseUrl}
          clusterId={clusterId}
        />
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function PersistentVolumeClaimDetail() {
  const customTabs: CustomTab[] = [
    { id: 'overview', label: 'Overview', icon: Info, render: (ctx) => <OverviewTab {...ctx} /> },
    { id: 'browse', label: 'Browse Files', icon: FolderOpen, render: (ctx) => <BrowseFilesTab {...ctx} /> },
  ];

  return (
    <GenericResourceDetail<K8sPVC>
      resourceType="persistentvolumeclaims"
      kind="PersistentVolumeClaim"
      pluralLabel="Persistent Volume Claims"
      listPath="/persistentvolumeclaims"
      resourceIcon={Database}
      loadingCardCount={5}
      customTabs={customTabs}
      deriveStatus={(pvc) => (pvc?.status?.phase ?? 'Unknown') as ResourceStatus}
      extraHeaderActions={() => [
        { label: 'Expand', icon: Expand, variant: 'outline', onClick: () => toast.info('Expand requires backend support'), className: 'press-effect' },
      ]}
      extraActionItems={() => [
        { icon: Expand, label: 'Expand Volume', description: 'Increase the storage capacity', onClick: () => toast.info('Expand requires backend support'), className: 'press-effect' },
      ]}
      buildStatusCards={(ctx) => {
        const pvc = ctx.resource;
        const requestedCapacity = pvc?.spec?.resources?.requests?.storage ?? '—';
        const usedCapacity = pvc?.status?.capacity?.storage ?? '—';
        const volumeName = pvc?.spec?.volumeName ?? '—';

        return [
          { label: 'Status', value: pvc?.status?.phase ?? '—', icon: Database, iconColor: 'primary' as const },
          { label: 'Capacity', value: requestedCapacity, icon: HardDrive, iconColor: 'info' as const },
          { label: 'Used', value: usedCapacity, icon: HardDrive, iconColor: 'muted' as const },
          { label: 'Volume', value: volumeName, icon: Server, iconColor: 'muted' as const },
          { label: 'Used By', value: '—', icon: Database, iconColor: 'muted' as const },
        ];
      }}
    />
  );
}
