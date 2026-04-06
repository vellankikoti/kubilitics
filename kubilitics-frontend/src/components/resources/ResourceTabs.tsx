import { ReactNode, useId } from 'react';
import { motion } from 'framer-motion';
import { LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';

/**
 * Tabs whose content should stay mounted (hidden) when switching away,
 * so stateful widgets (xterm terminals, file browsers) are preserved.
 */
const KEEP_ALIVE_TABS = new Set(['terminal', 'shell', 'files']);

export interface TabConfig {
  id: string;
  label: string;
  content: ReactNode;
  /** Optional icon shown left of label */
  icon?: LucideIcon;
  /** Optional badge (e.g. event count, "Live") shown as small pill */
  badge?: number | string;
}

export interface ResourceTabsProps {
  tabs: TabConfig[];
  activeTab: string;
  onTabChange: (tabId: string) => void;
  className?: string;
}

export function ResourceTabs({ tabs, activeTab, onTabChange, className }: ResourceTabsProps) {
  const instanceId = useId();

  return (
    <div className={cn('space-y-6 w-full', className)}>
      {/* Tab bar — sticky so it stays visible while scrolling through content */}
      <div className="sticky top-0 z-30 -mx-1 px-1 pt-1 pb-2 bg-background/95 backdrop-blur-sm">
        <div className="w-full rounded-xl bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-1 overflow-x-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]">
          <nav className="flex items-center gap-0.5" aria-label="Tabs">
          {tabs.map((tab) => {
            const isActive = activeTab === tab.id;
            const Icon = tab.icon;

            return (
              <button
                key={tab.id}
                onClick={() => onTabChange(tab.id)}
                className={cn(
                  'relative flex items-center gap-1.5 shrink-0 px-3.5 py-2 rounded-lg text-[13px] font-medium transition-colors duration-150',
                  'focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/40',
                  isActive
                    ? 'text-foreground font-semibold'
                    : 'text-slate-600 dark:text-slate-400 hover:text-foreground hover:bg-slate-50 dark:hover:bg-slate-700/50'
                )}
              >
                {/* Animated background pill for active tab */}
                {isActive && (
                  <motion.div
                    layoutId={`tab-pill-${instanceId}`}
                    className="absolute inset-0 bg-white dark:bg-slate-700 rounded-lg shadow-sm border border-slate-200 dark:border-slate-600 border-b-2 border-b-primary"
                    transition={{ type: 'spring', stiffness: 400, damping: 30 }}
                    style={{ zIndex: 0 }}
                  />
                )}
                <span className="relative z-10 flex items-center gap-1.5">
                  {Icon && <Icon className={cn('h-3.5 w-3.5 shrink-0', isActive ? 'text-primary' : '')} aria-hidden />}
                  <span>{tab.label}</span>
                  {tab.badge != null && (
                    <span
                      className={cn(
                        'shrink-0 min-w-[1.125rem] h-[1.125rem] px-1 rounded-full text-[10px] font-semibold flex items-center justify-center leading-none',
                        isActive
                          ? 'bg-primary text-primary-foreground'
                          : 'bg-foreground/10 text-foreground/60'
                      )}
                    >
                      {typeof tab.badge === 'number' && tab.badge > 99 ? '99+' : tab.badge}
                    </span>
                  )}
                </span>
              </button>
            );
          })}
        </nav>
        </div>
      </div>

      {/* Tab content — keep-alive tabs stay mounted but hidden to preserve state */}
      {tabs.map((tab) => {
        const isActive = tab.id === activeTab;
        const keepAlive = KEEP_ALIVE_TABS.has(tab.id);

        // Non-active, non-keepalive tabs are fully unmounted for performance
        if (!isActive && !keepAlive) return null;

        if (isActive) {
          return (
            <motion.div
              key={tab.id}
              initial={{ opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.15, ease: 'easeOut' }}
              className="min-h-[60vh]"
              role="tabpanel"
            >
              {tab.content}
            </motion.div>
          );
        }

        // Keep-alive but inactive: visually hidden but preserves real dimensions
        // so xterm/WebSocket stays connected and doesn't refit to 0×0.
        return (
          <div
            key={tab.id}
            style={{
              visibility: 'hidden',
              position: 'absolute',
              left: '-9999px',
              width: '100%',
              height: '60vh',
              overflow: 'hidden',
              pointerEvents: 'none',
            }}
            aria-hidden
          >
            {tab.content}
          </div>
        );
      })}
    </div>
  );
}
