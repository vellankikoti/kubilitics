import { useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Shield, Users, List, Link2 } from 'lucide-react';
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

interface RoleRule {
  apiGroups?: string[];
  resources?: string[];
  resourceNames?: string[];
  verbs?: string[];
}

interface RoleResource extends KubernetesResource {
  rules?: RoleRule[];
}

const VERBS_ORDER = ['get', 'list', 'watch', 'create', 'update', 'patch', 'delete', 'deletecollection'];

function buildPermissionMatrix(rules: RoleRule[]): Map<string, Set<string>> {
  const matrix = new Map<string, Set<string>>();
  for (const rule of rules || []) {
    const apiGroup = (rule.apiGroups || ['']).join(',') || 'core';
    for (const res of rule.resources || []) {
      const key = apiGroup ? `${res} (${apiGroup})` : res;
      if (!matrix.has(key)) matrix.set(key, new Set());
      for (const v of rule.verbs || []) matrix.get(key)!.add(v);
    }
  }
  return matrix;
}

function OverviewTab({ resource, namespace }: ResourceContext<RoleResource>) {
  const rules = resource?.rules ?? [];
  const labels = resource?.metadata?.labels ?? {};
  const annotations = resource?.metadata?.annotations ?? {};

  return (
    <div className="grid grid-cols-1 gap-6">
      <SectionCard icon={Shield} title="Rules">
        <div className="space-y-4">
          {rules.length === 0 ? (
            <p className="text-muted-foreground text-sm">No rules</p>
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
                    label="Resources"
                    value={
                      <div className="flex flex-wrap gap-1">
                        {(rule.resources ?? []).map((r, j) => (
                          <Badge key={j} variant="outline" className="font-mono">{r}</Badge>
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

function PermissionMatrixTab({ resource }: ResourceContext<RoleResource>) {
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

function BindingsTab({ namespace }: ResourceContext<RoleResource>) {
  const navigate = useNavigate();
  return (
    <SectionCard icon={Link2} title="RoleBindings">
        <p className="text-muted-foreground text-sm">RoleBindings that reference this Role can be listed by viewing RoleBindings in this namespace and filtering by role.</p>
        <Button variant="outline" size="sm" className="mt-2" onClick={() => navigate(`/rolebindings?namespace=${namespace}`)}>View RoleBindings in {namespace}</Button>
    </SectionCard>
  );
}

function EffectiveSubjectsTab() {
  return (
    <SectionCard icon={Users} title="Subjects">
        <p className="text-muted-foreground text-sm">Subjects are derived from RoleBindings that reference this Role. View Bindings tab and open each RoleBinding to see subjects.</p>
    </SectionCard>
  );
}

export default function RoleDetail() {
  const customTabs: CustomTab[] = [
    { id: 'overview', label: 'Overview', render: (ctx) => <OverviewTab {...ctx} /> },
    { id: 'permission-matrix', label: 'Permission Matrix', render: (ctx) => <PermissionMatrixTab {...ctx} /> },
    { id: 'bindings', label: 'Bindings', render: (ctx) => <BindingsTab {...ctx} /> },
    { id: 'effective-subjects', label: 'Effective Subjects', render: () => <EffectiveSubjectsTab /> },
  ];

  return (
    <GenericResourceDetail<RoleResource>
      resourceType="roles"
      kind="Role"
      pluralLabel="Roles"
      listPath="/roles"
      resourceIcon={Shield}
      customTabs={customTabs}
      buildStatusCards={(ctx) => {
        const rules = ctx.resource?.rules ?? [];
        const apiGroupsSet = new Set<string>();
        rules.forEach((r) => (r.apiGroups ?? []).forEach((g) => apiGroupsSet.add(g || 'core')));
        const resourcesCoveredSet = new Set<string>();
        rules.forEach((r) => (r.resources ?? []).forEach((res) => resourcesCoveredSet.add(res)));
        const resourcesCoveredArr = Array.from(resourcesCoveredSet);
        const resourcesCovered = resourcesCoveredArr.slice(0, 5).join(', ') + (resourcesCoveredArr.length > 5 ? '…' : '');

        return [
          { label: 'Rules Count', value: rules.length, icon: Shield, iconColor: 'primary' as const },
          { label: 'API Groups', value: apiGroupsSet.size || '–', icon: Shield, iconColor: 'muted' as const },
          { label: 'Resources Covered', value: resourcesCovered || '–', icon: Shield, iconColor: 'muted' as const },
          { label: 'Bindings Count', value: '–', icon: Shield, iconColor: 'muted' as const },
        ];
      }}
    />
  );
}
