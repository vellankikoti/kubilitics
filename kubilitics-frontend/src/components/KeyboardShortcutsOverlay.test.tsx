import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup, within } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import { KeyboardShortcutsOverlay } from './KeyboardShortcutsOverlay';

afterEach(() => {
  cleanup();
  document.body.innerHTML = '';
});

describe('KeyboardShortcutsOverlay', () => {
  const onClose = vi.fn();

  it('returns null when not visible', () => {
    const { container } = render(
      <KeyboardShortcutsOverlay visible={false} onClose={onClose} />,
    );
    expect(container.innerHTML).toBe('');
  });

  it('renders shortcut list when visible', () => {
    render(<KeyboardShortcutsOverlay visible={true} onClose={onClose} />);
    const dialog = screen.getByRole('dialog');

    // Section headings
    expect(within(dialog).getByText('Navigation')).toBeInTheDocument();
    expect(within(dialog).getByText('Topology')).toBeInTheDocument();
    expect(within(dialog).getByText('General')).toBeInTheDocument();

    // Some shortcut descriptions
    expect(within(dialog).getByText('Open search')).toBeInTheDocument();
    expect(within(dialog).getByText('Go to Dashboard')).toBeInTheDocument();
    expect(within(dialog).getByText('Fit to screen')).toBeInTheDocument();
    expect(within(dialog).getByText('Show this dialog')).toBeInTheDocument();
  });

  it('renders the dialog with correct aria attributes', () => {
    render(<KeyboardShortcutsOverlay visible={true} onClose={onClose} />);
    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveAttribute('aria-modal', 'true');
    expect(dialog).toHaveAttribute('aria-label', 'Keyboard shortcuts');
  });

  it('shows the Keyboard Shortcuts heading', () => {
    render(<KeyboardShortcutsOverlay visible={true} onClose={onClose} />);
    const dialog = screen.getByRole('dialog');
    expect(within(dialog).getByText('Keyboard Shortcuts')).toBeInTheDocument();
  });

  it('shows close button', () => {
    render(<KeyboardShortcutsOverlay visible={true} onClose={onClose} />);
    const dialog = screen.getByRole('dialog');
    expect(within(dialog).getByLabelText('Close shortcuts dialog')).toBeInTheDocument();
  });

  it('shows platform-aware modifier keys', () => {
    render(<KeyboardShortcutsOverlay visible={true} onClose={onClose} />);
    const dialog = screen.getByRole('dialog');

    // In jsdom, navigator.userAgent does not contain "Mac" so we expect "Ctrl"
    // The key string "Ctrl+K" is not split (no spaces around +), so the full
    // text appears inside a single <kbd> element.
    const ctrlKbd = within(dialog).getByText('Ctrl+K');
    expect(ctrlKbd).toBeInTheDocument();
    expect(ctrlKbd.tagName).toBe('KBD');
  });

  it('shows Esc hint in footer', () => {
    render(<KeyboardShortcutsOverlay visible={true} onClose={onClose} />);
    const dialog = screen.getByRole('dialog');
    // "Esc" appears in both the shortcuts list and the footer — check the footer text
    expect(within(dialog).getByText(/to close/)).toBeInTheDocument();
    const escElements = within(dialog).getAllByText('Esc');
    expect(escElements.length).toBeGreaterThanOrEqual(2);
  });
});
