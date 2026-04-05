import { useState, useCallback, useMemo, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import {
  Gauge,
  Search,
  ArrowUpRight,
  Cpu,
  Zap,
  Ruler,
} from 'lucide-react';
import { EmptyState } from '@/components/ui/empty-state';
import { motion } from 'framer-motion';
import { useResourcesOverview } from '@/hooks/useResourcesOverview';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { cn } from '@/lib/utils';
import { SectionOverviewHeader } from '@/components/layout/SectionOverviewHeader';
import { QuotaPulse } from '@/components/resources/QuotaPulse';
import { ListPagination } from '@/components/list/ListPagination';
import { PageLayout } from '@/components/layout/PageLayout';
import { PageLoadingState } from '@/components/PageLoadingState';
import { ApiError } from '@/components/ui/error-state';

type ResourceItem = {
  kind: string;
  name: string;
  namespace: string;
  status: string;
};

function getResourceKey(r: ResourceItem): string {
  return `${r.kind}/${r.namespace}/${r.name}`;
}

export default function ResourcesOverview() {
  const [searchQuery, setSearchQuery] = useState('');
  const [isSyncing, setIsSyncing] = useState(false);
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());
  const [pageSize] = useState(10);
  const [pageIndex, setPageIndex] = useState(0);
  const queryClient = useQueryClient();
  const { data, isLoading, isError } = useResourcesOverview();

  const handleSync = useCallback(() => {
    setIsSyncing(true);
    queryClient.invalidateQueries({ queryKey: ['k8s'] });
    setTimeout(() => setIsSyncing(false), 1500);
  }, [queryClient]);

 // eslint-disable-next-line react-hooks/exhaustive-deps
  const resources: ResourceItem[] = data?.resources ?? [];

  const filteredResources = useMemo(() => {
    if (!searchQuery.trim()) return resources;
    const q = searchQuery.toLowerCase();
    return resources.filter(
      (r) => r.name.toLowerCase().includes(q) || r.kind.toLowerCase().includes(q)
    );
  }, [resources, searchQuery]);

  const totalFiltered = filteredResources.length;
  const totalPages = Math.max(1, Math.ceil(totalFiltered / pageSize));
  const safePageIndex = Math.min(pageIndex, totalPages - 1);
  const start = safePageIndex * pageSize;
  const itemsOnPage = filteredResources.slice(start, start + pageSize);

  useEffect(() => {
    if (safePageIndex !== pageIndex) setPageIndex(safePageIndex);
  }, [safePageIndex, pageIndex]);

  useEffect(() => { setPageIndex(0); }, [searchQuery]);

  const toggleSelection = (r: ResourceItem) => {
    const key = getResourceKey(r);
    setSelectedItems((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key); else next.add(key);
      return next;
    });
  };

  const toggleAll = () => {
    if (selectedItems.size === itemsOnPage.length) setSelectedItems(new Set());
    else setSelectedItems(new Set(itemsOnPage.map(getResourceKey)));
  };

  const isAllSelected = itemsOnPage.length > 0 && selectedItems.size === itemsOnPage.length;

  if (isError) {
    return (
      <PageLayout label="Resources Overview">
        <ApiError onRetry={() => queryClient.invalidateQueries({ queryKey: ['k8s'] })} />
      </PageLayout>
    );
  }

  if (isLoading) {
    return <PageLoadingState message="Loading resource constraints..." />;
  }

  const quotaCount = resources.filter((r) => r.kind === 'ResourceQuota').length;
  const limitCount = resources.filter((r) => r.kind === 'LimitRange').length;
  const sliceCount = resources.filter((r) => r.kind === 'ResourceSlice').length;
  const classCount = resources.filter((r) => r.kind === 'DeviceClass').length;

  return (
    <PageLayout label="Resources Overview">

      <SectionOverviewHeader
        title="Resources Overview"
        description="Resource quotas, limit ranges, and dynamic resource allocation."
        icon={Gauge}
        onSync={handleSync}
        isSyncing={isSyncing}
      />

      {/* Hero: Quota Pulse & DRA */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        <Card className="lg:col-span-8 overflow-hidden border-none soft-shadow glass-panel">
          <CardHeader className="pb-0 pt-8 px-8">
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="text-xl font-bold tracking-tight text-foreground">Resource Usage</CardTitle>
                <p className="text-sm text-muted-foreground mt-1">Quota allocation across CPU, memory, and storage</p>
              </div>
              <Badge variant="outline" className="text-xs font-semibold border-border text-muted-foreground">
                {quotaCount} quotas · {limitCount} limits
              </Badge>
            </div>
          </CardHeader>
          <CardContent className="pt-4 pb-8 px-8">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <QuotaPulse title="CPU Quota" percent={24} color="#3b82f6" />
              <QuotaPulse title="Memory Quota" percent={68} color="#8b5cf6" />
              <QuotaPulse title="Storage Quota" percent={42} color="#06b6d4" />
            </div>

            <div className="mt-6 border-t border-border/60 pt-5 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Badge variant="outline" className="text-xs font-medium border-border text-muted-foreground">{quotaCount} Quotas</Badge>
                <Badge variant="outline" className="text-xs font-medium border-border text-muted-foreground">{limitCount} Limits</Badge>
              </div>
              <Button variant="outline" asChild className="h-9 border-border text-muted-foreground font-medium hover:bg-muted rounded-lg">
                <Link to="/resource-quotas">View Quotas</Link>
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card className="lg:col-span-4 border-none soft-shadow glass-panel flex flex-col p-8 overflow-hidden">
          <div className="flex items-center gap-2 mb-4">
            <Cpu className="h-5 w-5 text-amber-500" />
            <h3 className="text-sm font-bold text-foreground">Dynamic Resource Allocation</h3>
          </div>
          <p className="text-xs text-muted-foreground mb-5">Request specialized hardware like GPUs and FPGAs beyond standard resource limits.</p>

          <div className="grid grid-cols-2 gap-3 mb-5">
            <div className="py-3 px-4 rounded-xl bg-muted border border-border/60">
              <span className="block text-xs font-medium text-muted-foreground">Resource Slices</span>
              <span className="text-xl font-bold text-foreground tabular-nums">{sliceCount}</span>
            </div>
            <div className="py-3 px-4 rounded-xl bg-muted border border-border/60">
              <span className="block text-xs font-medium text-muted-foreground">Device Classes</span>
              <span className="text-xl font-bold text-foreground tabular-nums">{classCount}</span>
            </div>
          </div>

          <div className="mt-auto">
            <Button variant="outline" asChild className="w-full h-9 border-border text-muted-foreground font-medium hover:bg-muted rounded-lg">
              <Link to="/resource-slices">Manage DRA</Link>
            </Button>
          </div>
        </Card>
      </div>

      {/* Resources Table */}
      <div className="bg-card border border-border/60 rounded-2xl overflow-hidden shadow-sm">
        <div className="p-6 border-b border-border/60">
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
            <div>
              <h3 className="text-lg font-bold tracking-tight text-foreground">Resource Constraints</h3>
              <p className="text-sm text-muted-foreground mt-0.5">Quotas, limit ranges, and device classes</p>
            </div>
            <div className="flex items-center gap-3">
              <div className="relative min-w-[280px]">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" aria-hidden />
                <Input
                  placeholder="Search constraints..."
                  className="pl-10 bg-muted border-border rounded-xl focus:bg-card focus:ring-2 focus:ring-blue-500/10 focus:border-blue-300 dark:focus:border-blue-600 h-10 text-sm"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  aria-label="Search resource constraints"
                />
              </div>
              {selectedItems.size > 0 && (
                <Badge variant="secondary" className="bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 border-blue-200 dark:border-blue-700">
                  {selectedItems.size} selected
                </Badge>
              )}
            </div>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-muted/60">
                <th className="px-6 py-3.5 border-b border-border/60 w-10">
                  <Checkbox checked={isAllSelected} onCheckedChange={toggleAll} />
                </th>
                <th className="px-6 py-3.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground border-b border-border/60">Name</th>
                <th className="px-6 py-3.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground border-b border-border/60">Kind</th>
                <th className="px-6 py-3.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground border-b border-border/60">Namespace</th>
                <th className="px-6 py-3.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground border-b border-border/60">Status</th>
                <th className="px-6 py-3.5 border-b border-border/60"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/30 text-sm">
              {itemsOnPage.map((resource, idx) => {
                const isSelected = selectedItems.has(getResourceKey(resource));
                return (
                  <motion.tr
                    initial={{ opacity: 0, y: 4 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: idx * 0.02 }}
                    key={getResourceKey(resource)}
                    className={cn('group hover:bg-muted/40 transition-colors', isSelected && 'bg-blue-50/40 dark:bg-blue-900/20')}
                  >
                    <td className="px-6 py-3.5">
                      <Checkbox checked={isSelected} onCheckedChange={() => toggleSelection(resource)} />
                    </td>
                    <td className="px-6 py-3.5">
                      <span className="font-semibold text-foreground group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">{resource.name}</span>
                    </td>
                    <td className="px-6 py-3.5">
                      <Badge variant="outline" className="text-xs uppercase tracking-wider font-semibold border-border text-muted-foreground">{resource.kind}</Badge>
                    </td>
                    <td className="px-6 py-3.5">
                      <span className="font-mono text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-md">{resource.namespace}</span>
                    </td>
                    <td className="px-6 py-3.5">
                      <div className="flex items-center gap-2">
                        <div className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                        <span className="text-xs font-medium text-foreground/80">Enforced</span>
                      </div>
                    </td>
                    <td className="px-6 py-3.5 text-right">
                      <Button variant="ghost" size="sm" className="h-8 w-8 p-0 hover:bg-card hover:text-blue-600 dark:hover:text-blue-400 hover:shadow-sm rounded-lg transition-all border border-transparent hover:border-border">
                        <ArrowUpRight className="h-4 w-4" aria-hidden />
                      </Button>
                    </td>
                  </motion.tr>
                );
              })}
              {itemsOnPage.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-6 py-16 text-center">
                    <EmptyState
                      icon={Ruler}
                      title={searchQuery ? "No resources match your search" : "No resource constraints found"}
                      description={searchQuery ? "Try adjusting your search terms." : "Resource quotas and limit ranges will appear here once configured."}
                      size="sm"
                      primaryAction={searchQuery ? { label: "Clear search", onClick: () => setSearchQuery('') } : { label: "View Quotas", href: "/resource-quotas" }}
                    />
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {totalFiltered > 0 && (
          <div className="p-4 border-t border-border/60 bg-muted/40 flex flex-col sm:flex-row items-center justify-between gap-4">
            <ListPagination
              rangeLabel={`${totalFiltered} ${totalFiltered === 1 ? 'resource' : 'resources'}`}
              hasPrev={safePageIndex > 0}
              hasNext={start + pageSize < totalFiltered}
              onPrev={() => setPageIndex((i) => Math.max(0, i - 1))}
              onNext={() => setPageIndex((i) => Math.min(totalPages - 1, i + 1))}
              currentPage={safePageIndex + 1}
              totalPages={totalPages}
              onPageChange={(p) => setPageIndex(p - 1)}
            />
            <div className="flex gap-2">
              <Button variant="outline" size="sm" asChild className="h-9 px-4 font-medium border-border text-muted-foreground hover:bg-card hover:text-blue-600 dark:hover:text-blue-400 rounded-lg transition-all">
                <Link to="/resource-quotas">Quotas</Link>
              </Button>
              <Button variant="outline" size="sm" asChild className="h-9 px-4 font-medium border-border text-muted-foreground hover:bg-card hover:text-blue-600 dark:hover:text-blue-400 rounded-lg transition-all">
                <Link to="/limit-ranges">Limits</Link>
              </Button>
            </div>
          </div>
        )}
      </div>
    </PageLayout>
  );
}
