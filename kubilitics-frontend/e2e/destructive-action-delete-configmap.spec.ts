/**
 * E2E: Destructive action actually completes (ConfigMap delete)
 *
 * Every pre-existing delete-related e2e spec (detail-page-actions.spec.ts,
 * buttons-actions-audit.spec.ts) opens the delete confirmation dialog and
 * clicks Cancel — none of them ever click through to a real deletion. This
 * spec closes that gap: it creates a disposable ConfigMap on a live cluster
 * via kubectl, deletes it through the real UI confirm flow (typing the
 * resource name + clicking the destructive Delete action), and verifies the
 * object is actually gone — both in the UI and via `kubectl get` against the
 * live API server — so an optimistic-UI bug can't masquerade as success.
 *
 * Live-cluster-gated, following the same pattern as
 * pod-terminal-live-backend.spec.ts: it test.skip()s cleanly (never blocks
 * CI) unless the required env vars are set.
 *
 * Required env vars to run live:
 *   PLAYWRIGHT_CLUSTER_ID          - cluster id seeded into kubilitics-cluster store
 *   PLAYWRIGHT_TEST_NAMESPACE      - namespace to create/delete the test ConfigMap in
 *   PLAYWRIGHT_BACKEND_BASE_URL    - backend base URL seeded into kubilitics-backend-config
 * Optional:
 *   PLAYWRIGHT_CLUSTER_NAME        - display name for the seeded cluster (default: playwright-cluster)
 *   PLAYWRIGHT_KUBECTL_CONTEXT     - kubectl --context to target (default: current context)
 */
import { test, expect } from '@playwright/test';
import { execFileSync } from 'node:child_process';

const clusterId = process.env.PLAYWRIGHT_CLUSTER_ID ?? '';
const clusterName = process.env.PLAYWRIGHT_CLUSTER_NAME ?? 'playwright-cluster';
const testNamespace = process.env.PLAYWRIGHT_TEST_NAMESPACE ?? '';
const backendBaseUrl = process.env.PLAYWRIGHT_BACKEND_BASE_URL ?? '';
const kubectlContext = process.env.PLAYWRIGHT_KUBECTL_CONTEXT ?? '';

const hasLiveDestructiveConfig = Boolean(clusterId && testNamespace && backendBaseUrl);

function kubectlArgs(...args: string[]): string[] {
  return kubectlContext ? ['--context', kubectlContext, ...args] : args;
}

function kubectlCreateConfigMap(name: string) {
  execFileSync(
    'kubectl',
    kubectlArgs(
      'create',
      'configmap',
      name,
      '-n',
      testNamespace,
      '--from-literal=marker=playwright-destructive-e2e'
    ),
    { stdio: 'pipe' }
  );
}

function kubectlDeleteConfigMap(name: string) {
  // Idempotent: --ignore-not-found means this never throws even if the
  // resource is already gone (the expected/successful case).
  execFileSync(
    'kubectl',
    kubectlArgs('delete', 'configmap', name, '-n', testNamespace, '--ignore-not-found=true', '--wait=false'),
    { stdio: 'pipe' }
  );
}

function kubectlGetConfigMapExists(name: string): boolean {
  try {
    execFileSync('kubectl', kubectlArgs('get', 'configmap', name, '-n', testNamespace), { stdio: 'pipe' });
    return true;
  } catch {
    // Non-zero exit = NotFound (or other error) — treat as "does not exist".
    return false;
  }
}

function buildActiveCluster() {
  return {
    id: clusterId,
    name: clusterName,
    context: clusterName,
    version: 'v1.29.0',
    status: 'healthy',
    region: 'unknown',
    provider: 'on-prem',
    nodes: 1,
    namespaces: 1,
    pods: { running: 1, pending: 0, failed: 0 },
    cpu: { used: 0, total: 100 },
    memory: { used: 0, total: 100 },
  };
}

