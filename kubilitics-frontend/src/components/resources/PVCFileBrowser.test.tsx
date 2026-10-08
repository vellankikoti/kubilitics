/**
 * Tests for PVCFileBrowser — covers reported PVC "browse files" bugs:
 *  - an unbounded directory listing (file_transfer.go now caps and reports
 *    truncation) must surface that truncation to the user, not silently
 *    show an incomplete listing as complete
 *  - rapid directory navigation could let a slow, stale response overwrite
 *    state after a newer navigation had already completed
 *  - toggleExpand's error toast discarded the real error message
 * None of these had any test coverage before this change.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { PVCFileBrowser } from './PVCFileBrowser';
import { toast } from '@/components/ui/sonner';

let mockListContainerFiles: ReturnType<typeof vi.fn>;

vi.mock('@/services/backendApiClient', () => ({
  listContainerFiles: (...args: unknown[]) => mockListContainerFiles(...args),
  getContainerFileDownloadUrl: () => 'http://localhost/download',
  uploadContainerFile: vi.fn(),
}));

vi.mock('@/components/ui/sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
}));

function entry(name: string, type: 'file' | 'dir' = 'file') {
  return { name, type, size: 100, modified: '2026-01-01T00:00:00Z' };
}

const baseProps = {
  podName: 'my-pod',
  namespace: 'default',
  containerName: 'main',
  mountPath: '/data',
  baseUrl: 'http://localhost:8190',
  clusterId: 'c1',
};

describe('PVCFileBrowser', () => {
  beforeEach(() => {
    mockListContainerFiles = vi.fn();
    vi.clearAllMocks();
  });

  it('shows a truncation banner when the backend reports more entries than it returned', async () => {
    mockListContainerFiles.mockResolvedValue({
      entries: [entry('a.txt'), entry('b.txt')],
      truncated: true,
      totalCount: 5000,
    });

    render(<PVCFileBrowser {...baseProps} />);

    await waitFor(() => expect(screen.getByText(/truncated/i)).toBeTruthy());
    expect(screen.getByText(/showing first 2 of 5000 entries/i)).toBeTruthy();
  });

  it('does not show a truncation banner when everything fit', async () => {
    mockListContainerFiles.mockResolvedValue({
      entries: [entry('a.txt')],
      truncated: false,
      totalCount: 1,
    });

    render(<PVCFileBrowser {...baseProps} />);

    await waitFor(() => expect(screen.getByText('a.txt')).toBeTruthy());
    expect(screen.queryByText(/truncated/i)).toBeNull();
  });

  it('CRITICAL FIX: a slow response from an earlier navigation does not overwrite a newer one (race guard)', async () => {
    let resolveFirst!: (v: unknown) => void;
    mockListContainerFiles
      .mockImplementationOnce(() => new Promise((resolve) => { resolveFirst = resolve; }))
      .mockResolvedValueOnce({ entries: [entry('second-dir-file.txt')], truncated: false, totalCount: 1 });

    const { rerender } = render(<PVCFileBrowser {...baseProps} />);

    // Second navigation (e.g. the user clicked through to another dir)
    // completes BEFORE the first (slow) one resolves.
    rerender(<PVCFileBrowser {...baseProps} mountPath="/data/other" />);
    await waitFor(() => expect(screen.getByText('second-dir-file.txt')).toBeTruthy());

    // Now let the stale first request resolve — it must NOT overwrite the
    // listing that's already showing the newer directory's contents.
    resolveFirst({ entries: [entry('stale-first-dir-file.txt')], truncated: false, totalCount: 1 });
    await Promise.resolve();

    expect(screen.queryByText('stale-first-dir-file.txt')).toBeNull();
    expect(screen.getByText('second-dir-file.txt')).toBeTruthy();
  });

  it('surfaces the real error message when expanding a tree directory fails', async () => {
    mockListContainerFiles
      .mockResolvedValueOnce({ entries: [entry('subdir', 'dir')], truncated: false, totalCount: 1 })
      .mockRejectedValueOnce(new Error('permission denied'));

    render(<PVCFileBrowser {...baseProps} />);
    await waitFor(() => expect(screen.getByText('subdir')).toBeTruthy());

    fireEvent.click(screen.getByText('subdir'));

    await waitFor(() => expect(toast.error).toHaveBeenCalled());
    const [message] = (toast.error as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(message).toContain('permission denied');
  });
});
