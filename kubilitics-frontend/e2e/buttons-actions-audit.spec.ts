/**
 * E2E: Buttons & Actions Audit
 * Systematically navigates to every major page and verifies that buttons
 * and interactive elements do not crash when clicked.
 */
import { test, expect, Page } from '@playwright/test';

// ─── Helpers ──────────────────────────────────────────────────────────────────

/** Wait for the page to settle after navigation (lazy-load + hydration). */
async function waitForPageReady(page: Page) {
  await page.waitForLoadState('load');
  // Allow lazy chunks to load and React to hydrate
  await page.waitForTimeout(800);
}

/** Collect console errors during a callback. */
async function collectConsoleErrors(page: Page, fn: () => Promise<void>): Promise<string[]> {
  const errors: string[] = [];
  const handler = (msg: import('@playwright/test').ConsoleMessage) => {
    if (msg.type() === 'error') errors.push(msg.text());
  };
  page.on('console', handler);
  await fn();
  page.off('console', handler);
  return errors;
}

// ─── Major pages to audit ─────────────────────────────────────────────────────

const MAJOR_PAGES = [
  { name: 'Dashboard', path: '/dashboard' },
  { name: 'Fleet', path: '/fleet' },
  { name: 'Topology', path: '/topology' },
  { name: 'Templates', path: '/templates' },
  { name: 'Pods list', path: '/pods' },
  { name: 'Deployments list', path: '/deployments' },
  { name: 'Services list', path: '/services' },
  { name: 'Settings', path: '/settings' },
  { name: 'Workloads Overview', path: '/workloads' },
  { name: 'Networking Overview', path: '/networking' },
  { name: 'Storage Overview', path: '/storage' },
  { name: 'Cluster Overview', path: '/cluster' },
  { name: 'Nodes list', path: '/nodes' },
  { name: 'Namespaces list', path: '/namespaces' },
  { name: 'ConfigMaps list', path: '/configmaps' },
  { name: 'Secrets list', path: '/secrets' },
  { name: 'Ingresses list', path: '/ingresses' },
  { name: 'CronJobs list', path: '/cronjobs' },
  { name: 'StatefulSets list', path: '/statefulsets' },
  { name: 'DaemonSets list', path: '/daemonsets' },
];

// ─── Tests ────────────────────────────────────────────────────────────────────

