import { cn } from '@/lib/utils';
import { ConnectionRequiredBanner } from '@/components/layout/ConnectionRequiredBanner';

interface PageLayoutProps {
  children: React.ReactNode;
  /** aria-label for the main content area */
  label: string;
  /** Show connection required banner (default: true) */
  showBanner?: boolean;
  className?: string;
}

export function PageLayout({
  children,
  label,
  showBanner = true,
  className,
}: PageLayoutProps) {
  return (
    <div className="page-container" role="main" aria-label={label}>
      <div className={cn("page-inner p-6 gap-6 flex flex-col", className)}>
        {showBanner && <ConnectionRequiredBanner />}
        {children}
      </div>
    </div>
  );
}

PageLayout.displayName = "PageLayout";
