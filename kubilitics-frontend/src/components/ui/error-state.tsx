import { motion } from "framer-motion";
import { type LucideIcon } from "lucide-react";
import {
  WifiOff,
  AlertCircle,
  ShieldX,
  Clock,
  AlertTriangle,
} from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

/* ── Types ── */

export type ErrorVariant = "connection" | "api" | "permission" | "timeout" | "generic";

export interface ErrorStateProps {
  /** Lucide icon to display above the title */
  icon?: LucideIcon;
  /** Main heading */
  title: string;
  /** Secondary explanation text */
  description: string;
  /** Primary call-to-action label */
  actionLabel?: string;
  /** Primary call-to-action handler */
  onAction?: () => void;
  /** Secondary action label */
  secondaryLabel?: string;
  /** Secondary action handler */
  onSecondary?: () => void;
  /** Error variant — controls default icon and icon-well color */
  variant?: ErrorVariant;
  /** Additional CSS classes */
  className?: string;
}

/* ── Variant configuration ── */

const VARIANT_CONFIG: Record<
  ErrorVariant,
  { icon: LucideIcon; well: string }
> = {
  connection: {
    icon: WifiOff,
    well: "bg-destructive/10 dark:bg-destructive/20 text-destructive",
  },
  api: {
    icon: AlertCircle,
    well: "bg-amber-500/10 dark:bg-amber-500/20 text-amber-600 dark:text-amber-400",
  },
  permission: {
    icon: ShieldX,
    well: "bg-amber-500/10 dark:bg-amber-500/20 text-amber-600 dark:text-amber-400",
  },
  timeout: {
    icon: Clock,
    well: "bg-muted/60 dark:bg-muted/30 text-muted-foreground",
  },
  generic: {
    icon: AlertTriangle,
    well: "bg-destructive/10 dark:bg-destructive/20 text-destructive",
  },
};

/* ── Fade-in animation ── */

const fadeIn = {
  hidden: { opacity: 0, y: 8 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.35, ease: "easeOut" },
  },
};

/* ── Core component ── */

/**
 * ErrorState -- Reusable error placeholder.
 *
 * Renders a centered area with a colored icon well, title,
 * description, and up to two action buttons. Includes a subtle
 * fade-in animation via Framer Motion.
 */
export function ErrorState({
  icon,
  title,
  description,
  actionLabel,
  onAction,
  secondaryLabel,
  onSecondary,
  variant = "generic",
  className,
}: ErrorStateProps) {
  const config = VARIANT_CONFIG[variant];
  const Icon = icon ?? config.icon;

  return (
    <motion.div
      variants={fadeIn}
      initial="hidden"
      animate="visible"
      role="alert"
      aria-label={title}
      className={cn(
        "flex flex-col items-center justify-center text-center p-8 gap-3",
        className,
      )}
    >
      {/* Icon well */}
      <div
        className={cn(
          "flex items-center justify-center rounded-full p-3",
          config.well,
        )}
        aria-hidden="true"
      >
        <Icon className="h-12 w-12" strokeWidth={1.5} />
      </div>

      {/* Text content */}
      <div className="flex flex-col gap-1">
        <h3 className="text-base font-semibold text-foreground">
          {title}
        </h3>
        <p className="max-w-sm text-sm text-muted-foreground">
          {description}
        </p>
      </div>

      {/* Actions */}
      {(actionLabel || secondaryLabel) && (
        <div className="mt-1 flex items-center gap-2">
          {actionLabel && onAction && (
            <Button variant="default" onClick={onAction}>
              {actionLabel}
            </Button>
          )}
          {secondaryLabel && onSecondary && (
            <Button variant="outline" onClick={onSecondary}>
              {secondaryLabel}
            </Button>
          )}
        </div>
      )}
    </motion.div>
  );
}

ErrorState.displayName = "ErrorState";

/* ═══════════════════════════════════════════════════════════════════════
   Pre-built error states
   ═══════════════════════════════════════════════════════════════════════ */

/** Connection lost / unreachable cluster */
export function ConnectionError({ onRetry }: { onRetry?: () => void }) {
  return (
    <ErrorState
      variant="connection"
      title="Connection failed"
      description="Unable to reach the cluster API server. Check your network connection and cluster status."
      actionLabel={onRetry ? "Retry" : undefined}
      onAction={onRetry}
    />
  );
}
ConnectionError.displayName = "ConnectionError";

/** API / server error */
export function ApiError({
  onRetry,
  message,
}: {
  onRetry?: () => void;
  message?: string;
}) {
  return (
    <ErrorState
      variant="api"
      title="Something went wrong"
      description={
        message ?? "An unexpected error occurred while fetching data. Please try again."
      }
      actionLabel={onRetry ? "Retry" : undefined}
      onAction={onRetry}
    />
  );
}
ApiError.displayName = "ApiError";

/** Insufficient permissions */
export function PermissionDenied({
  onRequestAccess,
}: {
  onRequestAccess?: () => void;
}) {
  return (
    <ErrorState
      variant="permission"
      title="Access denied"
      description="You don't have permission to view this resource. Contact your cluster administrator for access."
      actionLabel={onRequestAccess ? "Request Access" : undefined}
      onAction={onRequestAccess}
    />
  );
}
PermissionDenied.displayName = "PermissionDenied";

/** Request timed out */
export function TimeoutError({ onRetry }: { onRetry?: () => void }) {
  return (
    <ErrorState
      variant="timeout"
      title="Request timed out"
      description="The request took too long to complete. The cluster may be under heavy load."
      actionLabel={onRetry ? "Retry" : undefined}
      onAction={onRetry}
    />
  );
}
TimeoutError.displayName = "TimeoutError";
