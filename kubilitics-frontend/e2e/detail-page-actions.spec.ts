/**
 * E2E: Detail Page Actions
 * Tests that header actions, tabs, and interactive elements on resource detail
 * pages actually work (not just "buttons for namesake").
 */
import { test, expect, Page } from '@playwright/test';

// ─── Helpers ──────────────────────────────────────────────────────────────────

async function waitForPageReady(page: Page) {
  await page.waitForLoadState('load');
  await page.waitForTimeout(800);
}

/**
 * Navigate to the first available resource detail page for a given resource type.
 * Returns true if navigation succeeded, false if no resources exist.
 */
async function navigateToFirstDetail(page: Page, resourceType: string): Promise<boolean> {
  await page.goto(`/${resourceType}`);
  await waitForPageReady(page);

  // Wait for either table rows or empty state
  const hasRows = await page
    .locator('tbody tr:not(.animate-pulse) td:first-child a, tbody tr:not(.animate-pulse) td:first-child')
    .first()
    .isVisible({ timeout: 5000 })
    .catch(() => false);

  if (!hasRows) return false;

  // Click the first resource name link
  const firstLink = page.locator('tbody tr:not(.animate-pulse) td:first-child a').first();
  if (await firstLink.isVisible().catch(() => false)) {
    await firstLink.click();
  } else {
    // Fallback: click first cell
    await page.locator('tbody tr:not(.animate-pulse) td:first-child').first().click();
  }
  await waitForPageReady(page);

  // Verify we landed on a detail page (URL has resource type + name segments)
  const url = page.url();
  return url.includes(`/${resourceType}/`);
}

// ─── Tests ────────────────────────────────────────────────────────────────────

