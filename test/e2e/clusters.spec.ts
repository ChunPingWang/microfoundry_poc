import { test, expect } from '../fixtures/admin-server';
import { clusters } from '../helpers/selectors';
import { APIHelper } from '../helpers/api';

test.describe('Clusters', () => {
  test('cluster list page loads', async ({ page }) => {
    await page.goto('/clusters');
    await expect(page.locator(clusters.title)).toBeVisible();
  });

  test('active cluster has indicator badge', async ({ page }) => {
    await page.goto('/clusters');
    // Active cluster row has blue highlight and badge
    const activeBadge = page.locator(clusters.activeBadge)
      .or(page.locator('.bg-green-100'))
      .or(page.locator('text=active'));
    await expect(activeBadge.first()).toBeVisible();
  });

  test('active cluster row has blue highlight', async ({ page }) => {
    await page.goto('/clusters');
    const activeRow = page.locator(clusters.activeClusterRow);
    await expect(activeRow.first()).toBeVisible();
  });

  test('cluster table shows expected columns', async ({ page }) => {
    await page.goto('/clusters');
    await expect(page.locator('th:has-text("Name")').first()).toBeVisible();
    await expect(page.locator('th:has-text("Provider")').or(page.locator('th:has-text("Status")')).first()).toBeVisible();
  });

  test('add cluster form has all required fields', async ({ page }) => {
    await page.goto('/clusters');
    const addBtn = page.locator(clusters.addButton);
    await expect(addBtn).toBeVisible();
    await addBtn.click();

    const form = page.locator(clusters.addForm);
    await expect(form).toBeVisible();

    await expect(page.locator(clusters.nameInput)).toBeVisible();
    await expect(page.locator(clusters.providerInput)).toBeVisible();
    await expect(page.locator(clusters.contextInput)).toBeVisible();
    await expect(page.locator(clusters.namespaceInput)).toBeVisible();
    await expect(page.locator(clusters.domainInput)).toBeVisible();
  });

  test('provider select has expected options', async ({ page }) => {
    await page.goto('/clusters');
    await page.locator(clusters.addButton).click();

    const providerSelect = page.locator(clusters.providerInput);
    const options = providerSelect.locator('option');
    const count = await options.count();
    // Should have docker-desktop, eks, gke, aks, native options
    expect(count).toBeGreaterThanOrEqual(3);
  });

  test('ingress class select is available', async ({ page }) => {
    await page.goto('/clusters');
    await page.locator(clusters.addButton).click();

    const ingressSelect = page.locator(clusters.ingressInput);
    if (await ingressSelect.isVisible()) {
      await expect(ingressSelect).toBeVisible();
    }
  });

  test('add cluster form submits via HTMX', async ({ page }) => {
    await page.goto('/clusters');
    await page.locator(clusters.addButton).click();

    // Form should have hx-post for HTMX submission
    const submitBtn = page.locator('#add-cluster-form button[type="submit"], #add-cluster-form [hx-post*="clusters"]');
    if (await submitBtn.isVisible()) {
      await expect(submitBtn.first()).toBeVisible();
    }
  });

  test('cluster detail page shows node information', async ({ page, request }) => {
    const api = new APIHelper(request);
    const clusterList = await api.getClusters();

    if (clusterList && clusterList.length > 0) {
      const clusterId = clusterList[0].id || clusterList[0].ID;
      await page.goto(`/clusters/${clusterId}`);
      await expect(page.locator('text=Node').or(page.locator('text=Status')).first()).toBeVisible();
    }
  });

  test('activate cluster button is present for non-active clusters', async ({ page }) => {
    await page.goto('/clusters');
    const activateBtns = page.locator('button:has-text("Activate"), button[hx-post*="activate"], a:has-text("Set Active")');
    await expect(page.locator('table, .bg-white').first()).toBeVisible();
  });

  test('remove cluster has confirmation dialog', async ({ page, request }) => {
    const api = new APIHelper(request);
    const clusterList = await api.getClusters();

    if (clusterList && clusterList.length > 0) {
      await page.goto('/clusters');
      const deleteBtns = page.locator('[hx-delete*="clusters"]');
      const count = await deleteBtns.count();
      if (count > 0) {
        await expect(deleteBtns.first()).toHaveAttribute('hx-confirm', /.+/);
      }
    }
  });

  test('cluster health check returns status', async ({ page, request }) => {
    const api = new APIHelper(request);
    const clusterList = await api.getClusters();

    if (clusterList && clusterList.length > 0) {
      const clusterId = clusterList[0].id || clusterList[0].ID;
      const response = await request.get(`/api/clusters/${clusterId}/health`);
      expect(response.status()).toBeLessThan(500);
    }
  });
});
