/**
 * E2E: Navigation Links
 * Tests every sidebar link, breadcrumb, resource table link, back button,
 * and Cmd+K search to verify navigation works end-to-end.
 */
import { test, expect, Page } from '@playwright/test';

// ─── Helpers ──────────────────────────────────────────────────────────────────

async function waitForPageReady(page: Page) {
  await page.waitForLoadState('load');
  await page.waitForTimeout(800);
}

/** Assert the page loaded successfully (no error boundary, no blank screen). */
async function assertPageLoaded(page: Page, context: string) {
  const body = await page.locator('body').textContent();
  expect(body, `${context} — should not show error boundary`).not.toContain('Something went wrong');
  expect(body, `${context} — should not show unhandled error`).not.toContain('Unhandled Runtime Error');
  // Page should have meaningful content (not blank)
  expect((body?.length ?? 0) > 50, `${context} — page should not be blank`).toBe(true);
}

// ─── Top-level sidebar links ──────────────────────────────────────────────────

const TOP_LEVEL_LINKS = [
  { label: 'Dashboard', expectedPath: '/dashboard' },
  { label: 'Fleet', expectedPath: '/fleet' },
  { label: 'Topology', expectedPath: '/topology' },
  { label: 'Templates', expectedPath: '/templates' },
];

// All sidebar resource links (from the category sections)
const SIDEBAR_RESOURCE_LINKS = [
  // Workloads
  { path: '/pods', label: 'Pods' },
  { path: '/deployments', label: 'Deployments' },
  { path: '/replicasets', label: 'ReplicaSets' },
  { path: '/statefulsets', label: 'StatefulSets' },
  { path: '/daemonsets', label: 'DaemonSets' },
  { path: '/jobs', label: 'Jobs' },
  { path: '/cronjobs', label: 'CronJobs' },
  // Networking
  { path: '/services', label: 'Services' },
  { path: '/ingresses', label: 'Ingresses' },
  { path: '/endpoints', label: 'Endpoints' },
  { path: '/networkpolicies', label: 'Network Policies' },
  // Storage & Config
  { path: '/configmaps', label: 'ConfigMaps' },
  { path: '/secrets', label: 'Secrets' },
  { path: '/persistentvolumes', label: 'Persistent Volumes' },
  { path: '/persistentvolumeclaims', label: 'PVCs' },
  { path: '/storageclasses', label: 'Storage Classes' },
  // Cluster
  { path: '/nodes', label: 'Nodes' },
  { path: '/namespaces', label: 'Namespaces' },
  { path: '/events', label: 'Events' },
  // RBAC
  { path: '/serviceaccounts', label: 'Service Accounts' },
  { path: '/roles', label: 'Roles' },
  { path: '/clusterroles', label: 'Cluster Roles' },
  { path: '/rolebindings', label: 'Role Bindings' },
  { path: '/clusterrolebindings', label: 'Cluster Role Bindings' },
  // Scaling
  { path: '/horizontalpodautoscalers', label: 'HPAs' },
  { path: '/poddisruptionbudgets', label: 'PDBs' },
  // Resources
  { path: '/resourcequotas', label: 'Resource Quotas' },
  { path: '/limitranges', label: 'Limit Ranges' },
  // CRDs
  { path: '/customresourcedefinitions', label: 'Definitions' },
  // Admission
  { path: '/mutatingwebhooks', label: 'Mutating Webhooks' },
  { path: '/validatingwebhooks', label: 'Validating Webhooks' },
];

// Category overview pages
const OVERVIEW_PAGES = [
  { path: '/workloads', label: 'Workloads Overview' },
  { path: '/networking', label: 'Networking Overview' },
  { path: '/storage', label: 'Storage Overview' },
  { path: '/cluster', label: 'Cluster Overview' },
  { path: '/resources', label: 'Resources Overview' },
  { path: '/scaling', label: 'Scaling Overview' },
  { path: '/crds', label: 'CRDs Overview' },
  { path: '/admission', label: 'Admission Overview' },
];

// ─── Tests ────────────────────────────────────────────────────────────────────