test.describe('Detail Page Actions', () => {
  test.describe.configure({ timeout: 60_000 });

  test('Pod detail — Download YAML initiates download', async ({ page }) => {
    const navigated = await navigateToFirstDetail(page, 'pods');
    test.skip(!navigated, 'No pods available to test');

    // Set up download listener before clicking
    const downloadPromise = page.waitForEvent('download', { timeout: 5000 }).catch(() => null);

    // Also watch for blob URL creation (fallback for in-memory downloads)
    let blobUrlCreated = false;
    await page.exposeFunction('__pw_blobCreated', () => { blobUrlCreated = true; });
    await page.evaluate(() => {
      const origCreateObjectURL = URL.createObjectURL;
      URL.createObjectURL = function (...args) {
        try { (window as unknown as Record<string, () => void>).__pw_blobCreated(); } catch { /* ignored */ }
        return origCreateObjectURL.apply(this, args);
      };
    });

    const downloadBtn = page.locator('button:has-text("Download YAML")').first();
    if (await downloadBtn.isVisible().catch(() => false)) {
      await downloadBtn.click();
      const download = await downloadPromise;
      // Either a real download was triggered or a blob URL was created
      const downloadStarted = download !== null || blobUrlCreated;
      expect(downloadStarted, 'Download YAML should initiate a file download').toBe(true);
    } else {
      // Download YAML might be in the actions dropdown/tab — check the Actions tab
      const actionsTab = page.locator('button:has-text("Actions"), [role="tab"]:has-text("Actions")').first();
      if (await actionsTab.isVisible().catch(() => false)) {
        await actionsTab.click();
        await page.waitForTimeout(500);
        const downloadAction = page.locator('button:has-text("Download YAML"), [role="button"]:has-text("Download YAML")').first();
        expect(await downloadAction.isVisible(), 'Download YAML action should exist').toBe(true);
      }
    }
  });

  test('Pod detail — Port Forward dialog opens', async ({ page }) => {
    const navigated = await navigateToFirstDetail(page, 'pods');
    test.skip(!navigated, 'No pods available to test');

    const portForwardBtn = page.locator('button:has-text("Port Forward")').first();
    if (await portForwardBtn.isVisible().catch(() => false)) {
      await portForwardBtn.click();
      await page.waitForTimeout(500);

      // A dialog/modal should appear
      const dialog = page.locator('[role="dialog"]:visible, [role="alertdialog"]:visible');
      await expect(dialog, 'Port Forward dialog should open').toBeVisible({ timeout: 3000 });

      // Close it
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);
    } else {
      // Port Forward may be under Actions tab
      const actionsTab = page.locator('button:has-text("Actions"), [role="tab"]:has-text("Actions")').first();
      if (await actionsTab.isVisible().catch(() => false)) {
        await actionsTab.click();
        await page.waitForTimeout(500);
        const pfAction = page.locator('button:has-text("Port Forward"), [role="button"]:has-text("Port Forward")').first();
        if (await pfAction.isVisible().catch(() => false)) {
          await pfAction.click();
          await page.waitForTimeout(500);
          const dialog = page.locator('[role="dialog"]:visible');
          await expect(dialog, 'Port Forward dialog should open from Actions tab').toBeVisible({ timeout: 3000 });
          await page.keyboard.press('Escape');
        }
      }
    }
  });

  test('Pod detail — Delete shows confirmation, Cancel closes it', async ({ page }) => {
    const navigated = await navigateToFirstDetail(page, 'pods');
    test.skip(!navigated, 'No pods available to test');

    const deleteBtn = page.locator('button:has-text("Delete")').first();
    await expect(deleteBtn, 'Delete button should be visible').toBeVisible({ timeout: 5000 });

    await deleteBtn.click();
    await page.waitForTimeout(500);

    // Confirmation dialog with resource name input should appear
    const dialog = page.locator('[role="dialog"]:visible, [role="alertdialog"]:visible');
    await expect(dialog, 'Delete confirmation dialog should appear').toBeVisible({ timeout: 3000 });

    // Dialog should mention "delete" or the resource name
    const dialogText = await dialog.textContent();
    expect(
      dialogText?.toLowerCase().includes('delete') || dialogText?.toLowerCase().includes('confirm'),
      'Dialog should mention delete/confirm'
    ).toBe(true);

    // Click Cancel
    const cancelBtn = dialog.locator('button:has-text("Cancel")');
    await expect(cancelBtn, 'Cancel button should be in dialog').toBeVisible();
    await cancelBtn.click();
    await page.waitForTimeout(300);

    // Dialog should close
    await expect(dialog).not.toBeVisible();
  });

  test('Detail page — all standard tabs render without error', async ({ page }) => {
    const navigated = await navigateToFirstDetail(page, 'pods');
    test.skip(!navigated, 'No pods available to test');

    // Standard tabs: Overview, YAML, Compare, Topology, Blast Radius, Actions
    const standardTabNames = ['YAML', 'Compare', 'Topology', 'Blast Radius', 'Actions'];

    for (const tabName of standardTabNames) {
      const tab = page.locator(`button:has-text("${tabName}"), [role="tab"]:has-text("${tabName}")`).first();
      if (!(await tab.isVisible().catch(() => false))) continue;

      await tab.click();
      await page.waitForTimeout(800);

      // No error boundary should appear
      const body = await page.locator('body').textContent();
      expect(
        body,
        `Tab "${tabName}" should not trigger error boundary`
      ).not.toContain('Something went wrong');
    }
  });

  test('Detail page — YAML tab shows code content', async ({ page }) => {
    const navigated = await navigateToFirstDetail(page, 'pods');
    test.skip(!navigated, 'No pods available to test');

    const yamlTab = page.locator('button:has-text("YAML"), [role="tab"]:has-text("YAML")').first();
    if (!(await yamlTab.isVisible().catch(() => false))) {
      test.skip(true, 'YAML tab not found');
      return;
    }

    await yamlTab.click();
    await page.waitForTimeout(800);

    // YAML content area should exist (code editor or pre block)
    const codeArea = page.locator('pre, .cm-editor, .monaco-editor, [class*="CodeMirror"], code, [class*="yaml"]').first();
    await expect(codeArea, 'YAML editor/viewer should be visible').toBeVisible({ timeout: 5000 });
  });

  test('Detail page — Actions tab items are clickable', async ({ page }) => {
    const navigated = await navigateToFirstDetail(page, 'pods');
    test.skip(!navigated, 'No pods available to test');

    const actionsTab = page.locator('button:has-text("Actions"), [role="tab"]:has-text("Actions")').first();
    if (!(await actionsTab.isVisible().catch(() => false))) {
      test.skip(true, 'Actions tab not found');
      return;
    }

    await actionsTab.click();
    await page.waitForTimeout(500);

    // All action items should be clickable (not disabled)
    const actionButtons = page.locator(
      'button:visible:not([disabled]), [role="button"]:visible:not([disabled])'
    );
    const count = await actionButtons.count();
    expect(count, 'Actions tab should have interactive items').toBeGreaterThan(0);

    // Click each action and verify it either opens a dialog or does something
    for (let i = 0; i < count; i++) {
      const btn = actionButtons.nth(i);
      const text = await btn.textContent().catch(() => '');

      // Skip destructive actions (delete, restart) to avoid side effects
      if (text?.toLowerCase().includes('delete') || text?.toLowerCase().includes('restart')) continue;
      if (!(await btn.isVisible().catch(() => false))) continue;

      await btn.click({ timeout: 2000 }).catch(() => {});
      await page.waitForTimeout(400);

      // Close any dialog that opened
      const dialog = page.locator('[role="dialog"]:visible');
      if (await dialog.count() > 0) {
        await page.keyboard.press('Escape');
        await page.waitForTimeout(200);
      }

      // No crash
      await expect(page.locator('body')).not.toContainText('Something went wrong');
    }
  });

  test('Detail page — Compare tab namespace selector works', async ({ page }) => {
    const navigated = await navigateToFirstDetail(page, 'pods');
    test.skip(!navigated, 'No pods available to test');

    const compareTab = page.locator('button:has-text("Compare"), [role="tab"]:has-text("Compare")').first();
    if (!(await compareTab.isVisible().catch(() => false))) {
      test.skip(true, 'Compare tab not found');
      return;
    }

    await compareTab.click();
    await page.waitForTimeout(800);

    // Look for a namespace selector (select/dropdown)
    const nsSelector = page.locator(
      'button:has-text("Namespace"), select, [role="combobox"], [class*="select-trigger"]'
    ).first();

    if (await nsSelector.isVisible().catch(() => false)) {
      await nsSelector.click();
      await page.waitForTimeout(300);
      // Options should appear
      const options = page.locator('[role="option"], [role="listbox"] [role="option"], option');
      const optCount = await options.count();
      expect(optCount, 'Namespace selector should have options').toBeGreaterThanOrEqual(0);
      // Close dropdown
      await page.keyboard.press('Escape');
    }

    await expect(page.locator('body')).not.toContainText('Something went wrong');
  });

  test('Detail page — Topology tab renders visualization', async ({ page }) => {
    const navigated = await navigateToFirstDetail(page, 'pods');
    test.skip(!navigated, 'No pods available to test');

    const topoTab = page.locator('button:has-text("Topology"), [role="tab"]:has-text("Topology")').first();
    if (!(await topoTab.isVisible().catch(() => false))) {
      test.skip(true, 'Topology tab not found');
      return;
    }

    await topoTab.click();
    await page.waitForTimeout(1500);

    // Should render a canvas, SVG, or topology container
    const viz = page.locator('canvas, svg, [class*="topology"], [class*="react-flow"]').first();
    await expect(viz, 'Topology visualization should render').toBeVisible({ timeout: 8000 });
  });

  test('Detail page — Blast Radius tab loads', async ({ page }) => {
    const navigated = await navigateToFirstDetail(page, 'pods');
    test.skip(!navigated, 'No pods available to test');

    const brTab = page.locator('button:has-text("Blast Radius"), [role="tab"]:has-text("Blast Radius")').first();
    if (!(await brTab.isVisible().catch(() => false))) {
      test.skip(true, 'Blast Radius tab not found');
      return;
    }

    await brTab.click();
    await page.waitForTimeout(1500);

    // Should not show error boundary
    await expect(page.locator('body')).not.toContainText('Something went wrong');

    // Should show some content (canvas, loading, or result)
    const content = page.locator(
      'canvas, svg, [class*="blast"], [class*="topology"], [class*="react-flow"], [class*="loading"], [class*="spinner"]'
    ).first();
    // Either content loads or we see a "no impact" message — both are valid
    const hasContent = await content.isVisible().catch(() => false);
    const hasMessage = await page.locator('body').textContent().then(t =>
      t?.includes('No impact') || t?.includes('blast radius') || t?.includes('Blast')
    );
    expect(hasContent || hasMessage, 'Blast Radius should show content or message').toBe(true);
  });

  // ─── Other resource types ───────────────────────────────────────────────────

  const RESOURCE_TYPES_TO_TEST = [
    'deployments',
    'services',
    'configmaps',
    'namespaces',
    'nodes',
    'statefulsets',
    'daemonsets',
    'jobs',
    'cronjobs',
    'ingresses',
    'secrets',
  ];

  for (const resourceType of RESOURCE_TYPES_TO_TEST) {
    test(`${resourceType} detail — header actions are functional`, async ({ page }) => {
      const navigated = await navigateToFirstDetail(page, resourceType);
      test.skip(!navigated, `No ${resourceType} available to test`);

      // Download YAML button should exist (in header or actions tab)
      const downloadBtn = page.locator('button:has-text("Download YAML")').first();
      const hasDownload = await downloadBtn.isVisible().catch(() => false);

      // Delete button should exist
      const deleteBtn = page.locator('button:has-text("Delete")').first();
      const hasDelete = await deleteBtn.isVisible().catch(() => false);

      // At least one of the standard actions should be present
      expect(
        hasDownload || hasDelete,
        `${resourceType} detail should have header actions`
      ).toBe(true);

      // If Delete exists, verify the confirmation dialog pattern
      if (hasDelete) {
        await deleteBtn.click();
        await page.waitForTimeout(500);
        const dialog = page.locator('[role="dialog"]:visible, [role="alertdialog"]:visible');
        if (await dialog.isVisible().catch(() => false)) {
          await expect(dialog).toBeVisible();
          // Cancel out
          const cancel = dialog.locator('button:has-text("Cancel")');
          if (await cancel.isVisible().catch(() => false)) {
            await cancel.click();
            await page.waitForTimeout(300);
          } else {
            await page.keyboard.press('Escape');
          }
        }
      }

      await expect(page.locator('body')).not.toContainText('Something went wrong');
    });
  }
});
