import { useState, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { KeyRound, Clock, Copy, Info, Eye, EyeOff, Edit } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import {
  GenericResourceDetail,
  SectionCard,
  LabelList,
  AnnotationList,
  DetailRow,
  type CustomTab,
  type ResourceContext,
} from '@/components/resources';
import { type KubernetesResource } from '@/hooks/useKubernetes';
import { useBackendConfigStore, getEffectiveBackendBaseUrl } from '@/stores/backendConfigStore';
import { useActiveClusterId } from '@/hooks/useActiveClusterId';
import { getSecretConsumers, getSecretTLSInfo } from '@/services/backendApiClient';
import { toast } from '@/components/ui/sonner';
import { cn } from '@/lib/utils';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

interface SecretResource extends KubernetesResource {
  type?: string;
  data?: Record<string, string>;
  stringData?: Record<string, string>;
  immutable?: boolean;
}

function daysRemainingColor(days: number): string {
  if (days < 0) return 'bg-red-900/30 text-red-900 dark:bg-red-950/50 dark:text-red-400';
  if (days <= 7) return 'bg-red-500/20 text-red-700 dark:text-red-400';
  if (days <= 30) return 'bg-amber-500/20 text-amber-700 dark:text-amber-400';
  return 'bg-emerald-500/20 text-emerald-600';
}

function OverviewTab({ resource: s, age }: ResourceContext<SecretResource>) {
  const data = s.data || {};
  const secretType = s.type || 'Opaque';
  const labels = s.metadata?.labels || {};
  const sName = s.metadata?.name || '';
  const sNamespace = s.metadata?.namespace || '';

  return (
    <div className="space-y-6">
      <SectionCard
        icon={Info}
        title="Secret information"
        tooltip={<p className="text-xs text-muted-foreground">Identity and metadata for this Secret</p>}
      >
        <div className="grid grid-cols-2 gap-x-8 gap-y-3">
          <DetailRow label="Name" value={sName} />
          <DetailRow label="Namespace" value={sNamespace} />
          <DetailRow label="Type" value={<Badge variant="secondary">{secretType}</Badge>} />
          <DetailRow label="Age" value={age} />
          <DetailRow label="Data Keys" value={String(Object.keys(data).length)} />
          <DetailRow label="Immutable" value={<Badge variant="outline">{s.immutable ? 'Yes' : 'No'}</Badge>} />
        </div>
      </SectionCard>
      <SectionCard
        icon={KeyRound}
        title="Data keys"
        tooltip={<p className="text-xs text-muted-foreground">Keys defined in this Secret (see Data tab for values)</p>}
      >
        <div className="flex flex-wrap gap-2">
          {Object.keys(data).map((key) => (
            <Badge key={key} variant="secondary" className="font-mono">{key}</Badge>
          ))}
          {Object.keys(data).length === 0 && <p className="text-muted-foreground text-sm">No keys</p>}
        </div>
      </SectionCard>
      <div className="lg:col-span-2">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <LabelList labels={labels} />
        </div>
      </div>
      <div className="lg:col-span-2">
        <AnnotationList annotations={s?.metadata?.annotations || {}} />
      </div>
    </div>
  );
}

function DataTab({ resource: s }: ResourceContext<SecretResource>) {
  const [showValues, setShowValues] = useState<Record<string, boolean>>({});
  const data = s.data || {};
  const secretType = s.type || 'Opaque';

  const clusterId = useActiveClusterId();
  const isBackendConfigured = useBackendConfigStore((s) => s.isBackendConfigured)();
  const storedUrl = useBackendConfigStore((s) => s.backendBaseUrl);
  const baseUrl = getEffectiveBackendBaseUrl(storedUrl);
  const namespace = s.metadata?.namespace ?? '';
  const name = s.metadata?.name ?? '';

  const tlsInfoQuery = useQuery({
    queryKey: ['secret-tls-info', clusterId, namespace, name],
    queryFn: () => getSecretTLSInfo(baseUrl!, clusterId!, namespace, name),
    enabled: !!(isBackendConfigured && clusterId && namespace && name && (secretType === 'kubernetes.io/tls')),
    staleTime: 60_000,
  });
  const tlsInfo = tlsInfoQuery.data;

  const toggleShow = (key: string) => setShowValues(prev => ({ ...prev, [key]: !prev[key] }));

  const decodeValue = useCallback((b64: string): string => {
    try {
      return atob(b64);
    } catch {
      return b64;
    }
  }, []);

  const copyDecoded = useCallback((key: string) => {
    const raw = data[key];
    if (raw == null) return;
    const decoded = decodeValue(raw);
    navigator.clipboard.writeText(decoded).then(
      () => toast.success(`Copied value of "${key}"`),
      () => toast.error('Copy failed')
    );
  }, [data, decodeValue]);

  const decodedSize = useCallback((b64: string): number => Math.round((b64?.length ?? 0) * 0.75), []);

  return (
    <div className="space-y-6">
      {/* TLS certificate section */}
      {secretType === 'kubernetes.io/tls' && (
        <SectionCard icon={KeyRound} title="TLS certificate" tooltip={<p className="text-xs text-muted-foreground">Parsed certificate info (raw cert data is not shown)</p>}>
          {!baseUrl || !clusterId ? (
            <p className="text-muted-foreground text-sm">Connect to backend and select cluster to load certificate details.</p>
          ) : tlsInfoQuery.isLoading ? (
            <Skeleton className="h-24 w-full" />
          ) : tlsInfo?.hasValidCert && tlsInfo ? (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
              <DetailRow label="Issuer" value={tlsInfo.issuer ?? '—'} />
              <DetailRow label="Subject" value={tlsInfo.subject ?? '—'} />
              <DetailRow label="Valid From" value={tlsInfo.validFrom ?? '—'} />
              <DetailRow label="Valid To" value={tlsInfo.validTo ?? '—'} />
              <DetailRow
                label="Days Remaining"
                value={
                  <Badge className={cn('font-mono', daysRemainingColor(tlsInfo.daysRemaining ?? 0))}>
                    {(tlsInfo.daysRemaining ?? 0) < 0 ? `Expired ${-(tlsInfo.daysRemaining ?? 0)}d ago` : `${tlsInfo.daysRemaining} days`}
                  </Badge>
                }
              />
            </div>
          ) : (
            <p className="text-muted-foreground text-sm">{tlsInfo?.error ?? 'No valid certificate in tls.crt'}</p>
          )}
        </SectionCard>
      )}

      {/* Docker config section */}
      {secretType === 'kubernetes.io/dockerconfigjson' && data['.dockerconfigjson'] && (() => {
        try {
          const decoded = decodeValue(data['.dockerconfigjson']);
          const parsed = JSON.parse(decoded) as { auths?: Record<string, { username?: string; password?: string; auth?: string }> };
          const auths = parsed?.auths ?? {};
          const entries = Object.entries(auths);
          if (entries.length === 0) return null;
          return (
            <SectionCard icon={KeyRound} title="Docker registries" tooltip={<p className="text-xs text-muted-foreground">Registry URLs and usernames (passwords masked)</p>}>
              <div className="space-y-3">
                {entries.map(([registry, cred]) => (
                  <div key={registry} className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm">
                    <span className="font-mono text-primary">{registry}</span>
                    <span className="text-muted-foreground">username: {cred?.username ?? '—'}</span>
                    <span className="text-muted-foreground">password: ••••••••</span>
                  </div>
                ))}
              </div>
            </SectionCard>
          );
        } catch {
          return null;
        }
      })()}

      {/* Per-key data table */}
      <SectionCard icon={KeyRound} title="Keys and values" tooltip={<p className="text-xs text-muted-foreground">Reveal or copy decoded values</p>}>
        {Object.keys(data).length === 0 ? (
          <p className="text-muted-foreground text-sm">No keys</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="font-medium">Key</TableHead>
                <TableHead className="font-medium">Value</TableHead>
                <TableHead className="font-medium w-20">Size</TableHead>
                <TableHead className="w-24 text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {Object.entries(data).map(([key, value]) => (
                <TableRow key={key}>
                  <TableCell className="font-mono text-sm">{key}</TableCell>
                  <TableCell>
                    <pre className={cn("p-2 rounded bg-muted text-sm font-mono overflow-x-auto max-w-md", !showValues[key] && "select-none")}>
                      {showValues[key] ? decodeValue(value) : '••••••••••••'}
                    </pre>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">{decodedSize(value)} B</TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button variant="ghost" size="sm" className="press-effect" onClick={() => toggleShow(key)} aria-label={showValues[key] ? 'Hide value' : 'Reveal value'}>
                        {showValues[key] ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                      </Button>
                      <Button variant="ghost" size="sm" className="press-effect" onClick={() => copyDecoded(key)} aria-label="Copy value">
                        <Copy className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </SectionCard>
    </div>
  );
}

function UsedByTab({ namespace, name }: { namespace?: string; name?: string }) {
  const navigate = useNavigate();
  const clusterId = useActiveClusterId();
  const isBackendConfigured = useBackendConfigStore((s) => s.isBackendConfigured)();
  const storedUrl = useBackendConfigStore((s) => s.backendBaseUrl);
  const baseUrl = getEffectiveBackendBaseUrl(storedUrl);

  const consumersQuery = useQuery({
    queryKey: ['secret-consumers', clusterId, namespace, name],
    queryFn: () => getSecretConsumers(baseUrl!, clusterId!, namespace ?? '', name!),
    enabled: !!(isBackendConfigured && clusterId && namespace && name),
    staleTime: 30_000,
  });
  const consumers = consumersQuery.data;

  if (!namespace || !name) return <p className="text-muted-foreground text-sm">No resource selected.</p>;
  if (!isBackendConfigured || !clusterId) return <p className="text-muted-foreground text-sm">Connect to Kubilitics backend to see which Pods and workloads use this Secret.</p>;
  if (consumersQuery.isLoading) return <Skeleton className="h-32 w-full" />;

  if (consumers) {
    const sections = [
      { label: 'Pods', items: consumers.pods, path: (ns: string, n: string) => `/pods/${ns}/${n}` },
      { label: 'Deployments', items: consumers.deployments, path: (ns: string, n: string) => `/deployments/${ns}/${n}` },
      { label: 'StatefulSets', items: consumers.statefulSets, path: (ns: string, n: string) => `/statefulsets/${ns}/${n}` },
      { label: 'DaemonSets', items: consumers.daemonSets, path: (ns: string, n: string) => `/daemonsets/${ns}/${n}` },
      { label: 'Jobs', items: consumers.jobs, path: (ns: string, n: string) => `/jobs/${ns}/${n}` },
      { label: 'CronJobs', items: consumers.cronJobs, path: (ns: string, n: string) => `/cronjobs/${ns}/${n}` },
    ].filter((s) => (s.items?.length ?? 0) > 0);

    return (
      <div className="space-y-4">
        {sections.map((section) => (
          <Card key={section.label}>
            <CardHeader><CardTitle className="text-base">{section.label}</CardTitle></CardHeader>
            <CardContent>
              <ul className="space-y-1">
                {(section.items ?? []).map((ref) => (
                  <li key={`${ref.namespace}/${ref.name}`}>
                    <button type="button" className="text-primary hover:underline font-mono text-sm" onClick={() => navigate(section.path(ref.namespace, ref.name))}>
                      {ref.namespace}/{ref.name}
                    </button>
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>
        ))}
        {sections.length === 0 && (
          <p className="text-muted-foreground text-sm">No consumers found.</p>
        )}
      </div>
    );
  }

  return <p className="text-muted-foreground text-sm">Could not load consumers.</p>;
}

export default function SecretDetail() {
  const { namespace, name } = useParams();
  const clusterId = useActiveClusterId();
  const isBackendConfigured = useBackendConfigStore((s) => s.isBackendConfigured)();
  const storedUrl = useBackendConfigStore((s) => s.backendBaseUrl);
  const baseUrl = getEffectiveBackendBaseUrl(storedUrl);

  // Pre-fetch consumers for status card count
  const consumersQuery = useQuery({
    queryKey: ['secret-consumers', clusterId, namespace, name],
    queryFn: () => getSecretConsumers(baseUrl!, clusterId!, namespace ?? '', name!),
    enabled: !!(isBackendConfigured && clusterId && namespace && name),
    staleTime: 30_000,
  });
  const consumers = consumersQuery.data;

  const customTabs: CustomTab[] = [
    { id: 'overview', label: 'Overview', icon: Info, render: (ctx) => <OverviewTab {...ctx} /> },
    { id: 'data', label: 'Data', icon: KeyRound, render: (ctx) => <DataTab {...ctx} /> },
    { id: 'used-by', label: 'Used By', icon: KeyRound, render: () => <UsedByTab namespace={namespace} name={name} /> },
  ];

  return (
    <GenericResourceDetail<SecretResource>
      resourceType="secrets"
      kind="Secret"
      pluralLabel="Secrets"
      listPath="/secrets"
      resourceIcon={KeyRound}
      loadingCardCount={3}
      customTabs={customTabs}
      deriveStatus={() => 'Healthy'}
      buildStatusCards={(ctx) => {
        const s = ctx.resource;
        const data = s.data || {};
        const secretType = s.type || 'Opaque';
        const totalSizeBytes = Object.values(data).reduce((acc, v) => acc + (typeof v === 'string' ? v.length : 0), 0);
        const totalSizeHuman = totalSizeBytes >= 1024 * 1024 ? `${(totalSizeBytes / (1024 * 1024)).toFixed(1)} MiB` : totalSizeBytes >= 1024 ? `${(totalSizeBytes / 1024).toFixed(1)} KiB` : `${totalSizeBytes} B`;
        const usedByCount = consumers ? (consumers.pods?.length ?? 0) + (consumers.deployments?.length ?? 0) + (consumers.statefulSets?.length ?? 0) + (consumers.daemonSets?.length ?? 0) + (consumers.jobs?.length ?? 0) + (consumers.cronJobs?.length ?? 0) : 0;

        return [
          { label: 'Type', value: secretType, icon: KeyRound, iconColor: 'primary' as const },
          { label: 'Keys', value: Object.keys(data).length, icon: KeyRound, iconColor: 'info' as const },
          { label: 'Size', value: totalSizeHuman, icon: KeyRound, iconColor: 'muted' as const },
          { label: 'Used By', value: usedByCount, icon: KeyRound, iconColor: 'primary' as const },
          { label: 'Age', value: ctx.age, icon: Clock, iconColor: 'muted' as const },
        ];
      }}
      extraActionItems={() => [
        { icon: Edit, label: 'Edit Secret', description: 'Modify secret data', className: 'press-effect' },
        { icon: Copy, label: 'Duplicate', description: 'Create a copy of this Secret', className: 'press-effect' },
      ]}
    />
  );
}