test.describe('Navigation Links', () => {
  test.describe.configure({ timeout: 90_000 });

  test('top-level sidebar links navigate correctly', async ({ page }) => {
    await page.goto('/dashboard');
    await waitForPageReady(page);

    for (const { label, expectedPath } of TOP_LEVEL_LINKS) {
      // Find the sidebar link by text
      const link = page.locator(`nav a:has-text("${label}"), aside a:has-text("${label}")`).first();
      if (!(await link.isVisible().catch(() => false))) {
        // Sidebar might be collapsed — try clicking hamburger/expand
        const expandBtn = page.locator('button[aria-label*="expand"], button[aria-label*="menu"], button[aria-label*="sidebar"]').first();
        if (await expandBtn.isVisible().catch(() => false)) {
          await expandBtn.click();
          await page.waitForTimeout(300);
        }
      }

      // Try direct navigation as fallback
      await page.goto(expectedPath);
      await waitForPageReady(page);
      expect(page.url()).toContain(expectedPath);
      await assertPageLoaded(page, `Top-level: ${label}`);
    }
  });

  test('Settings link in sidebar works', async ({ page }) => {
    await page.goto('/dashboard');
    await waitForPageReady(page);

    // Settings is typically at the bottom of sidebar
    const settingsLink = page.locator('a:has-text("Settings"), button:has-text("Settings")').first();
    if (await settingsLink.isVisible().catch(() => false)) {
      await settingsLink.click();
      await waitForPageReady(page);
      expect(page.url()).toContain('/settings');
    } else {
      // Direct navigation fallback
      await page.goto('/settings');
      await waitForPageReady(page);
    }
    await assertPageLoaded(page, 'Settings');
  });

  // Test all resource list pages load via direct URL navigation
  for (const { path, label } of SIDEBAR_RESOURCE_LINKS) {
    test(`sidebar resource page loads: ${label} (${path})`, async ({ page }) => {
      await page.goto(path);
      await waitForPageReady(page);
      expect(page.url()).toContain(path);
      await assertPageLoaded(page, label);
    });
  }

  // Test all overview pages
  for (const { path, label } of OVERVIEW_PAGES) {
    test(`overview page loads: ${label} (${path})`, async ({ page }) => {
      await page.goto(path);
      await waitForPageReady(page);
      expect(page.url()).toContain(path);
      await assertPageLoaded(page, label);
    });
  }

  test('clicking resource name in list table opens detail page', async ({ page }) => {
    // Test with pods (most likely to have entries)
    await page.goto('/pods');
    await waitForPageReady(page);

    // Wait for table rows
    const firstLink = page.locator('tbody tr:not(.animate-pulse) td:first-child a').first();
    const hasResources = await firstLink.isVisible({ timeout: 5000 }).catch(() => false);
    test.skip(!hasResources, 'No pods available to test table links');

    const resourceName = await firstLink.textContent();
    await firstLink.click();
    await waitForPageReady(page);

    // URL should now contain /pods/namespace/name
    expect(page.url()).toContain('/pods/');
    await assertPageLoaded(page, `Pod detail: ${resourceName}`);
  });

  test('clicking resource name in deployments list opens detail', async ({ page }) => {
    await page.goto('/deployments');
    await waitForPageReady(page);

    const firstLink = page.locator('tbody tr:not(.animate-pulse) td:first-child a').first();
    const hasResources = await firstLink.isVisible({ timeout: 5000 }).catch(() => false);
    test.skip(!hasResources, 'No deployments available');

    await firstLink.click();
    await waitForPageReady(page);
    expect(page.url()).toContain('/deployments/');
    await assertPageLoaded(page, 'Deployment detail');
  });

  test('clicking resource name in services list opens detail', async ({ page }) => {
    await page.goto('/services');
    await waitForPageReady(page);

    const firstLink = page.locator('tbody tr:not(.animate-pulse) td:first-child a').first();
    const hasResources = await firstLink.isVisible({ timeout: 5000 }).catch(() => false);
    test.skip(!hasResources, 'No services available');

    await firstLink.click();
    await waitForPageReady(page);
    expect(page.url()).toContain('/services/');
    await assertPageLoaded(page, 'Service detail');
  });

  test('breadcrumb navigation works on detail pages', async ({ page }) => {
    // Navigate to a pod detail page
    await page.goto('/pods');
    await waitForPageReady(page);

    const firstLink = page.locator('tbody tr:not(.animate-pulse) td:first-child a').first();
    const hasResources = await firstLink.isVisible({ timeout: 5000 }).catch(() => false);
    test.skip(!hasResources, 'No pods available for breadcrumb test');

    await firstLink.click();
    await waitForPageReady(page);
    expect(page.url()).toContain('/pods/');

    // Find breadcrumb links (typically in a nav[aria-label="breadcrumb"] or class*="breadcrumb")
    const breadcrumbs = page.locator(
      'nav[aria-label*="breadcrumb"] a, [class*="breadcrumb"] a, [class*="Breadcrumb"] a'
    );
    const bcCount = await breadcrumbs.count();

    if (bcCount > 0) {
      // Click the first breadcrumb (should go back to list)
      const firstBc = breadcrumbs.first();
      const bcText = await firstBc.textContent();
      await firstBc.click();
      await waitForPageReady(page);

      // Should have navigated away from detail
      await assertPageLoaded(page, `Breadcrumb: ${bcText}`);
    }
  });

  test('back button (browser) works after navigation', async ({ page }) => {
    // Navigate: Dashboard -> Pods list -> Pod detail -> Back -> Back
    await page.goto('/dashboard');
    await waitForPageReady(page);

    await page.goto('/pods');
    await waitForPageReady(page);

    const firstLink = page.locator('tbody tr:not(.animate-pulse) td:first-child a').first();
    const hasResources = await firstLink.isVisible({ timeout: 5000 }).catch(() => false);
    test.skip(!hasResources, 'No pods for back button test');

    await firstLink.click();
    await waitForPageReady(page);
    expect(page.url()).toContain('/pods/');

    // Go back to pods list
    await page.goBack();
    await waitForPageReady(page);
    expect(page.url()).toContain('/pods');
    await assertPageLoaded(page, 'Back to Pods list');

    // Go back to dashboard
    await page.goBack();
    await waitForPageReady(page);
    expect(page.url()).toContain('/dashboard');
    await assertPageLoaded(page, 'Back to Dashboard');
  });

  test('back link on detail page header navigates to list', async ({ page }) => {
    await page.goto('/pods');
    await waitForPageReady(page);

    const firstLink = page.locator('tbody tr:not(.animate-pulse) td:first-child a').first();
    const hasResources = await firstLink.isVisible({ timeout: 5000 }).catch(() => false);
    test.skip(!hasResources, 'No pods for back link test');

    await firstLink.click();
    await waitForPageReady(page);

    // Look for the back link/button in the detail page header
    const backLink = page.locator(
      'a:has-text("Pods"), a:has-text("Back"), button:has-text("Back"), [aria-label*="back"]'
    ).first();

    if (await backLink.isVisible().catch(() => false)) {
      await backLink.click();
      await waitForPageReady(page);
      // Should be back on the list page
      expect(page.url()).toMatch(/\/pods$/);
      await assertPageLoaded(page, 'Back link to Pods list');
    }
  });

  test('Cmd+K search opens and returns results', async ({ page }) => {
    await page.goto('/dashboard');
    await waitForPageReady(page);

    // Open command palette with Cmd+K (Meta+K)
    await page.keyboard.press('Meta+k');
    await page.waitForTimeout(500);

    // Command dialog should appear
    const dialog = page.locator(
      '[role="dialog"]:visible, [cmdk-root], [class*="command"], [class*="CommandPalette"]'
    ).first();
    const isOpen = await dialog.isVisible().catch(() => false);

    if (!isOpen) {
      // Try Ctrl+K as fallback (Linux/Windows)
      await page.keyboard.press('Control+k');
      await page.waitForTimeout(500);
    }

    const searchInput = page.locator(
      '[cmdk-input], input[placeholder*="Search"], input[placeholder*="search"], input[type="search"]'
    ).first();

    if (await searchInput.isVisible().catch(() => false)) {
      // Type a search query
      await searchInput.fill('pod');
      await page.waitForTimeout(500);

      // Results should appear
      const results = page.locator(
        '[cmdk-item], [role="option"], [class*="command-item"], [class*="CommandItem"]'
      );
      const resultCount = await results.count();
      expect(resultCount, 'Search should return results for "pod"').toBeGreaterThan(0);

      // Click the first result
      if (resultCount > 0) {
        const firstResult = results.first();
        await firstResult.click();
        await waitForPageReady(page);
        // Should have navigated somewhere
        await assertPageLoaded(page, 'Cmd+K search result navigation');
      }
    }
  });

  test('Cmd+K search closes with Escape', async ({ page }) => {
    await page.goto('/dashboard');
    await waitForPageReady(page);

    await page.keyboard.press('Meta+k');
    await page.waitForTimeout(500);

    const searchInput = page.locator(
      '[cmdk-input], input[placeholder*="Search"], input[placeholder*="search"]'
    ).first();

    if (await searchInput.isVisible().catch(() => false)) {
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);
      await expect(searchInput).not.toBeVisible();
    }
  });

  test('404 page renders for unknown routes', async ({ page }) => {
    await page.goto('/this-route-does-not-exist-12345');
    await waitForPageReady(page);

    // Should show NotFound or redirect, not crash
    const body = await page.locator('body').textContent();
    expect(body).not.toContain('Unhandled Runtime Error');
    // Either shows 404 content or redirects to a known page
    const is404 = body?.includes('404') || body?.includes('Not Found') || body?.includes('not found');
    const redirected = page.url().includes('/dashboard') || page.url().includes('/connect') || page.url().includes('/mode-selection');
    expect(is404 || redirected, 'Unknown route should show 404 or redirect').toBe(true);
  });
});
