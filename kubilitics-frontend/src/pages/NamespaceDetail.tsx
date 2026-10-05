import { useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Folder, Clock, Box, Globe, Settings, Layers, Package, Database, Shield, Activity, Network, BarChart2, Boxes, HardDrive, ArrowUpRight, Zap, Gauge } from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import {
  GenericResourceDetail,
  SectionCard,
  DetailRow,
  LabelList,
  AnnotationList,
  type CustomTab,
  type ResourceContext,
} from '@/components/resources';
import { useK8sResourceList, type KubernetesResource } from '@/hooks/useKubernetes';
import { useConnectionStatus } from '@/hooks/useConnectionStatus';

interface NamespaceResource extends KubernetesResource {
  spec?: {
    finalizers?: string[];
  };
  status?: {
    phase?: string;
  };
}

// ---------------------------------------------------------------------------
// Custom tab components
// ---------------------------------------------------------------------------

function OverviewTab({ resource: ns, age, resourceCounts, resourceQuotas }: ResourceContext<NamespaceResource> & { resourceCounts: Record<string, string | number>; resourceQuotas: KubernetesResource[] }) {
  const navigate = useNavigate();
  const nsName = ns?.metadata?.name ?? '';
  const labels = ns?.metadata?.labels || {};
  const annotations = ns?.metadata?.annotations || {};
  const phase = ns?.status?.phase || 'Active';
  // `|| []` only protects against falsy values — a malformed API response
  // with finalizers serialized as `{}` instead of `[]` is truthy and would
  // slip through, so guard the shape explicitly.
  const finalizers = Array.isArray(ns?.spec?.finalizers) ? ns.spec.finalizers : [];
  const hasQuota = Array.isArray(resourceQuotas) && resourceQuotas.length > 0;

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Namespace Info */}
        <SectionCard icon={Folder} title="Namespace Info" tooltip="Phase, finalizers, labels, annotations">
          <div className="grid grid-cols-2 gap-x-8 gap-y-3">
            <DetailRow label="Phase" value={<Badge variant={phase === 'Active' ? 'default' : 'secondary'} className={phase === 'Active' ? 'bg-emerald-600 hover:bg-emerald-600' : ''}>{phase}</Badge>} />
            <DetailRow label="Age" value={age} />
            <DetailRow label="Finalizers" value={finalizers.length ? finalizers.join(', ') : 'None'} />
            <DetailRow label="Labels" value={`${Object.keys(labels).length} labels`} />
            <DetailRow label="Annotations" value={`${Object.keys(annotations).length} annotations`} />
            {hasQuota && (
              <DetailRow label="Resource Quotas" value={`${resourceQuotas.length} quota${resourceQuotas.length !== 1 ? 's' : ''}`} />
            )}
          </div>
        </SectionCard>

        {/* Labels */}
        <div className="lg:col-span-2">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <LabelList labels={labels} />
          </div>
        </div>
        <div className="lg:col-span-2">
          <AnnotationList annotations={annotations} />
        </div>

        {/* Resource Summary — full grid */}
        <SectionCard icon={Boxes} title="Resource Summary" tooltip="Counts of all resource types in this namespace" className="lg:col-span-2">
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
            {([
              { label: 'Pods', value: resourceCounts.pods, icon: Package, path: '/pods', color: 'text-blue-600 dark:text-blue-400', bg: 'bg-blue-500/10' },
              { label: 'Deployments', value: resourceCounts.deployments, icon: Layers, path: '/deployments', color: 'text-indigo-600 dark:text-indigo-400', bg: 'bg-indigo-500/10' },
              { label: 'ReplicaSets', value: resourceCounts.replicasets, icon: Boxes, path: '/replicasets', color: 'text-violet-600 dark:text-violet-400', bg: 'bg-violet-500/10' },
              { label: 'StatefulSets', value: resourceCounts.statefulsets, icon: Database, path: '/statefulsets', color: 'text-purple-600 dark:text-purple-400', bg: 'bg-purple-500/10' },
              { label: 'DaemonSets', value: resourceCounts.daemonsets, icon: Activity, path: '/daemonsets', color: 'text-fuchsia-600 dark:text-fuchsia-400', bg: 'bg-fuchsia-500/10' },
              { label: 'Jobs', value: resourceCounts.jobs, icon: BarChart2, path: '/jobs', color: 'text-cyan-600 dark:text-cyan-400', bg: 'bg-cyan-500/10' },
              { label: 'Services', value: resourceCounts.services, icon: Globe, path: '/services', color: 'text-emerald-600 dark:text-emerald-400', bg: 'bg-emerald-500/10' },
              { label: 'Ingresses', value: resourceCounts.ingresses, icon: Network, path: '/ingresses', color: 'text-teal-600 dark:text-teal-400', bg: 'bg-teal-500/10' },
              { label: 'ConfigMaps', value: resourceCounts.configmaps, icon: Settings, path: '/configmaps', color: 'text-amber-600 dark:text-amber-400', bg: 'bg-amber-500/10' },
              { label: 'Secrets', value: resourceCounts.secrets, icon: Shield, path: '/secrets', color: 'text-rose-600 dark:text-rose-400', bg: 'bg-rose-500/10' },
              { label: 'PVCs', value: resourceCounts.pvcs, icon: HardDrive, path: '/persistentvolumeclaims', color: 'text-orange-600 dark:text-orange-400', bg: 'bg-orange-500/10' },
            ] as const).map(({ label, value, icon: Icon, path, color, bg }) => (
              <button
                key={label}
                className="group relative p-4 rounded-xl border border-border/50 bg-card hover:border-primary/30 hover:shadow-md transition-all duration-200 text-left cursor-pointer"
                onClick={() => navigate(`${path}?namespace=${encodeURIComponent(nsName)}`)}
              >
                <div className="flex items-center justify-between mb-3">
                  <div className={`h-8 w-8 rounded-lg ${bg} flex items-center justify-center`}>
                    <Icon className={`h-4 w-4 ${color}`} />
                  </div>
                  <ArrowUpRight className="h-3.5 w-3.5 text-muted-foreground/0 group-hover:text-primary group-hover:text-muted-foreground transition-all" />
                </div>
                <p className="text-2xl font-bold tabular-nums text-foreground">{value}</p>
                <p className="text-[11px] font-medium text-muted-foreground mt-0.5">{label}</p>
              </button>
            ))}
          </div>
        </SectionCard>
      </div>
    </div>
  );
}

