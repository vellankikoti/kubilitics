import { useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { ShieldCheck, Users, Globe, Layers, List, Link2 } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
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

interface ClusterRoleRule {
  apiGroups?: string[];
  resources?: string[];
  resourceNames?: string[];
  verbs?: string[];
  nonResourceURLs?: string[];
}

interface AggregationRule {
  clusterRoleSelectors?: Array<{ matchLabels?: Record<string, string>; matchExpressions?: unknown[] }>;
}

interface ClusterRoleResource extends KubernetesResource {
  rules?: ClusterRoleRule[];
  aggregationRule?: AggregationRule;
}

const VERBS_ORDER = ['get', 'list', 'watch', 'create', 'update', 'patch', 'delete', 'deletecollection'];

function buildPermissionMatrix(rules: ClusterRoleRule[]): Map<string, Set<string>> {
  const matrix = new Map<string, Set<string>>();
  for (const rule of rules || []) {
    if (rule.nonResourceURLs?.length) {
      for (const url of rule.nonResourceURLs) {
        const key = `(non-resource) ${url}`;
        if (!matrix.has(key)) matrix.set(key, new Set());
        for (const v of rule.verbs ?? []) matrix.get(key)!.add(v);
      }
    }
    const apiGroup = (rule.apiGroups ?? ['']).join(',') || 'core';
    for (const res of rule.resources ?? []) {
      const key = apiGroup ? `${res} (${apiGroup})` : res;
      if (!matrix.has(key)) matrix.set(key, new Set());
      for (const v of rule.verbs ?? []) matrix.get(key)!.add(v);
    }
  }
  return matrix;
}

function OverviewTab({ resource }: ResourceContext<ClusterRoleResource>) {
  const rules = resource?.rules ?? [];
  const aggregationRule = resource?.aggregationRule;
  const labels = resource?.metadata?.labels ?? {};
  const annotations = resource?.metadata?.annotations ?? {};

  return (
    <div className="grid grid-cols-1 gap-6">
      {aggregationRule?.clusterRoleSelectors?.length ? (
        <SectionCard icon={Layers} title="Aggregation Rule">
            <div className="space-y-2">
              {aggregationRule.clusterRoleSelectors.map((sel, i) => (
                <div key={i} className="p-3 rounded-lg bg-muted/50 text-sm">
                  {sel.matchLabels && Object.keys(sel.matchLabels).length > 0 ? (
                    <div className="flex flex-wrap gap-2">
                      {Object.entries(sel.matchLabels).map(([k, v]) => (
                        <Badge key={k} variant="outline">{k}={v}</Badge>
                      ))}
                    </div>
                  ) : (
                    <span className="text-muted-foreground">Label selector</span>
                  )}
                </div>
              ))}
            </div>
        </SectionCard>
      ) : null}
      <SectionCard icon={ShieldCheck} title="Rules">
        <div className="space-y-4">
          {rules.length === 0 ? (
            <p className="text-muted-foreground text-sm">No rules (aggregated role may inherit from others).</p>
          ) : (
            rules.map((rule, i) => (
              <div key={i} className="p-4 rounded-lg bg-muted/50">
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                  <DetailRow
                    label="API Groups"
                    value={
                      <div className="flex flex-wrap gap-1">
                        {(rule.apiGroups ?? ['']).map((g, j) => (
                          <Badge key={j} variant="secondary" className="font-mono">{g || 'core'}</Badge>
                        ))}
                      </div>
                    }
                  />
                  <DetailRow
                    label="Resources / Non-Resource URLs"
                    value={
                      <div className="flex flex-wrap gap-1">
                        {(rule.resources ?? []).map((r, j) => (
                          <Badge key={j} variant="outline" className="font-mono text-xs">{r}</Badge>
                        ))}
                        {(rule.nonResourceURLs ?? []).map((url, j) => (
                          <Badge key={`n-${j}`} variant="secondary" className="font-mono text-xs">{url}</Badge>
                        ))}
                      </div>
                    }
                  />
                  <DetailRow
                    label="Verbs"
                    value={
                      <div className="flex flex-wrap gap-1">
                        {(rule.verbs ?? []).map((v, j) => (
                          <Badge key={j} variant="default" className="font-mono text-xs">{v}</Badge>
                        ))}
                      </div>
                    }
                  />
                </div>
              </div>
            ))
          )}
        </div>
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

function PermissionMatrixTab({ resource }: ResourceContext<ClusterRoleResource>) {
  const permissionMatrix = useMemo(() => buildPermissionMatrix(resource?.rules ?? []), [resource?.rules]);

  return (
    <SectionCard icon={List} title="Resources x Verbs">
        {permissionMatrix.size === 0 ? (
          <p className="text-muted-foreground text-sm">No rules to display.</p>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-48">Resource</TableHead>
                  {VERBS_ORDER.map((v) => (
                    <TableHead key={v} className="text-center w-20">{v}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {Array.from(permissionMatrix.entries()).map(([res, verbs]) => (
                  <TableRow key={res}>
                    <TableCell className="font-mono text-sm">{res}</TableCell>
                    {VERBS_ORDER.map((v) => (
                      <TableCell key={v} className="text-center">
                        {verbs.has(v) ? <span className="inline-block w-4 h-4 rounded bg-green-500/80" title={v} /> : <span className="inline-block w-4 h-4 rounded bg-muted" />}
                      </TableCell>
                    ))}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
    </SectionCard>
  );
}

function BindingsTab() {
  const navigate = useNavigate();
  return (
    <SectionCard icon={Link2} title="ClusterRoleBindings / RoleBindings">
        <p className="text-muted-foreground text-sm">Bindings that reference this ClusterRole. View Cluster Role Bindings to see cluster-wide bindings.</p>
        <Button variant="outline" size="sm" className="mt-2" onClick={() => navigate('/clusterrolebindings')}>View Cluster Role Bindings</Button>
    </SectionCard>
  );
}

function AggregationTab({ resource }: ResourceContext<ClusterRoleResource>) {
  const aggregationRule = resource?.aggregationRule;
  return (
    <SectionCard icon={Layers} title="Aggregation">
        {aggregationRule?.clusterRoleSelectors?.length ? (
          <div className="space-y-2">
            {aggregationRule.clusterRoleSelectors.map((sel, i) => (
              <div key={i} className="p-3 rounded-lg bg-muted/50 text-sm">
                {sel.matchLabels && Object.keys(sel.matchLabels).length > 0
                  ? Object.entries(sel.matchLabels).map(([k, v]) => <Badge key={k} variant="outline" className="mr-1">{k}={v}</Badge>)
                  : <span className="text-muted-foreground">Label selector</span>}
              </div>
            ))}
          </div>
        ) : (
          <p className="text-muted-foreground text-sm">This ClusterRole is not aggregated.</p>
        )}
    </SectionCard>
  );
}

function EffectiveSubjectsTab() {
  return (
    <SectionCard icon={Users} title="Subjects">
        <p className="text-muted-foreground text-sm">Subjects are derived from ClusterRoleBindings (and namespaced RoleBindings) that reference this ClusterRole.</p>
    </SectionCard>
  );
}

export default function ClusterRoleDetail() {
  const customTabs: CustomTab[] = [
    { id: 'overview', label: 'Overview', render: (ctx) => <OverviewTab {...ctx} /> },
    { id: 'permission-matrix', label: 'Permission Matrix', render: (ctx) => <PermissionMatrixTab {...ctx} /> },
    { id: 'bindings', label: 'Bindings', render: () => <BindingsTab /> },
    { id: 'aggregation', label: 'Aggregation', render: (ctx) => <AggregationTab {...ctx} /> },
    { id: 'effective-subjects', label: 'Effective Subjects', render: () => <EffectiveSubjectsTab /> },
  ];

  return (
    <GenericResourceDetail<ClusterRoleResource>
      resourceType="clusterroles"
      kind="ClusterRole"
      pluralLabel="Cluster Roles"
      listPath="/clusterroles"
      resourceIcon={ShieldCheck}
      customTabs={customTabs}
      buildStatusCards={(ctx) => {
        const rules = ctx.resource?.rules ?? [];
        const aggregationRule = ctx.resource?.aggregationRule;

        return [
          { label: 'Rules Count', value: rules.length, icon: ShieldCheck, iconColor: 'primary' as const },
          { label: 'Bindings', value: '–', icon: ShieldCheck, iconColor: 'muted' as const },
          { label: 'Aggregation', value: aggregationRule ? 'Yes' : 'No', icon: ShieldCheck, iconColor: 'muted' as const },
          { label: 'Scope', value: 'Cluster-wide', icon: Globe, iconColor: 'info' as const },
        ];
      }}
    />
  );
}
