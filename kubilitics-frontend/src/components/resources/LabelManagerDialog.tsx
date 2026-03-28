import { useState, useMemo, useCallback } from 'react';
import { X, Plus, Tag } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { cn } from '@/lib/utils';

// ── Types ────────────────────────────────────────────────────────────────

export interface LabelManagerDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Map of resource key ("namespace/name") to its current labels */
  selectedResourceLabels: Map<string, Record<string, string>>;
  /** Number of selected resources (for display) */
  selectedCount: number;
  /** Singular resource name for display (e.g. "pod", "deployment") */
  resourceName: string;
  /**
   * Called when user clicks "Apply Changes".
   * Receives a label patch: key -> value for adds/updates, key -> null for removals.
   */
  onApply: (labelPatch: Record<string, string | null>) => Promise<void>;
}

/** Regex for valid Kubernetes label keys (simplified — allows dns subdomain prefix) */
const K8S_LABEL_KEY_RE = /^([a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?\/)?[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$/;
const K8S_LABEL_VALUE_RE = /^([a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?)?$/;

interface LabelEntry {
  key: string;
  value: string;
  /** Present on all selected resources */
  onAll: boolean;
  /** Present on some but not all */
  partial: boolean;
  /** Count of resources that have this exact key=value */
  count: number;
}

// ── Component ────────────────────────────────────────────────────────────

export function LabelManagerDialog({
  open,
  onOpenChange,
  selectedResourceLabels,
  selectedCount,
  resourceName,
  onApply,
}: LabelManagerDialogProps) {
  const totalResources = selectedResourceLabels.size;

  // Compute the initial union of all labels from selected resources
  const initialLabels = useMemo(() => {
    const labelCounts = new Map<string, { value: string; count: number }>();

    for (const [, labels] of selectedResourceLabels) {
      for (const [k, v] of Object.entries(labels)) {
        const compositeKey = `${k}=${v}`;
        const existing = labelCounts.get(compositeKey);
        if (existing) {
          existing.count++;
        } else {
          labelCounts.set(compositeKey, { value: v, count: 1 });
        }
      }
    }

    // Also track which keys exist (regardless of value) for partial detection
    const keyPresence = new Map<string, number>();
    for (const [, labels] of selectedResourceLabels) {
      for (const k of Object.keys(labels)) {
        keyPresence.set(k, (keyPresence.get(k) ?? 0) + 1);
      }
    }

    const entries: LabelEntry[] = [];
    const seenKeys = new Set<string>();

    for (const [compositeKey, { value, count }] of labelCounts) {
      const key = compositeKey.substring(0, compositeKey.indexOf('='));
      const keyCount = keyPresence.get(key) ?? 0;
      seenKeys.add(key);
      entries.push({
        key,
        value,
        onAll: count === totalResources,
        partial: count < totalResources,
        count,
      });
    }

    // Sort: full labels first, then partial, then alphabetically
    entries.sort((a, b) => {
      if (a.onAll !== b.onAll) return a.onAll ? -1 : 1;
      return a.key.localeCompare(b.key);
    });

    return entries;
  }, [selectedResourceLabels, totalResources]);

  // Track current label state: starts with initial, user can add/remove
  const [currentLabels, setCurrentLabels] = useState<LabelEntry[]>([]);
  const [removedKeys, setRemovedKeys] = useState<Set<string>>(new Set());
  const [addedLabels, setAddedLabels] = useState<Map<string, string>>(new Map());

  // New label inputs
  const [newKey, setNewKey] = useState('');
  const [newValue, setNewValue] = useState('');
  const [keyError, setKeyError] = useState('');
  const [valueError, setValueError] = useState('');
  const [isApplying, setIsApplying] = useState(false);

  // Reset state when dialog opens
  const handleOpenChange = useCallback((nextOpen: boolean) => {
    if (nextOpen) {
      setCurrentLabels([...initialLabels]);
      setRemovedKeys(new Set());
      setAddedLabels(new Map());
      setNewKey('');
      setNewValue('');
      setKeyError('');
      setValueError('');
      setIsApplying(false);
    }
    onOpenChange(nextOpen);
  }, [initialLabels, onOpenChange]);

  // Remove a label (mark for deletion)
  const handleRemoveLabel = useCallback((key: string) => {
    setCurrentLabels((prev) => prev.filter((l) => l.key !== key));
    setAddedLabels((prev) => {
      const next = new Map(prev);
      next.delete(key);
      return next;
    });
    // Only mark as removed if it was an original label
    const wasOriginal = initialLabels.some((l) => l.key === key);
    if (wasOriginal) {
      setRemovedKeys((prev) => new Set([...prev, key]));
    }
  }, [initialLabels]);

  // Add a new label
  const handleAddLabel = useCallback(() => {
    const trimmedKey = newKey.trim();
    const trimmedValue = newValue.trim();

    // Validate key
    if (!trimmedKey) {
      setKeyError('Key is required');
      return;
    }
    if (!K8S_LABEL_KEY_RE.test(trimmedKey)) {
      setKeyError('Invalid label key');
      return;
    }
    if (trimmedKey.length > 253 + 1 + 63) {
      setKeyError('Key too long');
      return;
    }

    // Validate value
    if (trimmedValue && !K8S_LABEL_VALUE_RE.test(trimmedValue)) {
      setValueError('Invalid label value');
      return;
    }
    if (trimmedValue.length > 63) {
      setValueError('Value must be 63 chars or less');
      return;
    }

    // Check for duplicate key
    const existingIdx = currentLabels.findIndex((l) => l.key === trimmedKey);
    if (existingIdx >= 0) {
      // Update the existing label value
      setCurrentLabels((prev) => prev.map((l) =>
        l.key === trimmedKey
          ? { ...l, value: trimmedValue, onAll: true, partial: false, count: totalResources }
          : l,
      ));
    } else {
      // Add new label entry
      setCurrentLabels((prev) => [
        ...prev,
        { key: trimmedKey, value: trimmedValue, onAll: true, partial: false, count: totalResources },
      ]);
    }

    // Track the add
    setAddedLabels((prev) => new Map([...prev, [trimmedKey, trimmedValue]]));
    // If this key was previously removed, un-remove it
    setRemovedKeys((prev) => {
      const next = new Set(prev);
      next.delete(trimmedKey);
      return next;
    });

    setNewKey('');
    setNewValue('');
    setKeyError('');
    setValueError('');
  }, [newKey, newValue, currentLabels, totalResources]);

  // Compute the diff to apply
  const labelPatch = useMemo(() => {
    const patch: Record<string, string | null> = {};

    // Removals
    for (const key of removedKeys) {
      patch[key] = null;
    }

    // Additions / updates
    for (const [key, value] of addedLabels) {
      patch[key] = value;
    }

    return patch;
  }, [removedKeys, addedLabels]);

  const hasChanges = Object.keys(labelPatch).length > 0;

  const handleApply = useCallback(async () => {
    if (!hasChanges) return;
    setIsApplying(true);
    try {
      await onApply(labelPatch);
      onOpenChange(false);
    } catch {
      // Error handling is done by the parent (toast, etc.)
    } finally {
      setIsApplying(false);
    }
  }, [hasChanges, labelPatch, onApply, onOpenChange]);

  const plural = selectedCount === 1 ? resourceName : `${resourceName}s`;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Tag className="h-5 w-5" />
            Manage Labels
          </DialogTitle>
          <DialogDescription>
            Labels on {selectedCount} selected {plural}. Add or remove labels, then apply changes.
          </DialogDescription>
        </DialogHeader>

        {/* Existing labels */}
        <div className="space-y-2">
          {currentLabels.length === 0 && (
            <p className="text-sm text-muted-foreground py-3 text-center">
              No labels on selected resources
            </p>
          )}
          {currentLabels.length > 0 && (
            <div className="flex flex-wrap gap-1.5 max-h-48 overflow-y-auto py-1">
              {currentLabels.map((label) => (
                <span
                  key={label.key}
                  className={cn(
                    'inline-flex items-center gap-1 pl-2.5 pr-1 py-1 rounded-md text-xs font-medium border',
                    addedLabels.has(label.key)
                      ? 'bg-emerald-50 border-emerald-200 text-emerald-800 dark:bg-emerald-950 dark:border-emerald-800 dark:text-emerald-200'
                      : label.partial
                        ? 'bg-amber-50 border-amber-200 text-amber-800 dark:bg-amber-950 dark:border-amber-800 dark:text-amber-200'
                        : 'bg-muted border-border text-foreground',
                  )}
                >
                  <span className="font-semibold">{label.key}</span>
                  <span className="text-muted-foreground mx-0.5">:</span>
                  <span>{label.value || <span className="italic text-muted-foreground">(empty)</span>}</span>
                  {label.partial && !addedLabels.has(label.key) && (
                    <span className="text-[10px] text-amber-600 dark:text-amber-400 ml-0.5">
                      ({label.count}/{totalResources})
                    </span>
                  )}
                  <button
                    type="button"
                    onClick={() => handleRemoveLabel(label.key)}
                    className="ml-0.5 p-0.5 rounded hover:bg-foreground/10 transition-colors"
                    aria-label={`Remove label ${label.key}`}
                    disabled={isApplying}
                  >
                    <X className="h-3 w-3" />
                  </button>
                </span>
              ))}
            </div>
          )}
        </div>

        {/* Add new label */}
        <div className="border-t border-border pt-3 space-y-2">
          <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Add Label</p>
          <div className="flex gap-2">
            <div className="flex-1 space-y-1">
              <Input
                value={newKey}
                onChange={(e) => { setNewKey(e.target.value); setKeyError(''); }}
                placeholder="key"
                className={cn('h-8 text-sm', keyError && 'border-destructive')}
                disabled={isApplying}
                onKeyDown={(e) => { if (e.key === 'Enter') handleAddLabel(); }}
              />
              {keyError && <p className="text-xs text-destructive">{keyError}</p>}
            </div>
            <div className="flex-1 space-y-1">
              <Input
                value={newValue}
                onChange={(e) => { setNewValue(e.target.value); setValueError(''); }}
                placeholder="value"
                className={cn('h-8 text-sm', valueError && 'border-destructive')}
                disabled={isApplying}
                onKeyDown={(e) => { if (e.key === 'Enter') handleAddLabel(); }}
              />
              {valueError && <p className="text-xs text-destructive">{valueError}</p>}
            </div>
            <Button
              variant="outline"
              size="sm"
              className="h-8 gap-1"
              onClick={handleAddLabel}
              disabled={isApplying || !newKey.trim()}
            >
              <Plus className="h-3.5 w-3.5" />
              Add
            </Button>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center justify-between pt-2 border-t border-border">
          <div className="text-xs text-muted-foreground">
            {hasChanges ? (
              <>
                {Object.values(labelPatch).filter((v) => v !== null).length > 0 && (
                  <span className="text-emerald-600 dark:text-emerald-400">
                    +{Object.values(labelPatch).filter((v) => v !== null).length} add
                  </span>
                )}
                {Object.values(labelPatch).filter((v) => v !== null).length > 0 &&
                  Object.values(labelPatch).filter((v) => v === null).length > 0 && ', '}
                {Object.values(labelPatch).filter((v) => v === null).length > 0 && (
                  <span className="text-destructive">
                    -{Object.values(labelPatch).filter((v) => v === null).length} remove
                  </span>
                )}
              </>
            ) : (
              'No changes'
            )}
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => onOpenChange(false)}
              disabled={isApplying}
            >
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={handleApply}
              disabled={!hasChanges || isApplying}
            >
              {isApplying ? 'Applying...' : 'Apply Changes'}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