test.describe('Buttons & Actions Audit', () => {
  test.describe.configure({ timeout: 60_000 });

  for (const { name, path } of MAJOR_PAGES) {
    test(`${name} — page loads without crash`, async ({ page }) => {
      await page.goto(path);
      await waitForPageReady(page);

      // Page should NOT show the error boundary or NotFound
      const body = page.locator('body');
      await expect(body).not.toContainText('Something went wrong');
      await expect(body).not.toContainText('Unhandled Runtime Error');

      // Title should still be Kubilitics (not a blank/error page)
      await expect(page).toHaveTitle(/Kubilitics/i);
    });

    test(`${name} — click all visible buttons without crash`, async ({ page }) => {
      await page.goto(path);
      await waitForPageReady(page);

      // Gather all visible buttons (skip hidden, disabled, or dropdown triggers that open menus)
      const buttons = page.locator(
        'button:visible:not([disabled]):not([aria-hidden="true"])'
      );
      const count = await buttons.count();

      // Click each button and verify no crash
      for (let i = 0; i < Math.min(count, 30); i++) {
        const btn = buttons.nth(i);
        const label = await btn.textContent().catch(() => '');
        const isVisible = await btn.isVisible().catch(() => false);
        if (!isVisible) continue;

        // Skip buttons that would navigate away (links disguised as buttons)
        const tagName = await btn.evaluate(el => el.tagName);
        if (tagName === 'A') continue;

        const errors = await collectConsoleErrors(page, async () => {
          await btn.click({ timeout: 3000 }).catch(() => {
            // Button may have become detached after a previous click caused re-render
          });
          await page.waitForTimeout(300);
        });

        // Close any dialog/modal that may have opened so subsequent clicks work
        const escCloseable = page.locator('[role="dialog"]:visible, [role="alertdialog"]:visible');
        if (await escCloseable.count() > 0) {
          await page.keyboard.press('Escape');
          await page.waitForTimeout(200);
        }

        // No unhandled JS errors should appear in the error boundary
        const bodyText = await page.locator('body').textContent();
        expect(
          bodyText,
          `Button "${label?.trim()}" on ${name} caused an error boundary`
        ).not.toContain('Something went wrong');
      }
    });
  }

  test('Dashboard — stat cards and panels are interactive', async ({ page }) => {
    await page.goto('/dashboard');
    await waitForPageReady(page);

    // Stat/metric cards should be present
    const cards = page.locator('[class*="card"], [class*="Card"]');
    const cardCount = await cards.count();
    expect(cardCount).toBeGreaterThan(0);

    // Click the first few cards — they often navigate or expand
    for (let i = 0; i < Math.min(cardCount, 5); i++) {
      const card = cards.nth(i);
      if (await card.isVisible()) {
        await card.click({ timeout: 2000 }).catch(() => {});
        await page.waitForTimeout(200);
      }
    }

    // No error boundary
    await expect(page.locator('body')).not.toContainText('Something went wrong');
  });

  test('Settings — all tabs/sections can be opened', async ({ page }) => {
    await page.goto('/settings');
    await waitForPageReady(page);

    // Settings typically has tab-like sections; click all tab triggers
    const tabs = page.locator('[role="tab"]:visible, button:has-text("General"):visible, button:has-text("Clusters"):visible, button:has-text("Theme"):visible');
    const tabCount = await tabs.count();

    for (let i = 0; i < tabCount; i++) {
      const tab = tabs.nth(i);
      if (await tab.isVisible()) {
        await tab.click({ timeout: 2000 }).catch(() => {});
        await page.waitForTimeout(300);
        await expect(page.locator('body')).not.toContainText('Something went wrong');
      }
    }
  });

  test('Modal/dialog buttons — Cancel and close work', async ({ page }) => {
    // Navigate to a list page and try to trigger a dialog via any "Delete" or action button
    await page.goto('/pods');
    await waitForPageReady(page);

    // Try clicking the first row to get to a detail page
    const firstRow = page.locator('tbody tr:not(.animate-pulse) td:first-child a, tbody tr:not(.animate-pulse) td:first-child').first();
    if (await firstRow.isVisible().catch(() => false)) {
      await firstRow.click({ timeout: 3000 }).catch(() => {});
      await waitForPageReady(page);
    }

    // Look for a Delete button in the header
    const deleteBtn = page.locator('button:has-text("Delete"):visible').first();
    if (await deleteBtn.isVisible().catch(() => false)) {
      await deleteBtn.click();
      await page.waitForTimeout(500);

      // Confirmation dialog should appear
      const dialog = page.locator('[role="dialog"]:visible, [role="alertdialog"]:visible');
      if (await dialog.isVisible().catch(() => false)) {
        // Look for Cancel button inside the dialog
        const cancelBtn = dialog.locator('button:has-text("Cancel")');
        if (await cancelBtn.isVisible().catch(() => false)) {
          await cancelBtn.click();
          await page.waitForTimeout(300);
          // Dialog should be closed
          await expect(dialog).not.toBeVisible();
        }
      }
    }

    await expect(page.locator('body')).not.toContainText('Something went wrong');
  });

  test('Topology page — controls and view buttons work', async ({ page }) => {
    await page.goto('/topology');
    await waitForPageReady(page);

    // Click all visible buttons on topology page
    const buttons = page.locator('button:visible:not([disabled])');
    const count = await buttons.count();

    for (let i = 0; i < Math.min(count, 20); i++) {
      const btn = buttons.nth(i);
      if (!(await btn.isVisible().catch(() => false))) continue;

      await btn.click({ timeout: 2000 }).catch(() => {});
      await page.waitForTimeout(200);

      // Close any overlays
      const dialog = page.locator('[role="dialog"]:visible');
      if (await dialog.count() > 0) {
        await page.keyboard.press('Escape');
        await page.waitForTimeout(200);
      }
    }

    await expect(page.locator('body')).not.toContainText('Something went wrong');
  });
});
