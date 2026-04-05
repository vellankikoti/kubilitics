/**
 * WorkloadLogsTab — reusable Logs tab for workload resources.
 *
 * Extracts the common pod-selector + container-selector + LogViewer pattern
 * used across Deployment, StatefulSet, DaemonSet, Job, and CronJob detail pages.
 *
 * Consumers pass in the pre-filtered list of pods owned by the workload,
 * the workload kind label, and optional fallback container names from the
 * pod template spec.
 */

import { useState } from 'react';
import { FileText } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { LogViewer } from './LogViewer';
import { SectionCard } from './SectionCard';

// ─── Types ─────────────────────────────────────────────────────────────────────

export interface WorkloadPod {
  metadata?: {
    name?: string;
    namespace?: string;
    labels?: Record<string, string>;
    ownerReferences?: Array<{ kind?: string; name?: string }>;
  };
  spec?: {
    containers?: Array<{ name: string }>;
    nodeName?: string;
  };
  status?: {
    phase?: string;
    containerStatuses?: Array<{ ready?: boolean }>;
  };
}

export interface WorkloadLogsTabProps {
  /** Pre-filtered pods owned by the workload */
  pods: WorkloadPod[];
  /** Namespace of the workload (for log streaming) */
  namespace?: string;
  /** Kind label for display, e.g. "Deployment", "StatefulSet" */
  kindLabel: string;
  /** Fallback container names from the workload pod template spec */
  templateContainers?: string[];
  /** Optional className for the root element */
  className?: string;
}

// ─── Component ─────────────────────────────────────────────────────────────────

export function WorkloadLogsTab({
  pods,
  namespace,
  kindLabel,
  templateContainers = [],
}: WorkloadLogsTabProps) {
  const [selectedLogPod, setSelectedLogPod] = useState<string>('');
  const [selectedLogContainer, setSelectedLogContainer] = useState<string>('');

  const firstPodName = pods[0]?.metadata?.name ?? '';
  const logPod = selectedLogPod || firstPodName;

  // Derive container list from the selected pod, falling back to template containers
  const logPodContainers =
    pods
      .find((p) => p.metadata?.name === logPod)
      ?.spec?.containers?.map((c) => c.name) ?? templateContainers;

  return (
    <SectionCard
      icon={FileText}
      title="Logs"
      tooltip={
        <p className="text-xs text-muted-foreground">
          Stream logs from {kindLabel} pods
        </p>
      }
    >
      {pods.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No pods available to view logs.
        </p>
      ) : (
        <div className="space-y-4">
          <div className="flex flex-wrap gap-4 items-end">
            <div className="space-y-2">
              <Label className="flex items-center gap-2">
                Pod
                <Badge variant="secondary" className="text-xs font-normal">
                  {pods.length} {pods.length === 1 ? 'pod' : 'pods'}
                </Badge>
              </Label>
              <Select value={logPod} onValueChange={setSelectedLogPod}>
                <SelectTrigger className="w-[280px]">
                  <SelectValue placeholder="Select pod" />
                </SelectTrigger>
                <SelectContent>
                  {pods.map((p) => (
                    <SelectItem
                      key={p.metadata?.name}
                      value={p.metadata?.name ?? ''}
                    >
                      {p.metadata?.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            {logPodContainers.length > 1 && (
              <div className="space-y-2">
                <Label>Container</Label>
                <Select
                  value={selectedLogContainer || logPodContainers[0]}
                  onValueChange={setSelectedLogContainer}
                >
                  <SelectTrigger className="w-[180px]">
                    <SelectValue placeholder="Select container" />
                  </SelectTrigger>
                  <SelectContent>
                    {logPodContainers.map((c) => (
                      <SelectItem key={c} value={c}>
                        {c}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
          </div>
          <LogViewer
            podName={logPod}
            namespace={namespace}
            containerName={selectedLogContainer || logPodContainers[0]}
            containers={logPodContainers}
            onContainerChange={setSelectedLogContainer}
          />
        </div>
      )}
    </SectionCard>
  );
}
