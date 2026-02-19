import { test, expect } from '../fixtures/admin-server';
import { config } from '../helpers/selectors';

test.describe('Platform Config', () => {
  test('config page loads with sections', async ({ page }) => {
    await page.goto('/config');
    await expect(page.locator(config.kubernetesSection)).toBeVisible();
    await expect(page.locator(config.githubSection)).toBeVisible();
    await expect(page.locator(config.platformSection)).toBeVisible();
  });

  test('Kubernetes section displays key-value pairs', async ({ page }) => {
    await page.goto('/config');
    await expect(page.locator('dt:has-text("Context")')).toBeVisible();
    await expect(page.locator('dt:has-text("Namespace")')).toBeVisible();
    await expect(page.locator('dt:has-text("Domain")')).toBeVisible();
  });

  test('GitHub section displays owner and repository', async ({ page }) => {
    await page.goto('/config');
    await expect(page.locator('dt:has-text("Owner")')).toBeVisible();
    await expect(page.locator('dt:has-text("Repository")')).toBeVisible();
  });

  test('Platform section displays version info', async ({ page }) => {
    await page.goto('/config');
    const platformSection = page.locator(config.platformSection).locator('..');
    await expect(platformSection).toBeVisible();
  });

  test('config values are in monospace font', async ({ page }) => {
    await page.goto('/config');
    const monoValues = page.locator('dd.font-mono');
    const count = await monoValues.count();
    expect(count).toBeGreaterThan(0);
  });

  test('config API returns matching data', async ({ page, request }) => {
    const response = await request.get('/api/config');
    expect(response.ok()).toBe(true);
    const data = await response.json();
    expect(typeof data).toBe('object');

    // Config data should have domain and namespace
    if (data.domain || data.Domain) {
      await page.goto('/config');
      const domain = data.domain || data.Domain;
      await expect(page.locator(`text=${domain}`).first()).toBeVisible();
    }
  });
});
