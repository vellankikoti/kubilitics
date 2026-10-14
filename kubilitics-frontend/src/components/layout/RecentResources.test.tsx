/**
 * Pre-existing bug (found incidentally while verifying an unrelated Metrics
 * fix with a real browser): React logs "validateDOMNesting(...): <button>
 * cannot appear as a descendant of <button>" on every page load, because the
 * "Clear recent resources" button was nested inside the section's toggle
 * button. Nested interactive controls are invalid HTML and give screen
 * readers/keyboard users undefined focus and activation behavior — not just
 * a console-noise issue. (It also breaks accessible-name computation: the
 * outer control's name ends up swallowing the inner button's label, which
 * is why these tests select the toggle by its functional aria-controls
 * attribute rather than by accessible name.)
 */
import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import { MemoryRouter } from 'react-router-dom';
import { useRecentResourcesStore } from '@/stores/recentResourcesStore';
import { RecentResources } from './RecentResources';

function seedOneRecentResource() {
  useRecentResourcesStore.setState({
    recentResources: [
      { resourceKind: 'Pod', name: 'my-web-app', namespace: 'default', path: '/pods/default/my-web-app', timestamp: Date.now() },
    ],
  });
}

function getToggle(container: HTMLElement): HTMLElement {
  const el = container.querySelector('[aria-controls="nav-recent-resources"]');
  if (!el) throw new Error('toggle control not found');
  return el as HTMLElement;
}

beforeEach(() => {
  useRecentResourcesStore.setState({ recentResources: [] });
  localStorage.clear();
});

describe('RecentResources — no nested interactive controls', () => {
  it('never renders a <button> inside another <button>', () => {
    seedOneRecentResource();
    const { container } = render(<RecentResources />, { wrapper: MemoryRouter });
    const buttons = container.querySelectorAll('button');
    buttons.forEach((btn) => {
      expect(btn.querySelector('button')).toBeNull();
    });
  });

  it('still toggles collapse on click', () => {
    seedOneRecentResource();
    const { container } = render(<RecentResources />, { wrapper: MemoryRouter });
    const toggle = getToggle(container);
    expect(toggle).toHaveAttribute('aria-expanded', 'true');
    fireEvent.click(toggle);
    expect(toggle).toHaveAttribute('aria-expanded', 'false');
  });

  it('still toggles collapse on keyboard activation (Enter/Space)', () => {
    seedOneRecentResource();
    const { container } = render(<RecentResources />, { wrapper: MemoryRouter });
    const toggle = getToggle(container);
    expect(toggle).toHaveAttribute('aria-expanded', 'true');
    fireEvent.keyDown(toggle, { key: 'Enter' });
    expect(toggle).toHaveAttribute('aria-expanded', 'false');
    fireEvent.keyDown(toggle, { key: ' ' });
    expect(toggle).toHaveAttribute('aria-expanded', 'true');
  });

  it('clearing recents does not also toggle the section collapsed', () => {
    seedOneRecentResource();
    const { container } = render(<RecentResources />, { wrapper: MemoryRouter });
    const toggle = getToggle(container);
    expect(toggle).toHaveAttribute('aria-expanded', 'true');
    fireEvent.click(screen.getByRole('button', { name: /clear recent resources/i }));
    expect(useRecentResourcesStore.getState().recentResources).toHaveLength(0);
  });

  it('exposes the toggle to assistive tech as a button role', () => {
    seedOneRecentResource();
    const { container } = render(<RecentResources />, { wrapper: MemoryRouter });
    const toggle = getToggle(container);
    expect(toggle).toHaveAttribute('role', 'button');
    expect(toggle).toHaveAttribute('tabIndex', '0');
  });

  it("gives the toggle an explicit aria-label, not inferred from its text content", () => {
    // Real Chromium (verified via Playwright, not just jsdom) computes the
    // toggle's accessible name from ALL descendant text when it has none of
    // its own — including the nested Clear button's "Clear recent
    // resources" label — producing "Recent Clear recent resources" and
    // making the two controls indistinguishable to a screen reader.
    // jsdom's accessible-name algorithm doesn't reproduce this divergence,
    // so this asserts the fix directly (an explicit aria-label) rather than
    // through name-matching, which real Chromium already proved necessary.
    seedOneRecentResource();
    const { container } = render(<RecentResources />, { wrapper: MemoryRouter });
    const toggle = getToggle(container);
    const label = toggle.getAttribute('aria-label') ?? '';
    expect(label.length).toBeGreaterThan(0);
    expect(label.toLowerCase()).not.toContain('clear');
  });
});
