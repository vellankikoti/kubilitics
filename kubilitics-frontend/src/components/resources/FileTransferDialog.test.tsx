/**
 * Same production crash class as InlineFileBrowser: this component also
 * stored the whole `listContainerFiles` response object directly into
 * `entries` state instead of unwrapping `.entries`, crashing with
 * "entries.filter is not a function" the moment the dialog opened.
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';

vi.mock('@/services/backendApiClient', () => ({
  listContainerFiles: vi.fn(),
  getContainerFileDownloadUrl: vi.fn(() => 'http://backend/download'),
  uploadContainerFile: vi.fn(),
}));

vi.mock('@/stores/clusterPresenceStore', () => ({
  getActiveCluster: () => ({ id: 'c1' }),
  useActiveCluster: () => ({ id: 'c1' }),
}));

import { listContainerFiles } from '@/services/backendApiClient';
import { FileTransferDialog } from './FileTransferDialog';

describe('FileTransferDialog — response-shape crash (production bug)', () => {
  it('does not crash when listContainerFiles returns the real { entries, truncated, totalCount } shape', async () => {
    vi.mocked(listContainerFiles).mockResolvedValue({
      entries: [
        { name: 'app.log', type: 'file', size: 1024, modified: '2026-10-14T00:00:00Z' },
      ],
      truncated: false,
      totalCount: 1,
    });

    render(
      <FileTransferDialog
        open={true}
        onOpenChange={() => {}}
        podName="my-pod"
        namespace="default"
        baseUrl="http://backend"
        clusterId="c1"
        containers={[{ name: 'main' }]}
      />
    );

    await waitFor(() => expect(screen.getByText('app.log')).toBeInTheDocument());
  });

  it('shows a truncation notice when the backend reports a truncated listing', async () => {
    vi.mocked(listContainerFiles).mockResolvedValue({
      entries: [{ name: 'a.txt', type: 'file', size: 1, modified: '2026-10-14T00:00:00Z' }],
      truncated: true,
      totalCount: 9000,
    });

    render(
      <FileTransferDialog
        open={true}
        onOpenChange={() => {}}
        podName="my-pod"
        namespace="default"
        baseUrl="http://backend"
        clusterId="c1"
        containers={[{ name: 'main' }]}
      />
    );

    await waitFor(() => expect(screen.getByText(/9000/)).toBeInTheDocument());
  });
});
