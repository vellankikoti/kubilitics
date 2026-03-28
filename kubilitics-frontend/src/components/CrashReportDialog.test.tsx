import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/vitest';

// ── Mocks ────────────────────────────────────────────────────────────────────

vi.mock('@/lib/errorTracker', () => ({
  ErrorTracker: {
    hasRemoteEndpoint: () => false,
    buildCrashReport: (error: Error) => ({
      appVersion: '0.4.0',
      platform: 'test',
      userAgent: 'vitest',
      url: 'http://localhost',
      timestamp: new Date().toISOString(),
      triggeringError: {
        name: error.name,
        message: error.message,
        stack: error.stack,
      },
      recentErrors: [],
    }),
    submitCrashReport: vi.fn().mockResolvedValue(undefined),
  },
}));

import { CrashReportDialog } from './CrashReportDialog';

afterEach(() => {
  cleanup();
  document.body.innerHTML = '';
});

// ── Tests ────────────────────────────────────────────────────────────────────

describe('CrashReportDialog', () => {
  const testError = new Error('Something broke badly');
  testError.name = 'TestError';

  it('shows the error message', () => {
    render(<CrashReportDialog error={testError} errorId="err-123" />);
    const dialog = screen.getByRole('dialog');
    expect(within(dialog).getByText('Something broke badly')).toBeInTheDocument();
  });

  it('shows "Something went wrong" title', () => {
    render(<CrashReportDialog error={testError} errorId="err-123" />);
    const dialog = screen.getByRole('dialog');
    expect(within(dialog).getByText('Something went wrong')).toBeInTheDocument();
  });

  it('shows error ID when provided', () => {
    render(<CrashReportDialog error={testError} errorId="err-123" />);
    const dialog = screen.getByRole('dialog');
    expect(within(dialog).getByText('err-123')).toBeInTheDocument();
  });

  it('shows "Copy to Clipboard" button when no remote endpoint', () => {
    render(<CrashReportDialog error={testError} errorId={null} />);
    const dialog = screen.getByRole('dialog');
    expect(
      within(dialog).getByRole('button', { name: /copy to clipboard/i }),
    ).toBeInTheDocument();
  });

  it('shows "Restart App" button', () => {
    render(<CrashReportDialog error={testError} errorId={null} />);
    const dialog = screen.getByRole('dialog');
    expect(
      within(dialog).getByRole('button', { name: /restart app/i }),
    ).toBeInTheDocument();
  });

  it('shows collapsible technical details', async () => {
    const user = userEvent.setup();
    render(<CrashReportDialog error={testError} errorId={null} />);
    const dialog = screen.getByRole('dialog');

    // Technical Details trigger should be visible
    const trigger = within(dialog).getByText('Technical Details');
    expect(trigger).toBeInTheDocument();

    // Click to expand
    await user.click(trigger);

    // After expanding, the error name and message should be visible in the bold title
    const errorTitle = within(dialog).getAllByText(/TestError: Something broke badly/);
    expect(errorTitle.length).toBeGreaterThanOrEqual(1);
    expect(within(dialog).getByText(/App Version:/)).toBeInTheDocument();
  });

  it('shows the dialog as modal (always open)', () => {
    render(<CrashReportDialog error={testError} errorId={null} />);
    const dialog = screen.getByRole('dialog');
    expect(dialog).toBeInTheDocument();
  });
});
