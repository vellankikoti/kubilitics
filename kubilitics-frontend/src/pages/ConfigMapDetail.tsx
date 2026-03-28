import { useState, useMemo } from 'react';
import { FileJson, Clock, Download, Copy, Info, Maximize2 } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
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
import { useBackendConfigStore, getEffectiveBackendBaseUrl } from '@/stores/backendConfigStore';
import { useActiveClusterId } from '@/hooks/useActiveClusterId';
import { useQuery } from '@tanstack/react-query';
import { getConfigMapConsumers } from '@/services/backendApiClient';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { toast } from '@/components/ui/sonner';
import { useNavigate, useParams } from 'react-router-dom';

interface ConfigMapResource extends KubernetesResource {
  data?: Record<string, string>;
  binaryData?: Record<string, string>;
  immutable?: boolean;
}

const PREVIEW_LEN = 200;
const HEX_PREVIEW_BYTES = 32;

function detectValueFormat(value: string): 'json' | 'yaml' | 'properties' | 'text' {
  const t = value.trim();
  if (!t) return 'text';
  if ((t.startsWith('{') && t.endsWith('}')) || (t.startsWith('[') && t.endsWith(']'))) {
    try { JSON.parse(value); return 'json'; } catch { /* fallback */ }
  }
  if (t.includes('\n') && /^[\w-]+:/.test(t)) return 'yaml';
  if (/^[\w.-]+=.*/m.test(t)) return 'properties';
  return 'text';
}

function getFileExtension(key: string, format: string): string {
  if (format === 'json') return key.endsWith('.json') ? '.json' : '.json';
  if (format === 'yaml') return key.endsWith('.yaml') || key.endsWith('.yml') ? '' : '.yaml';
  return '.txt';
}