function QuotaTab({ resourceQuotas }: { resourceQuotas: KubernetesResource[] }) {
  const hasQuota = resourceQuotas.length > 0;

  return (
    <div className="space-y-6">
      {!hasQuota ? (
        <p className="text-muted-foreground">No resource quotas in this namespace.</p>
      ) : (
        resourceQuotas.map((rq: KubernetesResource) => (
          <SectionCard key={rq.metadata?.uid} icon={Gauge} title={`Resource Quota: ${rq.metadata?.name}`} tooltip="Resource usage limits for this namespace">
              <pre className="text-xs font-mono bg-muted/50 p-4 rounded-lg overflow-auto">
                {JSON.stringify((rq as unknown as Record<string, unknown>).status ?? (rq as unknown as Record<string, unknown>).spec ?? {}, null, 2)}
              </pre>
          </SectionCard>
        ))
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export default function NamespaceDetail() {
  const { name } = useParams();
  const { isConnected } = useConnectionStatus();

  const nsName = name ?? '';
  const queryOpts = { enabled: !!nsName && isConnected, limit: 500 };
  const podsList = useK8sResourceList<KubernetesResource>('pods', nsName, { ...queryOpts, limit: 5000 });
  const deploymentsList = useK8sResourceList<KubernetesResource>('deployments', nsName, queryOpts);
  const servicesList = useK8sResourceList<KubernetesResource>('services', nsName, queryOpts);
  const configMapsList = useK8sResourceList<KubernetesResource>('configmaps', nsName, queryOpts);
  const secretsList = useK8sResourceList<KubernetesResource>('secrets', nsName, queryOpts);
  const replicaSetsList = useK8sResourceList<KubernetesResource>('replicasets', nsName, queryOpts);
  const statefulSetsList = useK8sResourceList<KubernetesResource>('statefulsets', nsName, queryOpts);
  const daemonSetsList = useK8sResourceList<KubernetesResource>('daemonsets', nsName, queryOpts);
  const jobsList = useK8sResourceList<KubernetesResource>('jobs', nsName, queryOpts);
  const ingressesList = useK8sResourceList<KubernetesResource>('ingresses', nsName, queryOpts);
  const pvcList = useK8sResourceList<KubernetesResource>('persistentvolumeclaims', nsName, queryOpts);
  const resourceQuotasList = useK8sResourceList<KubernetesResource>('resourcequotas', nsName, { ...queryOpts, limit: 100 });

  const getCount = (q: { data?: { items?: unknown[] } }) => q.data?.items?.length;
  const resourceCounts = useMemo(() => {
    if (!isConnected || !nsName) return {
      pods: '–', deployments: '–', services: '–', configmaps: '–', secrets: '–',
      replicasets: '–', statefulsets: '–', daemonsets: '–', jobs: '–', ingresses: '–', pvcs: '–',
    };
    return {
      pods: getCount(podsList) ?? '–',
      deployments: getCount(deploymentsList) ?? '–',
      services: getCount(servicesList) ?? '–',
      configmaps: getCount(configMapsList) ?? '–',
      secrets: getCount(secretsList) ?? '–',
      replicasets: getCount(replicaSetsList) ?? '–',
      statefulsets: getCount(statefulSetsList) ?? '–',
      daemonsets: getCount(daemonSetsList) ?? '–',
      jobs: getCount(jobsList) ?? '–',
      ingresses: getCount(ingressesList) ?? '–',
      pvcs: getCount(pvcList) ?? '–',
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isConnected, nsName, podsList.data, deploymentsList.data, servicesList.data, configMapsList.data, secretsList.data, replicaSetsList.data, statefulSetsList.data, daemonSetsList.data, jobsList.data, ingressesList.data, pvcList.data]);

  const resourceQuotas = useMemo(() => resourceQuotasList.data?.items ?? [], [resourceQuotasList.data?.items]);

  const totalResources = Object.values(resourceCounts).reduce((sum, v) => sum + (typeof v === 'number' ? v : 0), 0);

  const customTabs: CustomTab[] = [
    { id: 'overview', label: 'Overview', render: (ctx) => <OverviewTab {...ctx} resourceCounts={resourceCounts} resourceQuotas={resourceQuotas} /> },
    { id: 'quotas', label: 'Quota Status', render: () => <QuotaTab resourceQuotas={resourceQuotas} /> },
  ];

  return (
    <GenericResourceDetail<NamespaceResource>
      resourceType="namespaces"
      kind="Namespace"
      pluralLabel="Namespaces"
      listPath="/namespaces"
      resourceIcon={Folder}
      loadingCardCount={4}
      customTabs={customTabs}
      deriveStatus={(ns) => {
        const phase = ns?.status?.phase || 'Active';
        return phase === 'Active' ? 'Healthy' : phase === 'Terminating' ? 'Warning' : 'Unknown';
      }}
      headerMetadata={(ctx) => (
        <span className="flex items-center gap-1.5 ml-2 text-sm text-muted-foreground">
          <Clock className="h-3.5 w-3.5" />Created {ctx.age}
          <span className="mx-2">•</span>
          <Badge variant={(ctx.resource?.status?.phase || 'Active') === 'Active' ? 'default' : 'secondary'}>{ctx.resource?.status?.phase || 'Active'}</Badge>
          {ctx.isConnected && <Badge variant="outline" className="ml-2 text-xs">Live</Badge>}
        </span>
      )}
      buildStatusCards={(ctx) => {
        const phase = ctx.resource?.status?.phase || 'Active';
        return [
          { label: 'Status', value: phase, icon: Box, iconColor: (phase === 'Active' ? 'success' : 'warning') as const },
          { label: 'Pods', value: String(resourceCounts.pods), icon: Package, iconColor: 'primary' as const },
          { label: 'Deployments', value: String(resourceCounts.deployments), icon: Layers, iconColor: 'info' as const },
          { label: 'Services', value: String(resourceCounts.services), icon: Globe, iconColor: 'success' as const },
          { label: 'Total Resources', value: totalResources > 0 ? String(totalResources) : '–', icon: Boxes, iconColor: 'primary' as const },
          { label: 'Age', value: ctx.age, icon: Clock, iconColor: 'muted' as const },
        ];
      }}
    />
  );
}
