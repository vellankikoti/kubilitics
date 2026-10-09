/**
 * Shared namespace-filter state for list pages — multi-select (`Set<string>`,
 * empty = "All Namespaces"), persisted to the URL (`?ns=a,b,c`) so navigating
 * away (e.g. opening a resource's detail page) and back restores the exact
 * same filtered view instead of silently resetting to "show everything."
 *
 * Also seeds from the legacy singular `?namespace=<ns>` param as a one-time
 * fallback — many pages (Namespace detail, RelatedResourcesPanel,
 * breadcrumbs, dashboard widgets) link into list pages with that param, and
 * this keeps those links working without every linker needing to change.
 * Once seeded, the filter is managed via `ns` and `namespace` is dropped
 * from the URL.
 *
 * Pair with the <NamespaceFilter> component (triggerVariant="bar") for the
 * standard list-page namespace-filter UI: search box, checkboxes, Select
 * All / Only User / Only System / Clear.
 */
import { useCallback, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

export function useNamespaceFilter(): [Set<string>, (next: Set<string>) => void] {
  const [searchParams, setSearchParams] = useSearchParams();

  const [selected, setSelectedState] = useState<Set<string>>(() => {
    const nsParam = searchParams.get('ns');
    if (nsParam) return new Set(nsParam.split(',').filter(Boolean));
    const legacyNamespace = searchParams.get('namespace');
    if (legacyNamespace) return new Set([legacyNamespace]);
    return new Set();
  });

  const setSelected = useCallback(
    (next: Set<string>) => {
      setSelectedState(next);
      setSearchParams(
        (prev) => {
          const nextParams = new URLSearchParams(prev);
          if (next.size > 0) nextParams.set('ns', Array.from(next).join(','));
          else nextParams.delete('ns');
          // Legacy singular param is only a one-time seed (consumed into
          // initial state above) — drop it once we're managing the filter
          // ourselves, so the URL doesn't carry both.
          nextParams.delete('namespace');
          return nextParams;
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  return [selected, setSelected];
}
