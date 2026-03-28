/**
 * E2E: Dead Buttons Scanner
 * Generic scanner that visits each major page, finds all buttons,
 * clicks them, and reports buttons that produce NO visible effect.
 *
 * A "dead button" is one where clicking produces:
 *   - No dialog/modal opened
 *   - No navigation change
 *   - No toast/notification
 *   - No DOM change
 *   - No console error (errors indicate a broken button, not dead)
 *
 * This test collects and reports suspected dead buttons.
 */
import { test, expect, Page } from '@playwright/test';

// ─── Helpers ──────────────────────────────────────────────────────────────────

async function waitForPageReady(page: Page) {
  await page.waitForLoadState('load');
  await page.waitForTimeout(800);
}

interface ButtonClickResult {
  page: string;
  buttonText: string;
  buttonIndex: number;
  effect: 'dialog' | 'navigation' | 'toast' | 'dom-change' | 'console-error' | 'none';
  details?: string;
}

/**
 * Snapshot observable DOM state so we can diff after a click.
 */
async function capturePageState(page: Page) {
  return {
    url: page.url(),
    dialogCount: await page.locator('[role="dialog"]:visible, [role="alertdialog"]:visible').count(),
    toastCount: await page.locator(
      '[data-sonner-toast], [class*="toast"]:visible, [role="status"]:visible, [class*="Toaster"] [data-visible="true"]'
    ).count(),
    // Use a hash of visible text to detect DOM changes without being too noisy
    bodyLength: (await page.locator('body').textContent().catch(() => ''))?.length ?? 0,
  };
}

/**
 * Detect what changed after a button click by comparing snapshots.
 */
function detectEffect(
  before: Awaited<ReturnType<typeof capturePageState>>,
  after: Awaited<ReturnType<typeof capturePageState>>,
  consoleErrors: string[]
): ButtonClickResult['effect'] {
  if (after.url !== before.url) return 'navigation';
  if (after.dialogCount > before.dialogCount) return 'dialog';
  if (after.toastCount > before.toastCount) return 'toast';
  if (consoleErrors.length > 0) return 'console-error';
  // Allow some tolerance for minor DOM updates (re-renders, timestamps)
  if (Math.abs(after.bodyLength - before.bodyLength) > 20) return 'dom-change';
  return 'none';
}

// ─── Pages to scan ────────────────────────────────────────────────────────────

const PAGES_TO_SCAN = [
  { name: 'Dashboard', path: '/dashboard' },
  { name: 'Fleet', path: '/fleet' },
  { name: 'Topology', path: '/topology' },
  { name: 'Templates', path: '/templates' },
  { name: 'Settings', path: '/settings' },
  { name: 'Pods', path: '/pods' },
  { name: 'Deployments', path: '/deployments' },
  { name: 'Services', path: '/services' },
  { name: 'Nodes', path: '/nodes' },
  { name: 'Namespaces', path: '/namespaces' },
  { name: 'ConfigMaps', path: '/configmaps' },
  { name: 'Ingresses', path: '/ingresses' },
  { name: 'Workloads Overview', path: '/workloads' },
  { name: 'Networking Overview', path: '/networking' },
  { name: 'Storage Overview', path: '/storage' },
  { name: 'Cluster Overview', path: '/cluster' },
];

// Buttons to skip (they are expected to be non-interactive or are toggle/state buttons)
const SKIP_PATTERNS = [
  /collapse/i,
  /expand/i,
  /chevron/i,
  /close/i,
  /dismiss/i,
  // Sidebar collapse/expand toggles
  /menu/i,
  /sidebar/i,
];

// ─── Tests ────────────────────────────────────────────────────────────────────

