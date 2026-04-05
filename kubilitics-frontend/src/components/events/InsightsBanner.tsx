/**
 * InsightsBanner — proactive alert banner at the top of the Events page.
 */
import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { AlertTriangle, ChevronDown, X, ArrowRight, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import type { Insight } from '@/services/api/eventsIntelligence';

interface InsightsBannerProps {
  insights: Insight[];
  onInvestigate: (insight: Insight) => void;
  onDismiss: (insightId: string) => void;
  isDismissing?: boolean;
}

const SEVERITY_STYLES: Record<string, string> = {
  critical: 'bg-red-500/10 border-red-500/20',
  high: 'bg-amber-500/10 border-amber-500/20',
  medium: 'bg-yellow-500/10 border-yellow-500/20',
  low: 'bg-blue-500/10 border-blue-500/20',
};

const SEVERITY_TEXT: Record<string, string> = {
  critical: 'text-red-600 dark:text-red-400',
  high: 'text-amber-600 dark:text-amber-400',
  medium: 'text-yellow-600 dark:text-yellow-400',
  low: 'text-blue-600 dark:text-blue-400',
};

export function InsightsBanner({ insights, onInvestigate, onDismiss, isDismissing }: InsightsBannerProps) {
  const [expanded, setExpanded] = useState(false);

  if (insights.length === 0) return null;

  const primary = insights[0];
  const severityBg = SEVERITY_STYLES[primary.severity] ?? SEVERITY_STYLES.medium;
  const severityText = SEVERITY_TEXT[primary.severity] ?? SEVERITY_TEXT.medium;

  return (
    <motion.div
      initial={{ opacity: 0, y: -8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
      className={cn('rounded-xl border p-4', severityBg)}
    >
      <div className="flex items-start gap-3">
        <AlertTriangle className={cn('h-5 w-5 mt-0.5 shrink-0', severityText)} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h3 className={cn('text-sm font-semibold', severityText)}>{primary.title}</h3>
            <Badge variant="outline" className={cn('text-[10px] px-1.5 py-0 h-5', severityText)}>
              {primary.severity}
            </Badge>
            {insights.length > 1 && (
              <Badge variant="secondary" className="text-[10px] h-5">
                +{insights.length - 1} more
              </Badge>
            )}
          </div>
          <p className="text-xs text-muted-foreground mt-1">{primary.detail}</p>

          {/* Actions */}
          <div className="flex items-center gap-2 mt-2">
            <Button
              variant="outline"
              size="sm"
              className="h-7 text-xs gap-1"
              onClick={() => onInvestigate(primary)}
            >
              Investigate
              <ArrowRight className="h-3 w-3" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 text-xs"
              onClick={() => onDismiss(primary.insight_id)}
              disabled={isDismissing}
            >
              {isDismissing ? <Loader2 className="h-3 w-3 animate-spin" /> : 'Dismiss'}
            </Button>
            {insights.length > 1 && (
              <Button
                variant="ghost"
                size="sm"
                className="h-7 text-xs gap-1"
                onClick={() => setExpanded(!expanded)}
              >
                {expanded ? 'Collapse' : `Show ${insights.length - 1} more`}
                <ChevronDown className={cn('h-3 w-3 transition-transform', expanded && 'rotate-180')} />
              </Button>
            )}
          </div>
        </div>
      </div>

      {/* Additional insights */}
      <AnimatePresence>
        {expanded && insights.length > 1 && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="overflow-hidden mt-3 space-y-2 pl-8"
          >
            {insights.slice(1).map((insight) => (
              <div
                key={insight.insight_id}
                className="flex items-center justify-between py-2 border-t border-border/40"
              >
                <div className="min-w-0">
                  <p className="text-xs font-medium">{insight.title}</p>
                  <p className="text-[10px] text-muted-foreground truncate">{insight.detail}</p>
                </div>
                <div className="flex gap-1 shrink-0 ml-2">
                  <Button variant="ghost" size="sm" className="h-6 text-[10px] px-2" onClick={() => onInvestigate(insight)}>
                    Investigate
                  </Button>
                  <Button variant="ghost" size="sm" className="h-6 text-[10px] px-2" onClick={() => onDismiss(insight.insight_id)}>
                    Dismiss
                  </Button>
                </div>
              </div>
            ))}
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}
