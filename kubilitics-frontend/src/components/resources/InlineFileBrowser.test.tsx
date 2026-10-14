/**
 * Real production crash: opening a Pod's detail page renders InlineFileBrowser,
 * which stored the whole `listContainerFiles` response object directly into
 * `entries` state instead of unwrapping `.entries` — the backend's response
 * shape is `{ entries: ContainerFileEntry[], truncated, totalCount }`, not a
 * bare array. `entries.filter(...)` then crashed the page with "entries.filter
 * is not a function" (minified as "L.filter is not a function").
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';

vi.mock('@/services/backendApiClient', () => ({
  listContainerFiles: vi.fn(),
  getContainerFileDownloadUrl: vi.fn(() => 'http://backend/download'),
  uploadContainerFile: vi.fn(),
}));

vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: () => ({ isConfigured: true }),
  getEffectiveBackendBaseUrl: () => 'http://backend',
}));

vi.mock('@/stores/clusterPresenceStore', () => ({
  useActiveCluster: () => ({ id: 'c1' }),
}));

vi.mock('@/hooks/useActiveClusterId', () => ({
  useActiveClusterId: () => 'c1',
}));

import { listContainerFiles } from '@/services/backendApiClient';
import { InlineFileBrowser } from './InlineFileBrowser';

describe('InlineFileBrowser — response-shape crash (production bug)', () => {
  it('does not crash when listContainerFiles returns the real { entries, truncated, totalCount } shape', async () => {
    vi.mocked(listContainerFiles).mockResolvedValue({
      entries: [
        { name: 'app.log', type: 'file', size: 1024, modified: '2026-10-14T00:00:00Z' },
        { name: 'config', type: 'dir', size: 0, modified: '2026-10-14T00:00:00Z' },
      ],
      truncated: false,
      totalCount: 2,
    });

    render(
      <InlineFileBrowser podName="my-pod" namespace="default" containerName="main" clusterId="c1" baseUrl="http://backend" />
    );

    await waitFor(() => expect(screen.getByText('app.log')).toBeInTheDocument());
    expect(screen.getByText('config')).toBeInTheDocument();
  });

  it('shows a truncation notice when the backend reports a truncated listing', async () => {
    vi.mocked(listContainerFiles).mockResolvedValue({
      entries: [{ name: 'a.txt', type: 'file', size: 1, modified: '2026-10-14T00:00:00Z' }],
      truncated: true,
      totalCount: 5000,
    });

    render(
      <InlineFileBrowser podName="my-pod" namespace="default" containerName="main" clusterId="c1" baseUrl="http://backend" />
    );

    await waitFor(() => expect(screen.getByText(/5000/)).toBeInTheDocument());
  });
});