// ---------------------------------------------------------------------------
// "Used By" section — needs backend query so it lives outside the tabs
// ---------------------------------------------------------------------------
function UsedByContent({ namespace, name }: { namespace?: string; name?: string }) {
  const navigate = useNavigate();
  const clusterId = useActiveClusterId();
  const isBackendConfigured = useBackendConfigStore((s) => s.isBackendConfigured());
  const backendBaseUrl = useBackendConfigStore((s) => s.backendBaseUrl);
  const baseUrl = getEffectiveBackendBaseUrl(backendBaseUrl);

  const consumersQuery = useQuery({
    queryKey: ['configmap-consumers', clusterId, namespace, name],
    queryFn: () => getConfigMapConsumers(baseUrl!, clusterId!, namespace ?? '', name!),
    enabled: !!(isBackendConfigured && clusterId && namespace && name),
    staleTime: 30_000,
  });
  const consumers = consumersQuery.data;

  const usedByRows = useMemo(() => {
    if (!consumers) return [];
    const rows: { type: string; namespace: string; name: string; path: string }[] = [];
    const add = (type: string, pathPrefix: string, items: { namespace: string; name: string }[] | undefined) => {
      (items ?? []).forEach((ref) => rows.push({ type, namespace: ref.namespace, name: ref.name, path: `${pathPrefix}/${ref.namespace}/${ref.name}` }));
    };
    add('Pod', '/pods', consumers.pods);
    add('Deployment', '/deployments', consumers.deployments);
    add('StatefulSet', '/statefulsets', consumers.statefulSets);
    add('DaemonSet', '/daemonsets', consumers.daemonSets);
    add('Job', '/jobs', consumers.jobs);
    add('CronJob', '/cronjobs', consumers.cronJobs);
    return rows;
  }, [consumers]);

  if (!namespace || !name) return <p className="text-muted-foreground text-sm">No resource selected.</p>;
  if (!isBackendConfigured || !clusterId) return <p className="text-muted-foreground text-sm">Connect to Kubilitics backend to see which Pods and workloads use this ConfigMap.</p>;
  if (consumersQuery.isLoading) return <Skeleton className="h-32 w-full" />;
  if (consumers == null) return <p className="text-muted-foreground text-sm">Could not load consumers.</p>;

  return (
    <div className="space-y-4">
      {usedByRows.length > 0 ? (
        <>
          <p className="text-muted-foreground text-sm">Resources that reference this ConfigMap (volume, env, or envFrom).</p>
          <div className="rounded-md border overflow-x-auto">
            <table className="w-full text-sm">
              <thead><tr className="border-b bg-muted/50"><th className="text-left p-3 font-medium">Resource Type</th><th className="text-left p-3 font-medium">Resource Name</th><th className="text-left p-3 font-medium">How Used</th><th className="text-left p-3 font-medium">Mount Path / Prefix</th></tr></thead>
              <tbody>
                {usedByRows.map((row) => (
                  <tr key={`${row.type}-${row.namespace}/${row.name}`} className="border-b">
                    <td className="p-3">{row.type}</td>
                    <td className="p-3">
                      <button type="button" className="text-primary hover:underline font-mono text-sm" onClick={() => navigate(row.path)}>{row.name}</button>
                      <span className="text-muted-foreground text-xs ml-1">({row.namespace})</span>
                    </td>
                    <td className="p-3 text-muted-foreground">—</td>
                    <td className="p-3 text-muted-foreground">—</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {(consumers.pods?.length ?? 0) > 0 && (
            <p className="text-sm text-amber-600 dark:text-amber-400 bg-amber-500/10 border border-amber-500/20 rounded-lg p-3">
              Impact: Changing this ConfigMap will require restart of <strong>{consumers.pods?.length ?? 0} pod(s)</strong> to pick up changes (when mounted as volume or used in env).
            </p>
          )}
        </>
      ) : (
        <p className="text-muted-foreground text-sm">No consumers found.</p>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Custom tabs
// ---------------------------------------------------------------------------

function OverviewTab({ resource: cm, age }: ResourceContext<ConfigMapResource>) {
  const data = cm.data || {};
  const dataKeysCount = Object.keys(data).length + (cm.binaryData ? Object.keys(cm.binaryData).length : 0);
  const labels = cm.metadata?.labels || {};

  return (
    <div className="space-y-6">
      <SectionCard
        icon={Info}
        title="ConfigMap information"
        tooltip={<p className="text-xs text-muted-foreground">Identity and metadata for this ConfigMap</p>}
      >
        <div className="grid grid-cols-2 gap-x-8 gap-y-3">
          <DetailRow label="Name" value={cm.metadata?.name || ''} />
          <DetailRow label="Namespace" value={cm.metadata?.namespace || ''} />
          <DetailRow label="Age" value={age} />
          <DetailRow label="Data Keys" value={String(dataKeysCount)} />
          <DetailRow label="Immutable" value={<Badge variant="outline">{cm.immutable ? 'Yes' : 'No'}</Badge>} />
        </div>
      </SectionCard>
      <SectionCard
        icon={FileJson}
        title="Data keys"
        tooltip={<p className="text-xs text-muted-foreground">Keys defined in this ConfigMap (see Data tab for values)</p>}
      >
        <div className="flex flex-wrap gap-2">
          {Object.keys(data).map((key) => (
            <Badge key={key} variant="secondary" className="font-mono">{key}</Badge>
          ))}
          {cm.binaryData && Object.keys(cm.binaryData).map((key) => (
            <Badge key={`binary-${key}`} variant="secondary" className="font-mono">{key} (binary)</Badge>
          ))}
          {dataKeysCount === 0 && <p className="text-muted-foreground text-sm">No keys</p>}
        </div>
      </SectionCard>
      <div className="lg:col-span-2">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <LabelList labels={labels} />
        </div>
      </div>
      <div className="lg:col-span-2">
        <AnnotationList annotations={cm?.metadata?.annotations || {}} />
      </div>
    </div>
  );
}

function DataTab({ resource: cm }: ResourceContext<ConfigMapResource>) {
  const [expandKey, setExpandKey] = useState<string | null>(null);
  const [expandValue, setExpandValue] = useState<string>('');
  const data = cm.data || {};

  return (
    <div className="space-y-6">
      {Object.entries(data).length > 0 && (
        <SectionCard title="Data keys" icon={FileJson} tooltip={<p className="text-xs text-muted-foreground">Per-key value preview, copy, and download</p>}>
          <div className="space-y-4">
            {Object.entries(data).map(([key, value]) => {
              const size = (value ?? '').length;
              const preview = (value ?? '').length <= PREVIEW_LEN ? (value ?? '') : (value ?? '').slice(0, PREVIEW_LEN) + '\u2026';
              const format = detectValueFormat(value ?? '');
              const ext = getFileExtension(key, format);
              return (
                <Card key={key}>
                  <CardHeader className="flex flex-row items-center justify-between py-3 gap-2">
                    <CardTitle className="text-sm font-mono truncate">{key}</CardTitle>
                    <span className="text-xs text-muted-foreground shrink-0">{size} B</span>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <pre className="p-3 rounded-lg bg-muted/80 text-sm font-mono overflow-auto max-h-40 whitespace-pre-wrap break-words">{preview}</pre>
                    <div className="flex flex-wrap gap-2">
                      <Button variant="outline" size="sm" className="gap-1.5 press-effect" onClick={() => { setExpandKey(key); setExpandValue(value ?? ''); }} aria-label={`Expand ${key}`}>
                        <Maximize2 className="h-3.5 w-3.5" /> Expand
                      </Button>
                      <Button variant="outline" size="sm" className="gap-1.5 press-effect" onClick={() => { navigator.clipboard.writeText(value ?? ''); toast.success('Value copied'); }} aria-label={`Copy value of ${key}`}>
                        <Copy className="h-3.5 w-3.5" /> Copy Value
                      </Button>
                      <Button variant="outline" size="sm" className="gap-1.5 press-effect" onClick={() => { const blob = new Blob([value ?? ''], { type: 'text/plain' }); const url = URL.createObjectURL(blob); const a = document.createElement('a'); a.href = url; a.download = `${key}${ext}`; a.click(); setTimeout(() => URL.revokeObjectURL(url), 30_000); toast.success('Downloaded'); }} aria-label={`Download ${key} as file`}>
                        <Download className="h-3.5 w-3.5" /> Download as File
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        </SectionCard>
      )}
      {cm.binaryData && Object.keys(cm.binaryData).length > 0 && (
        <SectionCard title="Binary data" icon={FileJson} tooltip={<p className="text-xs text-muted-foreground">Base64-encoded keys with hex preview</p>}>
          <div className="space-y-3">
            {Object.entries(cm.binaryData).map(([key, b64]) => {
              let decodedLength = 0;
              let hexPreview = '\u2014';
              try {
                const bin = atob(b64 ?? '');
                decodedLength = bin.length;
                const bytes = new Uint8Array(bin.length);
                for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
                hexPreview = Array.from(bytes.slice(0, HEX_PREVIEW_BYTES)).map((b) => b.toString(16).padStart(2, '0')).join(' ') + (bytes.length > HEX_PREVIEW_BYTES ? '\u2026' : '');
              } catch { /* ignore */ }
              return (
                <div key={key} className="flex flex-wrap items-center justify-between gap-2 p-3 rounded-lg border bg-muted/30">
                  <span className="font-mono text-sm">{key}</span>
                  <span className="text-xs text-muted-foreground">{decodedLength} B</span>
                  <code className="text-xs font-mono text-muted-foreground break-all w-full">{hexPreview}</code>
                  <Button variant="outline" size="sm" className="gap-1.5 press-effect" onClick={() => { try { const bin = atob(b64 ?? ''); const bytes = new Uint8Array(bin.length); for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i); const blob = new Blob([bytes]); const url = URL.createObjectURL(blob); const a = document.createElement('a'); a.href = url; a.download = key; a.click(); setTimeout(() => URL.revokeObjectURL(url), 30_000); toast.success('Downloaded'); } catch { toast.error('Failed to decode'); } }} aria-label={`Download binary ${key}`}>
                    <Download className="h-3.5 w-3.5" /> Download
                  </Button>
                </div>
              );
            })}
          </div>
        </SectionCard>
      )}
      {Object.keys(data).length === 0 && !(cm.binaryData && Object.keys(cm.binaryData).length > 0) && (
        <p className="text-muted-foreground text-sm">No data keys.</p>
      )}
      <Dialog open={!!expandKey} onOpenChange={(open) => { if (!open) setExpandKey(null); }}>
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-hidden flex flex-col">
          <DialogHeader>
            <DialogTitle className="font-mono text-sm">{expandKey ?? ''}</DialogTitle>
          </DialogHeader>
          <pre className="p-4 rounded-lg bg-muted text-sm font-mono overflow-auto flex-1 min-h-0 whitespace-pre-wrap break-words">{expandValue}</pre>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export default function ConfigMapDetail() {
  const { namespace, name } = useParams();

  const customTabs: CustomTab[] = [
    { id: 'overview', label: 'Overview', icon: Info, render: (ctx) => <OverviewTab {...ctx} /> },
    { id: 'data', label: 'Data', icon: FileJson, render: (ctx) => <DataTab {...ctx} /> },
    { id: 'used-by', label: 'Used By', icon: FileJson, render: () => <UsedByContent namespace={namespace} name={name} /> },
  ];

  return (
    <GenericResourceDetail<ConfigMapResource>
      resourceType="configmaps"
      kind="ConfigMap"
      pluralLabel="ConfigMaps"
      listPath="/configmaps"
      resourceIcon={FileJson}
      loadingCardCount={2}
      customTabs={customTabs}
      buildStatusCards={(ctx) => {
        const cm = ctx.resource;
        const data = cm.data || {};
        const dataKeysCount = Object.keys(data).length + (cm.binaryData ? Object.keys(cm.binaryData).length : 0);
        let totalSizeBytes = 0;
        if (cm.data) for (const v of Object.values(cm.data)) totalSizeBytes += (v ?? '').length;
        if (cm.binaryData) for (const v of Object.values(cm.binaryData)) totalSizeBytes += (typeof v === 'string' ? v.length : 0);
        const totalSizeHuman = totalSizeBytes >= 1024 * 1024
          ? `${(totalSizeBytes / (1024 * 1024)).toFixed(1)} MiB`
          : totalSizeBytes >= 1024
            ? `${(totalSizeBytes / 1024).toFixed(1)} KiB`
            : `${totalSizeBytes} B`;

        return [
          { label: 'Keys', value: dataKeysCount, icon: FileJson, iconColor: 'primary' as const },
          { label: 'Total Size', value: totalSizeHuman, icon: FileJson, iconColor: 'muted' as const },
          { label: 'Immutable', value: cm.immutable ? 'Yes' : 'No', icon: FileJson, iconColor: 'muted' as const },
        ];
      }}
    />
  );
}
