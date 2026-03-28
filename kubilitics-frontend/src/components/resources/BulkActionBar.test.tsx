import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/vitest';
import { BulkActionBar } from './BulkActionBar';

// Framer-motion AnimatePresence needs a minimal mock so children render synchronously
vi.mock('framer-motion', () => ({
  AnimatePresence: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  motion: {
    div: ({ children, ...props }: React.HTMLAttributes<HTMLDivElement>) => (
      <div {...props}>{children}</div>
    ),
  },
}));

afterEach(() => {
  cleanup();
  // Remove any Radix portal content left in document.body
  document.body.innerHTML = '';
});

const baseProps = {
  selectedCount: 0,
  resourceName: 'pod',
  resourceType: 'pods' as const,
  onClearSelection: vi.fn(),
};

describe('BulkActionBar', () => {
  it('renders nothing when selectedCount is 0', () => {
    const { container } = render(<BulkActionBar {...baseProps} />);
    expect(container.querySelector('[role="toolbar"]')).not.toBeInTheDocument();
  });

  it('shows selected count text', () => {
    render(<BulkActionBar {...baseProps} selectedCount={3} />);
    const toolbar = screen.getByRole('toolbar');
    expect(within(toolbar).getByText('3')).toBeInTheDocument();
    expect(within(toolbar).getByText('pods selected')).toBeInTheDocument();
  });

  it('shows singular resource name when count is 1', () => {
    render(<BulkActionBar {...baseProps} selectedCount={1} />);
    const toolbar = screen.getByRole('toolbar');
    expect(within(toolbar).getByText('pod selected')).toBeInTheDocument();
  });

  it('shows Delete button when onBulkDelete is provided', () => {
    render(
      <BulkActionBar
        {...baseProps}
        selectedCount={2}
        onBulkDelete={vi.fn().mockResolvedValue([])}
      />,
    );
    const toolbar = screen.getByRole('toolbar');
    expect(within(toolbar).getByText('Delete')).toBeInTheDocument();
  });

  it('does not show Delete button when onBulkDelete is not provided', () => {
    render(<BulkActionBar {...baseProps} selectedCount={2} />);
    const toolbar = screen.getByRole('toolbar');
    expect(within(toolbar).queryByText('Delete')).not.toBeInTheDocument();
  });

  it('shows Restart button only for restartable resource types', () => {
    const restartHandler = vi.fn().mockResolvedValue([]);

    // pods are restartable
    render(
      <BulkActionBar
        {...baseProps}
        selectedCount={2}
        resourceType="pods"
        onBulkRestart={restartHandler}
      />,
    );
    let toolbar = screen.getByRole('toolbar');
    expect(within(toolbar).getByText('Restart')).toBeInTheDocument();

    cleanup();
    document.body.innerHTML = '';

    // services are NOT restartable
    render(
      <BulkActionBar
        {...baseProps}
        selectedCount={2}
        resourceType="services"
        onBulkRestart={restartHandler}
      />,
    );
    toolbar = screen.getByRole('toolbar');
    expect(within(toolbar).queryByText('Restart')).not.toBeInTheDocument();
  });

  it('shows Scale button only for scalable resource types', () => {
    const scaleHandler = vi.fn().mockResolvedValue([]);

    // deployments are scalable
    render(
      <BulkActionBar
        {...baseProps}
        selectedCount={2}
        resourceType="deployments"
        onBulkScale={scaleHandler}
      />,
    );
    let toolbar = screen.getByRole('toolbar');
    expect(within(toolbar).getByText('Scale')).toBeInTheDocument();

    cleanup();
    document.body.innerHTML = '';

    // pods are NOT scalable
    render(
      <BulkActionBar
        {...baseProps}
        selectedCount={2}
        resourceType="pods"
        onBulkScale={scaleHandler}
      />,
    );
    toolbar = screen.getByRole('toolbar');
    expect(within(toolbar).queryByText('Scale')).not.toBeInTheDocument();
  });

  it('shows Label button when onBulkLabel is provided', () => {
    render(
      <BulkActionBar
        {...baseProps}
        selectedCount={2}
        onBulkLabel={vi.fn().mockResolvedValue([])}
      />,
    );
    const toolbar = screen.getByRole('toolbar');
    expect(within(toolbar).getByText('Label')).toBeInTheDocument();
  });

  it('opens confirmation dialog when Delete button is clicked', async () => {
    const user = userEvent.setup();
    render(
      <BulkActionBar
        {...baseProps}
        selectedCount={2}
        onBulkDelete={vi.fn().mockResolvedValue([])}
      />,
    );

    const toolbar = screen.getByRole('toolbar');
    await user.click(within(toolbar).getByText('Delete'));

    // Dialog renders in a portal — query the whole document
    expect(screen.getByText(/Delete 2 pods\?/)).toBeInTheDocument();
    expect(screen.getByText(/This will permanently delete/)).toBeInTheDocument();
  });

  it('opens confirmation dialog when Restart button is clicked', async () => {
    const user = userEvent.setup();
    render(
      <BulkActionBar
        {...baseProps}
        selectedCount={1}
        resourceName="deployment"
        resourceType="deployments"
        onBulkRestart={vi.fn().mockResolvedValue([])}
      />,
    );

    const toolbar = screen.getByRole('toolbar');
    await user.click(within(toolbar).getByText('Restart'));

    expect(screen.getByText(/Restart 1 deployment\?/)).toBeInTheDocument();
  });
});
