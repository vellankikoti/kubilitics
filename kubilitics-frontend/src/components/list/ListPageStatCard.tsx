import { useEffect, useRef } from 'react';
import { type LucideIcon } from 'lucide-react';
import { motion, useMotionValue, useTransform, animate, useReducedMotion } from 'framer-motion';
import { Card, CardContent } from '@/components/ui/card';
import { cn } from '@/lib/utils';

/**
 * Maps an iconColor like "text-emerald-600" to a matching gradient well class.
 * Falls back to primary blue when no match is found.
 */
function iconWellGradient(iconColor: string): string {
  if (iconColor.includes('emerald') || iconColor.includes('green'))
    return 'bg-gradient-to-br from-emerald-500/15 to-emerald-500/5 group-hover/stat:from-emerald-500/20 group-hover/stat:to-emerald-500/8';
  if (iconColor.includes('amber') || iconColor.includes('yellow'))
    return 'bg-gradient-to-br from-amber-500/15 to-amber-500/5 group-hover/stat:from-amber-500/20 group-hover/stat:to-amber-500/8';
  if (iconColor.includes('rose') || iconColor.includes('red'))
    return 'bg-gradient-to-br from-rose-500/15 to-rose-500/5 group-hover/stat:from-rose-500/20 group-hover/stat:to-rose-500/8';
  if (iconColor.includes('cyan') || iconColor.includes('teal'))
    return 'bg-gradient-to-br from-cyan-500/15 to-cyan-500/5 group-hover/stat:from-cyan-500/20 group-hover/stat:to-cyan-500/8';
  if (iconColor.includes('purple') || iconColor.includes('violet'))
    return 'bg-gradient-to-br from-purple-500/15 to-purple-500/5 group-hover/stat:from-purple-500/20 group-hover/stat:to-purple-500/8';
  if (iconColor.includes('blue'))
    return 'bg-gradient-to-br from-blue-500/15 to-blue-500/5 group-hover/stat:from-blue-500/20 group-hover/stat:to-blue-500/8';
  if (iconColor.includes('slate') || iconColor.includes('gray'))
    return 'bg-gradient-to-br from-slate-500/15 to-slate-500/5 group-hover/stat:from-slate-500/20 group-hover/stat:to-slate-500/8';
  // Default: primary
  return 'bg-gradient-to-br from-primary/15 to-primary/5 group-hover/stat:from-primary/20 group-hover/stat:to-primary/8';
}

/**
 * Animated number that counts up from 0 to the target value on mount.
 * Only animates pure numeric values; renders non-numeric values as-is.
 */
function AnimatedNumber({ value, className }: { value: React.ReactNode; className?: string }) {
  const reduceMotion = useReducedMotion();
  const motionVal = useMotionValue(0);
  const rounded = useTransform(motionVal, (v) => Math.round(v));
  const displayRef = useRef<HTMLSpanElement>(null);

  // Parse numeric value from ReactNode
  const numericValue = typeof value === 'number'
    ? value
    : typeof value === 'string' && /^\d+$/.test(value.trim())
      ? parseInt(value.trim(), 10)
      : null;

  useEffect(() => {
    if (numericValue === null || reduceMotion) return;
    motionVal.set(0);
    const controls = animate(motionVal, numericValue, {
      duration: 0.6,
      ease: [0.16, 1, 0.3, 1],
    });
    return controls.stop;
  }, [numericValue, motionVal, reduceMotion]);

  useEffect(() => {
    if (numericValue === null || reduceMotion) return;
    const unsub = rounded.on('change', (v) => {
      if (displayRef.current) displayRef.current.textContent = String(v);
    });
    return unsub;
  }, [rounded, numericValue, reduceMotion]);

  if (numericValue === null || reduceMotion) {
    return <div className={className}>{value}</div>;
  }

  return (
    <div className={className}>
      <span ref={displayRef}>{numericValue}</span>
    </div>
  );
}

export interface ListPageStatCardProps {
  label: string;
  value: React.ReactNode;
  icon?: LucideIcon;
  iconColor?: string;
  /** Custom class for the icon well background. Overrides auto-detected gradient. */
  iconWellClassName?: string;
  valueClassName?: string;
  selected?: boolean;
  onClick?: () => void;
  className?: string;
  /** 'sm' uses smaller label (text-xs); default uses text-sm */
  size?: 'default' | 'sm';
  /** Stagger index for entry animation delay (0-based). */
  index?: number;
  [key: string]: unknown;
}

/**
 * Reusable stat card for list pages. Icon well uses a gradient that
 * automatically matches the iconColor for visual consistency.
 */
export function ListPageStatCard({
  label,
  value,
  icon: Icon,
  iconColor = 'text-primary',
  iconWellClassName,
  valueClassName,
  selected,
  onClick,
  className,
  size = 'default',
  index = 0,
  ...rest
}: ListPageStatCardProps) {
  // Consume known extra props so they don't spread to DOM
  const { isLoading: _isLoading, ...domSafe } = rest as Record<string, unknown>;
  void _isLoading;

  const labelClass = 'text-xs font-semibold uppercase tracking-wider text-muted-foreground';
  const wellGradient = iconWellClassName ?? iconWellGradient(iconColor);

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, ease: [0.16, 1, 0.3, 1], delay: index * 0.06 }}
    >
      <Card
        className={cn(
          'relative overflow-hidden group/stat',
          'transition-all duration-300 ease-out',
          'hover:-translate-y-0.5 hover:shadow-[var(--shadow-2)] hover:border-border/80',
          onClick && 'cursor-pointer hover:border-primary/40',
          selected && 'ring-2 ring-primary/50 border-primary/30 bg-primary/[0.03] shadow-[var(--shadow-2)]',
          className
        )}
        onClick={onClick}
        role={onClick ? 'button' : undefined}
        tabIndex={onClick ? 0 : undefined}
        {...domSafe}
      >
        <CardContent className={size === 'sm' ? 'p-4' : 'p-5'}>
          <div className="flex items-center justify-between gap-3">
            <div className="min-w-0 flex-1">
              <p className={cn(labelClass, 'truncate mb-1.5')}>{label}</p>
              <AnimatedNumber
                value={value}
                className={cn('text-2xl font-bold tabular-nums tracking-tight', valueClassName)}
              />
            </div>
            {Icon && (
              <div className={cn(
                'flex items-center justify-center rounded-xl shrink-0',
                wellGradient,
                size === 'sm' ? 'h-10 w-10' : 'h-12 w-12',
                'transition-all duration-300',
                'shadow-sm border border-black/[0.03] dark:border-white/[0.05]',
              )}>
                <Icon className={cn(
                  size === 'sm' ? 'h-5 w-5' : 'h-6 w-6',
                  iconColor,
                  'transition-transform duration-300 group-hover/stat:scale-110',
                )} aria-hidden />
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </motion.div>
  );
}
