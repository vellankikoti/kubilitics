import { motion } from 'framer-motion';
import { cn } from '@/lib/utils';
import { ScoreTooltip } from './ScoreTooltip';
import type { SubScores, ImpactSummary } from '@/services/api/types';

export interface RiskIndicatorCardsProps {
  subScores: SubScores;
  blastRadiusPercent: number;
  impactSummary: ImpactSummary;
  coverageLevel: string;
  onOpenDetail: (section: 'resilience' | 'exposure' | 'recovery' | 'impact') => void;
}

function scoreColor(score: number): string {
  if (score >= 70) return 'text-green-500';
  if (score >= 40) return 'text-yellow-500';
  return 'text-red-500';
}

function resilienceBadge(score: number) {
  if (score >= 70) return { bg: 'bg-green-500/10', text: 'text-green-500', label: 'STRONG' };
  if (score >= 40) return { bg: 'bg-yellow-500/10', text: 'text-yellow-500', label: 'MODERATE' };
  return { bg: 'bg-red-500/10', text: 'text-red-500', label: 'WEAK' };
}

function impactBadge(pct: number) {
  if (pct === 0) return { bg: 'bg-green-500/10', text: 'text-green-500', label: 'NONE' };
  if (pct < 5) return { bg: 'bg-green-500/10', text: 'text-green-500', label: 'LOW' };
  if (pct < 20) return { bg: 'bg-yellow-500/10', text: 'text-yellow-500', label: 'MODERATE' };
  return { bg: 'bg-red-500/10', text: 'text-red-500', label: 'HIGH' };
}

function exposureBadge(score: number) {
  if (score < 20) return { bg: 'bg-green-500/10', text: 'text-green-500', label: 'LOW' };
  if (score <= 50) return { bg: 'bg-yellow-500/10', text: 'text-yellow-500', label: 'MODERATE' };
  return { bg: 'bg-red-500/10', text: 'text-red-500', label: 'HIGH' };
}

function recoveryBadge(score: number) {
  if (score >= 70) return { bg: 'bg-green-500/10', text: 'text-green-500', label: 'FAST' };
  if (score >= 40) return { bg: 'bg-yellow-500/10', text: 'text-yellow-500', label: 'MODERATE' };
  return { bg: 'bg-red-500/10', text: 'text-red-500', label: 'SLOW' };
}

function impactColor(pct: number): string {
  if (pct < 5) return 'text-green-500';
  if (pct < 20) return 'text-yellow-500';
  return 'text-red-500';
}

function contextLine(factors: { note: string }[]): string {
  return factors.slice(0, 3).map(f => f.note).join(' · ').slice(0, 45);
}

export function RiskIndicatorCards({
  subScores,
  blastRadiusPercent,
  impactSummary,
  coverageLevel,
  onOpenDetail,
}: RiskIndicatorCardsProps) {
  const cards = [
    {
      key: 'resilience' as const,
      label: 'Resilience',
      score: subScores.resilience.score,
      displayValue: String(subScores.resilience.score),
      color: scoreColor(subScores.resilience.score),
      badge: resilienceBadge(subScores.resilience.score),
      context: contextLine(subScores.resilience.factors),
      factors: subScores.resilience.factors,
      extraBadge: null as string | null,
    },
    {
      key: 'impact' as const,
      label: 'Cluster Impact',
      score: Math.round(blastRadiusPercent),
      displayValue: `${blastRadiusPercent.toFixed(1)}%`,
      color: impactColor(blastRadiusPercent),
      badge: impactBadge(blastRadiusPercent),
      context: impactSummary.brokenCount === 0 && impactSummary.degradedCount === 0
        ? 'Self-healing'
        : `${impactSummary.brokenCount} broken · ${impactSummary.degradedCount} degraded`,
      factors: subScores.impact.factors,
      extraBadge: null as string | null,
    },
    {
      key: 'exposure' as const,
      label: 'Exposure',
      score: subScores.exposure.score,
      displayValue: String(subScores.exposure.score),
      color: subScores.exposure.score < 20 ? 'text-green-500' : subScores.exposure.score <= 50 ? 'text-yellow-500' : 'text-red-500',
      badge: exposureBadge(subScores.exposure.score),
      context: contextLine(subScores.exposure.factors),
      factors: subScores.exposure.factors,
      extraBadge: coverageLevel === 'partial' ? 'Partial' : null,
    },
    {
      key: 'recovery' as const,
      label: 'Recovery',
      score: subScores.recovery.score,
      displayValue: String(subScores.recovery.score),
      color: scoreColor(subScores.recovery.score),
      badge: recoveryBadge(subScores.recovery.score),
      context: contextLine(subScores.recovery.factors),
      factors: subScores.recovery.factors,
      extraBadge: null as string | null,
    },
  ];

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
      {cards.map((card, index) => (
        <motion.div
          key={card.key}
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.35, ease: 'easeOut', delay: index * 0.05 }}
        >
          <ScoreTooltip
            title={card.label}
            score={card.score}
            factors={card.factors}
            onViewDetails={() => onOpenDetail(card.key)}
          >
            <div className="border-none soft-shadow glass-panel rounded-xl p-4 cursor-pointer hover:shadow-md transition-shadow">
              <div className="flex items-center justify-between mb-1">
                <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  {card.label}
                </span>
                <div className="flex items-center gap-1.5">
                  {card.extraBadge && (
                    <span className="px-1.5 py-0.5 rounded-full text-[10px] font-semibold bg-amber-500/10 text-amber-500">
                      {card.extraBadge}
                    </span>
                  )}
                  <span className={cn('px-2 py-0.5 rounded-full text-[10px] font-semibold', card.badge.bg, card.badge.text)}>
                    {card.badge.label}
                  </span>
                </div>
              </div>
              <div className={cn('text-[28px] font-bold leading-none my-2', card.color)}>
                {card.displayValue}
              </div>
              <div className="text-[11px] text-muted-foreground truncate">
                {card.context}
              </div>
            </div>
          </ScoreTooltip>
        </motion.div>
      ))}
    </div>
  );
}
