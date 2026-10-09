/**
 * Tests for NamespaceFilter — covers a real usability bug reported in
 * production: with 20+ namespaces, the filter popover had no way to narrow
 * the list except scrolling. Also covers the Select All quick-action added
 * alongside the search box.
 *
 * The actual filter *persistence* across navigation (the other half of the
 * reported bug — selecting namespaces, visiting a pod, and coming back reset
 * the filter to "All Namespaces") lives in Pods.tsx's own URL-sync effect,
 * not in this component, and was verified live via a real browser session
 * rather than here (Pods.tsx has no existing unit-test harness to extend).
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, within } from '@testing-library/react';
import { NamespaceFilter } from './NamespaceFilter';

const MANY_NAMESPACES = [
  'default', 'kube-system', 'kube-public', 'kube-node-lease',
  'team-alpha', 'team-beta', 'team-gamma', 'team-delta', 'team-epsilon',
  'monitoring', 'logging', 'ingress-nginx', 'cert-manager', 'istio-system',
  'argo', 'argocd', 'vault', 'cattle-system', 'cattle-fleet-system', 'longhorn-system',
];

function openPopover() {
  fireEvent.click(screen.getByRole('button'));
}

describe('NamespaceFilter', () => {
  it('shows a search box and narrows the namespace list as the user types', () => {
    render(
      <NamespaceFilter
        namespaces={MANY_NAMESPACES}
        selected={new Set()}
        onSelectionChange={vi.fn()}
        triggerVariant="bar"
      />
    );
    openPopover();

    expect(screen.getByText('team-alpha')).toBeInTheDocument();
    expect(screen.getByText('istio-system')).toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText('Search namespaces...'), {
      target: { value: 'team-' },
    });

    expect(screen.getByText('team-alpha')).toBeInTheDocument();
    expect(screen.getByText('team-epsilon')).toBeInTheDocument();
    expect(screen.queryByText('istio-system')).not.toBeInTheDocument();
    expect(screen.queryByText('kube-system')).not.toBeInTheDocument();
  });

  it('shows an empty-state message when the search matches nothing', () => {
    render(
      <NamespaceFilter
        namespaces={MANY_NAMESPACES}
        selected={new Set()}
        onSelectionChange={vi.fn()}
        triggerVariant="bar"
      />
    );
    openPopover();

    fireEvent.change(screen.getByPlaceholderText('Search namespaces...'), {
      target: { value: 'zzz-does-not-exist' },
    });

    expect(screen.getByText(/no namespaces match/i)).toBeInTheDocument();
  });

  it('Select All selects every namespace, not just the currently-filtered subset', () => {
    const onSelectionChange = vi.fn();
    render(
      <NamespaceFilter
        namespaces={MANY_NAMESPACES}
        selected={new Set()}
        onSelectionChange={onSelectionChange}
        triggerVariant="bar"
      />
    );
    openPopover();

    fireEvent.click(screen.getByRole('button', { name: /select all/i }));

    expect(onSelectionChange).toHaveBeenCalledWith(new Set(MANY_NAMESPACES));
  });

  it('Clear resets the selection to empty (All Namespaces)', () => {
    const onSelectionChange = vi.fn();
    render(
      <NamespaceFilter
        namespaces={MANY_NAMESPACES}
        selected={new Set(['default', 'team-alpha'])}
        onSelectionChange={onSelectionChange}
        triggerVariant="bar"
      />
    );
    openPopover();

    fireEvent.click(screen.getByRole('button', { name: /^clear$/i }));

    expect(onSelectionChange).toHaveBeenCalledWith(new Set());
  });

  it('clicking a namespace label toggles it in the selection', () => {
    const onSelectionChange = vi.fn();
    render(
      <NamespaceFilter
        namespaces={['default', 'team-alpha']}
        selected={new Set(['default'])}
        onSelectionChange={onSelectionChange}
        triggerVariant="bar"
      />
    );
    openPopover();
    // Scoped to the popover panel: with exactly one namespace selected, the
    // trigger button's own label text ("default") would otherwise collide
    // with the list item of the same name.
    const panel = within(screen.getByRole('dialog'));

    fireEvent.click(panel.getByText('team-alpha'));
    expect(onSelectionChange).toHaveBeenCalledWith(new Set(['default', 'team-alpha']));

    fireEvent.click(panel.getByText('default'));
    expect(onSelectionChange).toHaveBeenCalledWith(new Set());
  });

  it('trigger label reflects selection count', () => {
    const { rerender } = render(
      <NamespaceFilter
        namespaces={MANY_NAMESPACES}
        selected={new Set()}
        onSelectionChange={vi.fn()}
        triggerVariant="bar"
      />
    );
    expect(screen.getByRole('button')).toHaveTextContent('All Namespaces');

    rerender(
      <NamespaceFilter
        namespaces={MANY_NAMESPACES}
        selected={new Set(['default', 'team-alpha'])}
        onSelectionChange={vi.fn()}
        triggerVariant="bar"
      />
    );
    expect(screen.getByRole('button')).toHaveTextContent('2 namespaces');
  });
});
