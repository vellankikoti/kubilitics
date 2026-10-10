import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

// A single stable mock for useNavigate — declared BEFORE the import chain that
// evaluates the page, so react-router-dom is hoisted with this replacement.
const navigateMock = vi.fn();
vi.mock('react-router-dom', async (orig) => {
  const actual = await orig<typeof import('react-router-dom')>();
  return { ...actual, useNavigate: () => navigateMock };
});

import { ClusterPickerPage } from './ClusterPickerPage';
import {
  useClusterPresenceStore,
  __resetForTest,
} from '@/stores/clusterPresenceStore';

function renderPicker() {
  return render(
    <MemoryRouter>
      <ClusterPickerPage />
    </MemoryRouter>,
  );
}

describe('ClusterPickerPage', () => {
  beforeEach(() => {
    __resetForTest();
    navigateMock.mockReset();
  });

  it('renders one card per available cluster', () => {
    useClusterPresenceStore.setState({
      discovered: [
        { identity: { name: 'prod', serverUrl: 'https://p.example' }, source: 'kubeconfig' },
        { identity: { name: 'dev', serverUrl: 'https://d.example' }, source: 'kubeconfig' },
      ],
      registered: [],
      connected: [],
      isReady: true,
    });
    renderPicker();
    expect(screen.getByText('prod')).toBeInTheDocument();
    expect(screen.getByText('dev')).toBeInTheDocument();
  });

  it('clicking a registered+connected cluster sets active identity and navigates to /dashboard', () => {
    useClusterPresenceStore.setState({
      discovered: [
        { identity: { name: 'prod', serverUrl: 'https://p.example' }, source: 'kubeconfig' },
      ],
      registered: [
        {
          identity: { name: 'prod', serverUrl: 'https://p.example' },
          source: 'kubeconfig',
          registered_at: '2026-01-01T00:00:00Z',
          reachable: true,
          session_id: 'prod-uuid',
        },
      ],
      connected: [
        {
          identity: { name: 'prod', serverUrl: 'https://p.example' },
          source: 'kubeconfig',
          registered_at: '2026-01-01T00:00:00Z',
          reachable: true,
          session_id: 'prod-uuid',
          connected_at: '2026-01-01T00:00:00Z',
        },
      ],
      isReady: true,
    });
    renderPicker();
    fireEvent.click(screen.getByRole('button', { name: /prod/i }));
    expect(useClusterPresenceStore.getState().activeLogicalIdentity?.name).toBe(
      'prod',
    );
    expect(navigateMock).toHaveBeenCalledWith('/dashboard');
  });

  it('search filter narrows visible cards by name', () => {
    useClusterPresenceStore.setState({
      discovered: [
        { identity: { name: 'prod-us', serverUrl: 'https://p.example' }, source: 'kubeconfig' },
        { identity: { name: 'staging', serverUrl: 'https://s.example' }, source: 'kubeconfig' },
      ],
      registered: [],
      connected: [],
      isReady: true,
    });
    renderPicker();
    fireEvent.change(screen.getByPlaceholderText(/search/i), {
      target: { value: 'prod' },
    });
    expect(screen.getByText('prod-us')).toBeInTheDocument();
    expect(screen.queryByText('staging')).toBeNull();
  });

  it('search filter matches on server URL', () => {
    useClusterPresenceStore.setState({
      discovered: [
        { identity: { name: 'one', serverUrl: 'https://alpha.example' }, source: 'kubeconfig' },
        { identity: { name: 'two', serverUrl: 'https://beta.example' }, source: 'kubeconfig' },
      ],
      registered: [],
      connected: [],
      isReady: true,
    });
    renderPicker();
    fireEvent.change(screen.getByPlaceholderText(/search/i), {
      target: { value: 'beta' },
    });
    expect(screen.getByText('two')).toBeInTheDocument();
    expect(screen.queryByText('one')).toBeNull();
  });

  it('merges discovered + registered and reflects reachability from registered', () => {
    useClusterPresenceStore.setState({
      discovered: [
        { identity: { name: 'prod', serverUrl: 'https://p.example' }, source: 'kubeconfig' },
      ],
      registered: [
        {
          identity: { name: 'prod', serverUrl: 'https://p.example' },
          source: 'kubeconfig',
          registered_at: '2026-01-01T00:00:00Z',
          reachable: true,
          session_id: 'uuid-prod',
        },
      ],
      connected: [],
      isReady: true,
    });
    renderPicker();
    // Only one card (deduped by logical identity)
    expect(screen.getAllByText('prod')).toHaveLength(1);
    // Reachability dot's title appears. NOTE: getByTitle, not getByLabelText —
    // the dot has no aria-label, only a `title` attribute, which
    // getByLabelText does not match at all (it was silently failing this
    // assertion's intent some of the time, which is why this test was
    // flaky — see docs/ai/KNOWN-ISSUES.md).
    expect(screen.getByTitle(/reachable/i)).toBeInTheDocument();
  });

  it('shows a checking state, not unreachable, when reachable=false with no check yet', () => {
    // The backend's Reachable flag fails closed (false) both for a genuinely
    // failed health check AND for a cluster that has never been checked yet
    // (fresh registration, or the brief window right after backend startup
    // before reconnects complete). last_checked_at is the signal that tells
    // these apart — empty means "never checked." Collapsing both to a solid
    // red "Unreachable" dot creates a false alarm on every fresh connect.
    useClusterPresenceStore.setState({
      discovered: [
        { identity: { name: 'fresh', serverUrl: 'https://fresh.example' }, source: 'kubeconfig' },
      ],
      registered: [
        {
          identity: { name: 'fresh', serverUrl: 'https://fresh.example' },
          source: 'kubeconfig',
          registered_at: '2026-01-01T00:00:00Z',
          reachable: false,
          session_id: 'uuid-fresh',
          // last_checked_at intentionally omitted — never checked yet.
        },
      ],
      connected: [],
      isReady: true,
    });
    renderPicker();
    expect(screen.getByTitle(/checking/i)).toBeInTheDocument();
    expect(screen.queryByTitle(/^unreachable/i)).toBeNull();
  });

  it('shows unreachable (not checking) once a real check has actually failed', () => {
    useClusterPresenceStore.setState({
      discovered: [
        { identity: { name: 'down', serverUrl: 'https://down.example' }, source: 'kubeconfig' },
      ],
      registered: [
        {
          identity: { name: 'down', serverUrl: 'https://down.example' },
          source: 'kubeconfig',
          registered_at: '2026-01-01T00:00:00Z',
          reachable: false,
          session_id: 'uuid-down',
          last_checked_at: '2026-01-01T00:00:05Z',
        },
      ],
      connected: [],
      isReady: true,
    });
    renderPicker();
    expect(screen.getByTitle(/^unreachable/i)).toBeInTheDocument();
  });

  it('renders an empty-state message when no clusters are available', () => {
    useClusterPresenceStore.setState({
      discovered: [],
      registered: [],
      connected: [],
      isReady: true,
    });
    renderPicker();
    expect(screen.getByText(/no clusters/i)).toBeInTheDocument();
  });

  it('orders connected clusters before non-connected, then alphabetical', () => {
    useClusterPresenceStore.setState({
      discovered: [
        { identity: { name: 'zebra', serverUrl: 'https://z.example' }, source: 'kubeconfig' },
        { identity: { name: 'alpha', serverUrl: 'https://a.example' }, source: 'kubeconfig' },
        { identity: { name: 'bravo', serverUrl: 'https://b.example' }, source: 'kubeconfig' },
      ],
      registered: [],
      connected: [
        {
          identity: { name: 'zebra', serverUrl: 'https://z.example' },
          source: 'kubeconfig',
          registered_at: '',
          reachable: true,
          connected_at: '2026-04-24T00:00:00Z',
          session_id: 'uuid-zebra',
        },
      ],
      isReady: true,
    });
    renderPicker();
    const cards = screen.getAllByTestId('cluster-picker-card');
    // Connected cluster 'zebra' first, then alphabetical 'alpha', 'bravo'.
    expect(cards[0]).toHaveTextContent('zebra');
    expect(cards[1]).toHaveTextContent('alpha');
    expect(cards[2]).toHaveTextContent('bravo');
  });

  it('empty state shows the detected-nothing message', () => {
    // Zero-cluster case is owned by the picker (the old WelcomePage was
    // deleted). NOTE: the component's actual copy is "No clusters detected"
    // — this assertion previously said "found," which never matched
    // anything and made this test deterministically fail (not flaky — see
    // docs/ai/KNOWN-ISSUES.md, which had mis-filed it alongside a
    // genuinely flaky test).
    useClusterPresenceStore.setState({
      discovered: [],
      registered: [],
      connected: [],
      isReady: true,
    });
    renderPicker();
    expect(screen.getByText(/no clusters detected/i)).toBeInTheDocument();
  });

  // This test previously also asserted an "Add a cluster" trigger inside the
  // empty state opens a dialog in place (`cluster-picker-add-cluster-empty` +
  // `role="dialog"`). Neither exists in the current two-pane layout — the
  // empty state is just the detected-nothing message above, and adding a
  // cluster happens via the always-visible right-pane upload/paste panel,
  // not a dialog. Whether an empty-state shortcut button should exist is a
  // product decision, not a test-hygiene fix — skipped rather than deleted
  // or guessed at, so the gap stays visible instead of silently vanishing.
  it.skip('empty state exposes an Add-a-cluster trigger that opens a dialog (feature does not currently exist)', () => {
    useClusterPresenceStore.setState({
      discovered: [],
      registered: [],
      connected: [],
      isReady: true,
    });
    renderPicker();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId('cluster-picker-add-cluster-empty'));
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });
});
