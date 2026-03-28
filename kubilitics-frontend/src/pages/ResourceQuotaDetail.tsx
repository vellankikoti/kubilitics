import { useMemo } from 'react';
import { Gauge, Clock, Box, AlertTriangle, Database, Info } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
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
import { formatUsagePercent } from '@/lib/k8s-utils';

interface ResourceQuotaResource extends KubernetesResource {
  spec?: { hard?: Record<string, string>; scopeSelector?: unknown };
  status?: { hard?: Record<string, string>; used?: Record<string, string> };
}

function getUsagePercent(used: string, hard: string): number | null {
  return formatUsagePercent(used, hard);
}

function usageBarIndicatorClass(pct: number | null): string {
  if (pct == null) return 'bg-muted-foreground/40';
  if (pct >= 100) return 'bg-destructive';
  if (pct >= 80) return 'bg-amber-500';
  return 'bg-emerald-600';
}

// ---------------------------------------------------------------------------
// Custom tab components
// ---------------------------------------------------------------------------

function OverviewTab({ resource, age }: ResourceContext<ResourceQuotaResource>) {
  const hard = resource?.status?.hard || resource?.spec?.hard || {};
  const used = resource?.status?.used || {};
  const labels = resource?.metadata?.labels ?? {};
  const annotations = resource?.metadata?.annotations ?? {};
  const quotaName = resource?.metadata?.name ?? '';
  const quotaNamespace = resource?.metadata?.namespace ?? '';
  const hasScopeSelector = !!(resource?.spec?.scopeSelector && Object.keys((resource.spec.scopeSelector as Record<string, unknown>) || {}).length > 0);
  const resourcesTracked = Object.keys(hard).length;

  const overallPct = useMemo(() => {
    const h = resource?.status?.hard || resource?.spec?.hard || {};
    const u = resource?.status?.used || {};
    let maxPct: number | null = null;
    for (const key of Object.keys(h)) {
      const pct = getUsagePercent(u[key] || '0', h[key] || '');
      if (pct != null && (maxPct == null || pct > maxPct)) maxPct = pct;
    }
    return maxPct;
  }, [resource?.status?.hard, resource?.spec?.hard, resource?.status?.used]);

  return (
    <div className="grid grid-cols-1 gap-6">
      <SectionCard icon={Info} title="Quota Information" tooltip="Resource quota metadata and scope">
        <div className="grid grid-cols-2 gap-x-8 gap-y-3">
          <DetailRow label="Name" value={quotaName} />
          <DetailRow label="Namespace" value={quotaNamespace} />
          <DetailRow label="Resources Tracked" value={String(resourcesTracked)} />
          <DetailRow label="Scope Selector" value={hasScopeSelector ? 'Yes' : 'No'} />
          <DetailRow label="Overall Usage" value={overallPct != null ? <Badge variant={overallPct >= 100 ? 'destructive' : overallPct >= 80 ? 'secondary' : 'default'}>{overallPct}%</Badge> : '—'} />
          <DetailRow label="Age" value={age} />
        </div>
      </SectionCard>
      <SectionCard icon={Database} title="Resource Usage" className="lg:col-span-2">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {Object.entries(hard).map(([key, hardVal]) => {
              const usedVal = used[key] ?? '0';
              const percent = getUsagePercent(usedVal, hardVal);
              return (
                <div key={key} className="p-4 rounded-lg bg-muted/50 space-y-2">
                  <div className="flex justify-between items-center">
                    <span className="font-medium text-sm font-mono">{key}</span>
                    {percent != null ? (
                      <Badge variant={percent >= 100 ? 'destructive' : percent >= 80 ? 'secondary' : 'default'}>
                        {percent}%
                      </Badge>
                    ) : (
                      <span className="text-muted-foreground text-sm">used / hard</span>
                    )}
                  </div>
                  {percent != null && <Progress value={Math.min(percent, 100)} className="h-2" />}
                  <p className="text-xs text-muted-foreground">{usedVal} / {hardVal}</p>
                </div>
              );
            })}
          </div>
          {Object.keys(hard).length === 0 && (
            <p className="text-muted-foreground text-sm">No hard limits defined.</p>
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

function UsageTab({ resource }: ResourceContext<ResourceQuotaResource>) {
  const hard = resource?.status?.hard || resource?.spec?.hard || {};
  const used = resource?.status?.used || {};

  const usageRows = useMemo(() => {
    const h = resource?.status?.hard || resource?.spec?.hard || {};
    const u = resource?.status?.used || {};
    return Object.keys(h)
      .sort()
      .map((resource) => {
        const hardVal = h[resource] ?? '';
        const usedVal = u[resource] ?? '0';
        const percent = getUsagePercent(usedVal, hardVal);
        return { resource, used: usedVal, hard: hardVal, percent };
      });
  }, [resource?.status?.hard, resource?.spec?.hard, resource?.status?.used]);

  const nearingLimitResources = useMemo(() => usageRows.filter((r) => r.percent != null && r.percent > 80), [usageRows]);

  return (
    <div className="space-y-6">
      {nearingLimitResources.length > 0 && (
        <Alert variant="destructive" className="border-amber-500/50 bg-amber-500/10">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>Nearing limit</AlertTitle>
          <AlertDescription>
            The following resources are above 80% of their hard limit: <span className="font-mono font-medium">{nearingLimitResources.map((r) => r.resource).join(', ')}</span>. Consider increasing quotas or reducing usage.
          </AlertDescription>
        </Alert>
      )}
      <SectionCard icon={Gauge} title="Per-Resource Usage" tooltip="Used vs hard limit for each quota resource. Bars are green (<80%), amber (80-99%), or red (>=100%).">
          {usageRows.length === 0 ? (
            <p className="text-muted-foreground text-sm">No hard limits defined.</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className="bg-muted/40">
                  <TableHead className="font-medium">Resource</TableHead>
                  <TableHead className="font-medium">Used</TableHead>
                  <TableHead className="font-medium">Hard limit</TableHead>
                  <TableHead className="font-medium w-24">Usage %</TableHead>
                  <TableHead className="font-medium min-w-[180px]">Bar</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {usageRows.map((row) => (
                  <TableRow key={row.resource}>
                    <TableCell className="font-mono text-sm">{row.resource}</TableCell>
                    <TableCell className="font-mono text-sm">{row.used}</TableCell>
                    <TableCell className="font-mono text-sm">{row.hard}</TableCell>
                    <TableCell>
                      {row.percent != null ? (
                        <span className={row.percent >= 100 ? 'text-destructive font-medium' : row.percent >= 80 ? 'text-amber-600 font-medium' : 'text-emerald-600 font-medium'}>
                          {row.percent}%
                        </span>
                      ) : (
                        <span className="text-muted-foreground">–</span>
                      )}
                    </TableCell>
                    <TableCell>
                      {row.percent != null ? (
                        <div className="flex items-center gap-2">
                          <Progress value={Math.min(row.percent, 100)} className="h-2.5 flex-1 max-w-[160px]" indicatorClassName={usageBarIndicatorClass(row.percent)} />
                        </div>
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
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export default function ResourceQuotaDetail() {
  const customTabs: CustomTab[] = [
    { id: 'overview', label: 'Overview', render: (ctx) => <OverviewTab {...ctx} /> },
    { id: 'usage', label: 'Usage', icon: Gauge, render: (ctx) => <UsageTab {...ctx} /> },
  ];

  return (
    <GenericResourceDetail<ResourceQuotaResource>
      resourceType="resourcequotas"
      kind="ResourceQuota"
      pluralLabel="Resource Quotas"
      listPath="/resourcequotas"
      resourceIcon={Gauge}
      loadingCardCount={4}
      customTabs={customTabs}
      deriveStatus={(resource) => {
        const hard = resource?.status?.hard || resource?.spec?.hard || {};
        const used = resource?.status?.used || {};
        let maxPct: number | null = null;
        for (const key of Object.keys(hard)) {
          const pct = getUsagePercent(used[key] || '0', hard[key] || '');
          if (pct != null && (maxPct == null || pct > maxPct)) maxPct = pct;
        }
        return maxPct != null && maxPct >= 100 ? 'Failed' : 'Healthy';
      }}
      buildStatusCards={(ctx) => {
        const resource = ctx.resource;
        const hard = resource?.status?.hard || resource?.spec?.hard || {};
        const used = resource?.status?.used || {};
        const quotaNamespace = resource?.metadata?.namespace ?? '';
        const hasScopeSelector = !!(resource?.spec?.scopeSelector && Object.keys((resource.spec.scopeSelector as Record<string, unknown>) || {}).length > 0);
        const resourcesTracked = Object.keys(hard).length;

        let maxPct: number | null = null;
        for (const key of Object.keys(hard)) {
          const pct = getUsagePercent(used[key] || '0', hard[key] || '');
          if (pct != null && (maxPct == null || pct > maxPct)) maxPct = pct;
        }

        return [
          { label: 'Overall Usage', value: maxPct != null ? `${maxPct}%` : '–', icon: Gauge, iconColor: 'primary' as const },
          { label: 'Resources Tracked', value: resourcesTracked, icon: Box, iconColor: 'muted' as const },
          { label: 'Namespace', value: quotaNamespace, icon: Clock, iconColor: 'info' as const },
          { label: 'Scopes', value: hasScopeSelector ? 'Yes' : 'No', icon: Gauge, iconColor: 'muted' as const },
        ];
      }}
    />
  );
}
