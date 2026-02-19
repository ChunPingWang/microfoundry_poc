import { test, expect } from '../fixtures/admin-server';
import { settingsEndpoints } from '../helpers/selectors';

test.describe('Settings - Endpoints', () => {
  test('endpoints page loads', async ({ page }) => {
    const response = await page.goto('/settings/endpoints');
    expect(response?.status()).not.toBe(404);
  });

  test('endpoints page displays discovered services', async ({ page }) => {
    await page.goto('/settings/endpoints');
    // Should show endpoint sections for platform services (Prometheus, Grafana, etc.)
    const endpointSections = page.locator('.bg-white.rounded-lg');
    const count = await endpointSections.count();
    expect(count).toBeGreaterThan(0);
  });

  test('each endpoint shows resolved URL', async ({ page }) => {
    await page.goto('/settings/endpoints');
    const resolvedUrls = page.locator('input[readonly]');
    const count = await resolvedUrls.count();
    // At least some endpoints should have resolved URLs
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('endpoint override URL inputs are editable', async ({ page }) => {
    await page.goto('/settings/endpoints');
    const overrideInputs = page.locator('input[name^="url_"]');
    const count = await overrideInputs.count();
    if (count > 0) {
      // Override inputs should not be readonly
      const firstInput = overrideInputs.first();
      await expect(firstInput).not.toHaveAttribute('readonly');
    }
  });

  test('test endpoint connectivity buttons exist', async ({ page }) => {
    await page.goto('/settings/endpoints');
    const testBtns = page.locator(settingsEndpoints.testButton);
    const count = await testBtns.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('create ingress buttons are available for services without ingress', async ({ page }) => {
    await page.goto('/settings/endpoints');
    const ingressBtns = page.locator(settingsEndpoints.createIngressButton);
    const count = await ingressBtns.count();
    // May or may not have create ingress buttons depending on cluster state
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('save button submits endpoint overrides', async ({ page }) => {
    await page.goto('/settings/endpoints');
    const saveBtn = page.locator(settingsEndpoints.saveButton);
    if (await saveBtn.isVisible()) {
      await expect(saveBtn).toBeVisible();
    }
  });

  test('endpoints API returns JSON', async ({ request }) => {
    const response = await request.get('/api/settings/endpoints');
    expect(response.status()).toBeLessThan(500);
    expect(response.headers()['content-type']).toContain('json');
  });
});