test.describe('Dead Buttons Scanner', () => {
  test.describe.configure({ timeout: 120_000 });

  for (const { name, path } of PAGES_TO_SCAN) {
    test(`scan ${name} for dead buttons`, async ({ page }) => {
      await page.goto(path);
      await waitForPageReady(page);

      const results: ButtonClickResult[] = [];

      // Find all visible, enabled buttons in the MAIN content area (skip sidebar nav)
      const buttons = page.locator(
        'main button:visible:not([disabled]):not([aria-hidden="true"]), ' +
        '[role="main"] button:visible:not([disabled]):not([aria-hidden="true"]), ' +
        // Fallback: all buttons if no main landmark
        'button:visible:not([disabled]):not([aria-hidden="true"])'
      );
      const count = await buttons.count();

      // Cap at 25 buttons per page to keep test time reasonable
      const maxButtons = Math.min(count, 25);

      for (let i = 0; i < maxButtons; i++) {
        const btn = buttons.nth(i);
        if (!(await btn.isVisible().catch(() => false))) continue;

        const buttonText = (await btn.textContent().catch(() => ''))?.trim() || '';
        const ariaLabel = (await btn.getAttribute('aria-label').catch(() => '')) || '';
        const identifier = buttonText || ariaLabel || `button[${i}]`;

        // Skip buttons matching skip patterns
        if (SKIP_PATTERNS.some(p => p.test(identifier))) continue;

        // Skip very small icon-only buttons that are likely decorative
        const box = await btn.boundingBox().catch(() => null);
        if (box && box.width < 16 && box.height < 16) continue;

        // Capture state before click
        const before = await capturePageState(page);
        const consoleErrors: string[] = [];
        const errorHandler = (msg: import('@playwright/test').ConsoleMessage) => {
          if (msg.type() === 'error') consoleErrors.push(msg.text());
        };
        page.on('console', errorHandler);

        // Click the button
        await btn.click({ timeout: 3000 }).catch(() => {});
        await page.waitForTimeout(500);

        page.off('console', errorHandler);

        // Capture state after click
        const after = await capturePageState(page);
        const effect = detectEffect(before, after, consoleErrors);

        results.push({
          page: name,
          buttonText: identifier.slice(0, 60),
          buttonIndex: i,
          effect,
          details: effect === 'console-error' ? consoleErrors[0]?.slice(0, 100) : undefined,
        });

        // Clean up: close any dialogs/modals that opened
        const openDialogs = page.locator('[role="dialog"]:visible, [role="alertdialog"]:visible');
        if (await openDialogs.count() > 0) {
          await page.keyboard.press('Escape');
          await page.waitForTimeout(200);
        }

        // If navigation happened, go back to the original page
        if (page.url() !== before.url) {
          await page.goto(path);
          await waitForPageReady(page);
        }

        // Verify no crash after each button
        const bodyText = await page.locator('body').textContent().catch(() => '');
        if (bodyText?.includes('Something went wrong')) {
          // Page crashed — reload and continue
          await page.goto(path);
          await waitForPageReady(page);
        }
      }

      // ─── Report ──────────────────────────────────────────────────────────

      const deadButtons = results.filter(r => r.effect === 'none');
      const errorButtons = results.filter(r => r.effect === 'console-error');

      // Log dead buttons for visibility (these appear in the HTML report)
      if (deadButtons.length > 0) {
        console.log(`\n--- DEAD BUTTONS on ${name} (${path}) ---`);
        for (const db of deadButtons) {
          console.log(`  [DEAD] Button: "${db.buttonText}" (index ${db.buttonIndex})`);
        }
        console.log(`  Total: ${deadButtons.length} / ${results.length} buttons had no visible effect\n`);
      }

      if (errorButtons.length > 0) {
        console.log(`\n--- ERROR BUTTONS on ${name} (${path}) ---`);
        for (const eb of errorButtons) {
          console.log(`  [ERROR] Button: "${eb.buttonText}" — ${eb.details}`);
        }
      }

      // Soft assertion: warn but don't fail for dead buttons (they're suspects, not confirmed bugs)
      // Hard assertion: no button should cause an error boundary crash
      const pageText = await page.locator('body').textContent().catch(() => '');
      expect(pageText).not.toContain('Something went wrong');
    });
  }

  test('scan detail page buttons for dead actions', async ({ page }) => {
    // Navigate to a pod detail page (if available)
    await page.goto('/pods');
    await waitForPageReady(page);

    const firstLink = page.locator('tbody tr:not(.animate-pulse) td:first-child a').first();
    const hasResources = await firstLink.isVisible({ timeout: 5000 }).catch(() => false);
    test.skip(!hasResources, 'No pods available for detail button scan');

    await firstLink.click();
    await waitForPageReady(page);

    const detailUrl = page.url();
    const results: ButtonClickResult[] = [];

    const buttons = page.locator('button:visible:not([disabled]):not([aria-hidden="true"])');
    const count = await buttons.count();

    for (let i = 0; i < Math.min(count, 30); i++) {
      const btn = buttons.nth(i);
      if (!(await btn.isVisible().catch(() => false))) continue;

      const buttonText = (await btn.textContent().catch(() => ''))?.trim() || '';
      const ariaLabel = (await btn.getAttribute('aria-label').catch(() => '')) || '';
      const identifier = buttonText || ariaLabel || `button[${i}]`;

      if (SKIP_PATTERNS.some(p => p.test(identifier))) continue;

      // Skip destructive actions
      if (/delete|restart|remove/i.test(identifier)) continue;

      const before = await capturePageState(page);
      const consoleErrors: string[] = [];
      const errorHandler = (msg: import('@playwright/test').ConsoleMessage) => {
        if (msg.type() === 'error') consoleErrors.push(msg.text());
      };
      page.on('console', errorHandler);

      await btn.click({ timeout: 3000 }).catch(() => {});
      await page.waitForTimeout(500);

      page.off('console', errorHandler);
      const after = await capturePageState(page);
      const effect = detectEffect(before, after, consoleErrors);

      results.push({
        page: 'Pod Detail',
        buttonText: identifier.slice(0, 60),
        buttonIndex: i,
        effect,
        details: effect === 'console-error' ? consoleErrors[0]?.slice(0, 100) : undefined,
      });

      // Clean up
      if (await page.locator('[role="dialog"]:visible').count() > 0) {
        await page.keyboard.press('Escape');
        await page.waitForTimeout(200);
      }
      if (page.url() !== detailUrl) {
        await page.goto(detailUrl);
        await waitForPageReady(page);
      }
      if ((await page.locator('body').textContent().catch(() => ''))?.includes('Something went wrong')) {
        await page.goto(detailUrl);
        await waitForPageReady(page);
      }
    }

    const deadButtons = results.filter(r => r.effect === 'none');

    if (deadButtons.length > 0) {
      console.log('\n--- DEAD BUTTONS on Pod Detail ---');
      for (const db of deadButtons) {
        console.log(`  [DEAD] Button: "${db.buttonText}" (index ${db.buttonIndex})`);
      }
      console.log(`  Total: ${deadButtons.length} / ${results.length} buttons had no visible effect\n`);
    }

    // Summary assertion
    const summary = results.reduce<Record<string, number>>((acc, r) => {
      acc[r.effect] = (acc[r.effect] || 0) + 1;
      return acc;
    }, {});
    console.log(`\nButton scan summary for Pod Detail: ${JSON.stringify(summary)}`);

    // At least some buttons should have had an effect
    const effectiveButtons = results.filter(r => r.effect !== 'none');
    expect(
      effectiveButtons.length,
      'At least some buttons on the detail page should produce a visible effect'
    ).toBeGreaterThan(0);

    await expect(page.locator('body')).not.toContainText('Something went wrong');
  });

  test('aggregate dead button report across all pages', async ({ page }) => {
    const allDeadButtons: ButtonClickResult[] = [];

    for (const { name, path } of PAGES_TO_SCAN.slice(0, 8)) {
      await page.goto(path);
      await waitForPageReady(page);

      // Quick scan: only check the first 10 buttons per page
      const buttons = page.locator('button:visible:not([disabled])');
      const count = Math.min(await buttons.count(), 10);

      for (let i = 0; i < count; i++) {
        const btn = buttons.nth(i);
        if (!(await btn.isVisible().catch(() => false))) continue;

        const buttonText = (await btn.textContent().catch(() => ''))?.trim() || '';
        if (SKIP_PATTERNS.some(p => p.test(buttonText))) continue;

        const before = await capturePageState(page);

        await btn.click({ timeout: 2000 }).catch(() => {});
        await page.waitForTimeout(400);

        const after = await capturePageState(page);
        const effect = detectEffect(before, after, []);

        if (effect === 'none') {
          allDeadButtons.push({
            page: name,
            buttonText: buttonText.slice(0, 60),
            buttonIndex: i,
            effect: 'none',
          });
        }

        // Clean up
        if (await page.locator('[role="dialog"]:visible').count() > 0) {
          await page.keyboard.press('Escape');
          await page.waitForTimeout(200);
        }
        if (page.url() !== `${page.url().split('?')[0]}`) {
          // URL changed — navigate back
          await page.goto(path);
          await waitForPageReady(page);
        }
        if ((await page.locator('body').textContent().catch(() => ''))?.includes('Something went wrong')) {
          await page.goto(path);
          await waitForPageReady(page);
        }
      }
    }

    // Print aggregate report
    console.log('\n=== AGGREGATE DEAD BUTTON REPORT ===');
    if (allDeadButtons.length === 0) {
      console.log('  No dead buttons detected across scanned pages.');
    } else {
      const byPage = allDeadButtons.reduce<Record<string, string[]>>((acc, db) => {
        (acc[db.page] = acc[db.page] || []).push(db.buttonText);
        return acc;
      }, {});
      for (const [pageName, btns] of Object.entries(byPage)) {
        console.log(`  ${pageName}:`);
        for (const b of btns) {
          console.log(`    - "${b}"`);
        }
      }
      console.log(`  TOTAL: ${allDeadButtons.length} suspected dead buttons`);
    }
    console.log('====================================\n');

    // No crash assertion
    await expect(page.locator('body')).not.toContainText('Something went wrong');
  });
});
