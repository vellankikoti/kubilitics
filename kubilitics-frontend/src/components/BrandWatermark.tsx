/**
 * BrandWatermark — Persistent Kubilitics branding for fullscreen/presentation views.
 *
 * Shows the logo + wordmark in a subtle glass-morphism pill. Positioned at
 * top-left by default (opposite the Exit button at top-right). Ensures brand
 * visibility when the main app chrome (header, sidebar) is hidden.
 */
import { BrandLogo } from './BrandLogo';
import { cn } from '@/lib/utils';

interface BrandWatermarkProps {
  /** Position on screen. Default: 'top-left' */
  position?: 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right';
  /** Logo height in pixels. Default: 20 */
  logoHeight?: number;
  /** Additional CSS classes */
  className?: string;
}

const positionClasses = {
  'top-left': 'top-4 left-4',
  'top-right': 'top-4 right-4',
  'bottom-left': 'bottom-4 left-4',
  'bottom-right': 'bottom-4 right-4',
};

export function BrandWatermark({
  position = 'top-left',
  logoHeight = 36,
  className,
}: BrandWatermarkProps) {
  return (
    <div
      className={cn(
        'absolute z-50 flex items-center gap-2.5 rounded-xl',
        'border border-white/20 dark:border-slate-700/50',
        'bg-white/80 dark:bg-slate-900/80 backdrop-blur-md',
        'px-4 py-2.5 shadow-lg select-none pointer-events-none',
        positionClasses[position],
        className,
      )}
    >
      <BrandLogo height={logoHeight} variant="dark" className="dark:hidden" />
      <BrandLogo height={logoHeight} variant="light" className="hidden dark:block" />
    </div>
  );
}