test.describe('destructive action — ConfigMap delete completes for real', () => {
  test.skip(
    !hasLiveDestructiveConfig,
    'Set PLAYWRIGHT_CLUSTER_ID, PLAYWRIGHT_TEST_NAMESPACE, PLAYWRIGHT_BACKEND_BASE_URL to run the live destructive-delete test.'
  );

  // Tracks the resource created by the current test so afterEach can force-clean it
  // even if the test fails before the UI delete completes.
  let pendingConfigMapName: string | null = null;

  test.beforeEach(async ({ page }) => {
    const activeCluster = buildActiveCluster();
    await page.addInitScript(
      ({ seededCluster, seededBackendBaseUrl }) => {
        localStorage.setItem(
          'kubilitics-cluster',
          JSON.stringify({
            state: {
              clusters: [seededCluster],
              activeCluster: seededCluster,
              activeNamespace: 'all',
              namespaces: [],
              isDemo: false,
              appMode: 'desktop',
              isOnboarded: true,
            },
            version: 0,
          })
        );
        localStorage.setItem(
          'kubilitics-backend-config',
          JSON.stringify({
            state: {
              backendBaseUrl: seededBackendBaseUrl,
              currentClusterId: seededCluster.id,
            },
            version: 0,
          })
        );
      },
      { seededCluster: activeCluster, seededBackendBaseUrl: backendBaseUrl }
    );
  });

  test.afterEach(() => {
    // Force-cleanup in case the test failed before the UI delete completed.
    // Idempotent via --ignore-not-found — never throws when already gone.
    if (!pendingConfigMapName) return;
    try {
      kubectlDeleteConfigMap(pendingConfigMapName);
    } catch {
      // Best-effort cleanup; swallow so afterEach never fails the run.
    }
    pendingConfigMapName = null;
  });

  test('deletes a real ConfigMap through the UI confirm flow, verified via kubectl', async ({ page }) => {
    const configMapName = `playwright-delete-test-${Date.now()}`;
    pendingConfigMapName = configMapName;

    // 1. Seed a disposable real resource on the live cluster.
    kubectlCreateConfigMap(configMapName);
    expect(
      kubectlGetConfigMapExists(configMapName),
      'Seeded ConfigMap should exist on the live cluster before the test runs'
    ).toBe(true);

    // 2. Navigate the real UI to that resource's detail page.
    await page.goto(`/configmaps/${encodeURIComponent(testNamespace)}/${encodeURIComponent(configMapName)}`);
    await page.waitForLoadState('load');
    await expect(page.getByText(configMapName).first()).toBeVisible({ timeout: 15000 });

    // 3. Open the delete confirmation dialog.
    const deleteBtn = page.locator('button:has-text("Delete")').first();
    await expect(deleteBtn, 'Delete button should be visible on ConfigMap detail page').toBeVisible({
      timeout: 10000,
    });
    await deleteBtn.click();

    const dialog = page.locator('[role="alertdialog"]:visible, [role="dialog"]:visible').first();
    await expect(dialog, 'Delete confirmation dialog should open').toBeVisible({ timeout: 5000 });

    // ConfigMapDetail requires typing the resource name to confirm.
    const nameInput = dialog.locator('#confirm-name, input[placeholder]');
    if (await nameInput.isVisible().catch(() => false)) {
      await nameInput.fill(configMapName);
    }

    // 4. Click through to completion — Confirm/Delete, NOT Cancel.
    const confirmDeleteBtn = dialog.getByRole('button', { name: /^delete$/i });
    await expect(confirmDeleteBtn, 'Confirm Delete action should be enabled once name is confirmed').toBeEnabled({
      timeout: 5000,
    });
    await confirmDeleteBtn.click();

    // 5a. UI proof: success toast appears.
    await expect(page.locator('[data-sonner-toast]').filter({ hasText: /deleted/i }).first()).toBeVisible({
      timeout: 15000,
    });

    // 5b. UI proof: navigated away from the (now-deleted) detail page, and the
    // resource no longer appears in the list.
    await page.waitForURL(/\/configmaps(?:\?.*)?$/, { timeout: 15000 });
    await expect(page.getByText(configMapName)).toHaveCount(0, { timeout: 15000 });

    // 6. Ground-truth proof: this was a REAL deletion, not an optimistic UI
    // update — assert directly against the live API server via kubectl.
    await expect
      .poll(() => kubectlGetConfigMapExists(configMapName), {
        message: 'kubectl get configmap should report NotFound after the UI delete completes',
        timeout: 15000,
      })
      .toBe(false);

    // Deletion confirmed for real — nothing left for afterEach to clean up.
    pendingConfigMapName = null;
  });
});
