import { ReactNode, useCallback, useState } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { ChevronLeft, Copy, Check, LucideIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { NamespaceBadge } from '@/components/list';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { toast } from '@/components/ui/sonner';
import { cn } from '@/lib/utils';

export type ResourceStatus = 'Running' | 'Pending' | 'Succeeded' | 'Failed' | 'Unknown' | 'Healthy' | 'Warning' | 'Error';

export interface ResourceAction {
  label: string;
  icon: LucideIcon;
  onClick?: () => void;
  variant?: 'default' | 'outline' | 'destructive';
}

export interface ResourceHeaderProps {
  resourceType: string;
  resourceIcon: LucideIcon;
  name: string;
  namespace?: string;
  status: ResourceStatus;
  backLink: string;
  backLabel: string;
  actions?: ResourceAction[];
  metadata?: ReactNode;
  /** Relative created label (e.g. "2h ago"). Shown with tooltip when createdAt is provided. */
  createdLabel?: string;
  /** ISO timestamp for tooltip on created time */
  createdAt?: string;
}

const statusConfig: Record<ResourceStatus, { bg: string; text: string; icon: string }> = {
  Running: { bg: 'bg-[hsl(var(--success)/0.1)]', text: 'text-[hsl(var(--success))]', icon: '✓' },
  Healthy: { bg: 'bg-[hsl(var(--success)/0.1)]', text: 'text-[hsl(var(--success))]', icon: '✓' },
  Succeeded: { bg: 'bg-[hsl(var(--success)/0.1)]', text: 'text-[hsl(var(--success))]', icon: '✓' },
  Pending: { bg: 'bg-[hsl(var(--warning)/0.1)]', text: 'text-[hsl(var(--warning))]', icon: '◔' },
  Warning: { bg: 'bg-[hsl(var(--warning)/0.1)]', text: 'text-[hsl(var(--warning))]', icon: '⚠' },
  Failed: { bg: 'bg-[hsl(var(--error)/0.1)]', text: 'text-[hsl(var(--error))]', icon: '✗' },
  Error: { bg: 'bg-[hsl(var(--error)/0.1)]', text: 'text-[hsl(var(--error))]', icon: '✗' },
  Unknown: { bg: 'bg-muted', text: 'text-muted-foreground', icon: '?' },
};

export function ResourceHeader({
  resourceType,
  resourceIcon: Icon,
  name,
  namespace,
  status,
  backLink,
  backLabel,
  actions = [],
  metadata,
  createdLabel,
  createdAt,
}: ResourceHeaderProps) {
  const statusStyle = statusConfig[status] || statusConfig.Unknown;
  const [isCopied, setIsCopied] = useState(false);

  const copyDisplayName = useCallback(() => {
    const toCopy = namespace ? `${namespace}/${name}` : name;
    navigator.clipboard.writeText(toCopy);
    toast.success('Copied to clipboard');

    // Show checkmark animation for 1.5s
    setIsCopied(true);
    setTimeout(() => {
      setIsCopied(false);
    }, 1500);
  }, [namespace, name]);

  const createdTooltip = createdAt
    ? new Date(createdAt).toISOString()
    : createdLabel
      ? createdLabel
      : null;

  return (
      <motion.div
        initial={{ opacity: 0, y: -10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.2 }}
        className="space-y-4"
      >
        {/* Back Link */}
        <Link
          to={backLink}
          aria-label={`Back to ${backLabel}`}
          className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          <ChevronLeft className="h-4 w-4" />
          Back to {backLabel}
        </Link>

        {/* Main Header */}
        <div
          role="banner"
          className="elevation-2 rounded-xl border border-border/50 bg-card p-4 sm:p-5"
        >
          {/* Top row: Icon + Name + Status */}
          <div className="flex items-start gap-4">
            <div className="p-2.5 rounded-xl bg-primary/10 shrink-0">
              <Icon className="h-7 w-7 text-primary" />
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <h1 className="text-lg sm:text-xl font-semibold tracking-tight truncate">{name}</h1>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 shrink-0 text-muted-foreground hover:text-foreground"
                      onClick={copyDisplayName}
                      aria-label="Copy resource name"
                    >
                      {isCopied ? (
                        <Check className="h-3.5 w-3.5 text-green-600" />
                      ) : (
                        <Copy className="h-3.5 w-3.5" />
                      )}
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    {isCopied ? 'Copied!' : `Copy ${namespace ? 'namespace/name' : 'name'}`}
                  </TooltipContent>
                </Tooltip>
                <div className={cn(
                  'flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium shrink-0',
                  statusStyle.bg,
                  statusStyle.text
                )}>
                  <span>{statusStyle.icon}</span>
                  {status}
                </div>
              </div>
              <div className="flex items-center gap-2 mt-1.5 text-xs text-muted-foreground flex-wrap">
                <span className="flex items-center gap-1.5">
                  {resourceType}
                  {namespace ? (
                    <>
                      {' '}in
                      <NamespaceBadge namespace={namespace} className="ml-0.5" />
                    </>
                  ) : (
                    <Badge variant="secondary" className="ml-1 text-[10px]">Cluster-scoped</Badge>
                  )}
                </span>
                {(createdLabel ?? createdAt) && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="flex items-center gap-1.5">
                        Created {createdLabel ?? (createdAt ? new Date(createdAt).toLocaleString() : '')}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>{createdTooltip}</TooltipContent>
                  </Tooltip>
                )}
                {metadata}
              </div>
            </div>
          </div>

          {/* Actions row — separate line, right-aligned */}
          {actions.length > 0 && (
            <div className="flex items-center gap-2 mt-3 pt-3 border-t border-border/40 justify-end flex-wrap">
              {actions.map((action) => (
                <Button
                  key={action.label}
                  variant={action.variant || 'outline'}
                  size="sm"
                  onClick={action.onClick}
                  aria-label={action.label}
                  className="gap-1.5 text-xs"
                >
                  <action.icon className="h-3.5 w-3.5" />
                  {action.label}
                </Button>
              ))}
            </div>
          )}
        </div>
      </motion.div>
  );
}
