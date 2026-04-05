import { useState, useCallback, useMemo, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import {
  Globe,
  Search,
  ArrowUpRight,
  Network,
} from 'lucide-react';
import { EmptyState } from '@/components/ui/empty-state';
import { motion } from 'framer-motion';
import { useNetworkingOverview } from '@/hooks/useNetworkingOverview';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { cn } from '@/lib/utils';
import { SectionOverviewHeader } from '@/components/layout/SectionOverviewHeader';
import { NetworkingPulse } from '@/components/networking/NetworkingPulse';
import { ServiceDistribution } from '@/components/networking/ServiceDistribution';
import { ListPagination } from '@/components/list/ListPagination';
import { PageLayout } from '@/components/layout/PageLayout';
import { PageLoadingState } from '@/components/PageLoadingState';
import { ApiError } from '@/components/ui/error-state';

type NetworkResource = {
  kind: string;
  name: string;
  namespace: string;
  status: string;
  type?: string;
};

function getResourceKey(r: NetworkResource): string {
  return `${r.kind}/${r.namespace}/${r.name}`;
}

export default function NetworkingOverview() {
  const [searchQuery, setSearchQuery] = useState('');
  const [isSyncing, setIsSyncing] = useState(false);
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());
  const [pageSize] = useState(10);
  const [pageIndex, setPageIndex] = useState(0);
  const queryClient = useQueryClient();
  const { data, isLoading, isError } = useNetworkingOverview();

  const handleSync = useCallback(() => {
    setIsSyncing(true);
    queryClient.invalidateQueries({ queryKey: ['k8s'] });
    setTimeout(() => setIsSyncing(false), 1500);
  }, [queryClient]);

 // eslint-disable-next-line react-hooks/exhaustive-deps
  const resources: NetworkResource[] = data?.resources ?? [];

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

  const toggleSelection = (r: NetworkResource) => {
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
      <PageLayout label="Networking Overview">
        <ApiError onRetry={() => queryClient.invalidateQueries({ queryKey: ['k8s'] })} />
      </PageLayout>
    );
  }

  if (isLoading) {
    return <PageLoadingState message="Loading network resources..." />;
  }

  const servicesCount = resources.filter((r) => r.kind === 'Service').length;
  const ingressCount = resources.filter((r) => r.kind === 'Ingress').length;
  const policyCount = resources.filter((r) => r.kind === 'NetworkPolicy').length;

  return (
    <PageLayout label="Networking Overview">

      <SectionOverviewHeader
        title="Networking Overview"
        description="Services, ingress rules, and network policies across your cluster."
        icon={Globe}
        onSync={handleSync}
        isSyncing={isSyncing}
      />

      {/* Hero: Traffic Pulse & Distribution */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        <Card className="lg:col-span-8 overflow-hidden border-none soft-shadow glass-panel" aria-live="polite">
          <CardHeader className="pb-0 pt-8 px-8">
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="text-xl font-bold tracking-tight text-foreground">Traffic Health</CardTitle>
                <p className="text-sm text-muted-foreground mt-1">Real-time connectivity and availability</p>
              </div>
              <div className="flex items-center gap-2">
                <span className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse" />
                <span className="text-xs font-semibold text-muted-foreground">Live</span>
              </div>
            </div>
          </CardHeader>
          <CardContent className="pt-0 px-8 pb-8">
            <NetworkingPulse />
            <div className="grid grid-cols-3 gap-4 mt-4 border-t border-border/60 pt-6">
              <div className="text-center">
                <span className="block text-2xl font-bold text-foreground tabular-nums">{data?.pulse.optimal_percent.toFixed(1)}%</span>
                <span className="text-xs font-medium text-muted-foreground">Availability</span>
              </div>
              <div className="text-center">
                <span className="block text-2xl font-bold text-foreground tabular-nums">{servicesCount}</span>
                <span className="text-xs font-medium text-muted-foreground">Services</span>
              </div>
              <div className="text-center">
                <span className="block text-2xl font-bold text-foreground tabular-nums">{policyCount}</span>
                <span className="text-xs font-medium text-muted-foreground">Policies</span>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="lg:col-span-4 border-none soft-shadow glass-panel flex flex-col items-center justify-center p-6 px-0 text-center overflow-hidden">
          <CardHeader className="pb-0">
            <CardTitle className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">Service Distribution</CardTitle>
          </CardHeader>
          <CardContent className="w-full flex flex-col items-center">
            <ServiceDistribution data={{ services: servicesCount, ingresses: ingressCount, policies: policyCount }} />
            <div className="flex flex-wrap justify-center gap-4 mt-2">
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-blue-500" />
                <span className="text-xs font-medium text-muted-foreground">Services</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-blue-300" />
                <span className="text-xs font-medium text-muted-foreground">Ingress</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-blue-200" />
                <span className="text-xs font-medium text-muted-foreground">Policies</span>
              </div>
            </div>
          </CardContent>
          <div className="mt-4 w-full px-6">
            <Button variant="outline" asChild className="w-full h-9 border-border text-muted-foreground font-medium hover:bg-muted rounded-lg transition-all">
              <Link to="/services">View All Services</Link>
            </Button>
          </div>
        </Card>
      </div>

      {/* Resources Table */}
      <div className="bg-card border border-border/60 rounded-2xl overflow-hidden shadow-sm">
        <div className="p-6 border-b border-border/60">
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
            <div>
              <h3 className="text-lg font-bold tracking-tight text-foreground">Network Resources</h3>
              <p className="text-sm text-muted-foreground mt-0.5">Services, ingress rules, and network policies</p>
            </div>
            <div className="flex items-center gap-3">
              <div className="relative min-w-[280px]">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" aria-hidden />
                <Input
                  placeholder="Search network resources..."
                  className="pl-10 bg-muted border-border rounded-xl focus:bg-card focus:ring-2 focus:ring-blue-500/10 focus:border-blue-300 dark:focus:border-blue-600 h-10 text-sm"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  aria-label="Search network resources"
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
                <th className="px-6 py-3.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground border-b border-border/60">Type</th>
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
                      <Badge variant="outline" className="text-xs uppercase tracking-wider font-semibold border-border text-muted-foreground">
                        {resource.kind}
                      </Badge>
                    </td>
                    <td className="px-6 py-3.5">
                      <span className="font-mono text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-md">{resource.namespace}</span>
                    </td>
                    <td className="px-6 py-3.5">
                      <div className="flex items-center gap-2">
                        <div className={cn('h-1.5 w-1.5 rounded-full', resource.status === 'Active' ? 'bg-emerald-500' : 'bg-amber-500')} />
                        <span className="text-xs font-medium text-foreground/80">{resource.status}</span>
                      </div>
                    </td>
                    <td className="px-6 py-3.5">
                      <span className="text-xs text-muted-foreground">{resource.type || '—'}</span>
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
                  <td colSpan={7} className="px-6 py-16 text-center">
                    <EmptyState
                      icon={Network}
                      title={searchQuery ? "No resources match your search" : "No network resources found"}
                      description={searchQuery ? "Try adjusting your search terms." : "Services, ingress rules, and network policies will appear here once created."}
                      size="sm"
                      primaryAction={searchQuery ? { label: "Clear search", onClick: () => setSearchQuery('') } : { label: "View Services", href: "/services" }}
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
                <Link to="/services">Services</Link>
              </Button>
              <Button variant="outline" size="sm" asChild className="h-9 px-4 font-medium border-border text-muted-foreground hover:bg-card hover:text-blue-600 dark:hover:text-blue-400 rounded-lg transition-all">
                <Link to="/ingresses">Ingresses</Link>
              </Button>
              <Button variant="outline" size="sm" asChild className="h-9 px-4 font-medium border-border text-muted-foreground hover:bg-card hover:text-blue-600 dark:hover:text-blue-400 rounded-lg transition-all">
                <Link to="/network-policies">Policies</Link>
              </Button>
            </div>
          </div>
        )}
      </div>
    </PageLayout>
  );
}
