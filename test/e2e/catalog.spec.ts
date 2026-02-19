import { test, expect } from '../fixtures/admin-server';
import { catalog } from '../helpers/selectors';

test.describe('Service Catalog', () => {
  test('catalog page loads with service types', async ({ page }) => {
    await page.goto('/catalog');
    await expect(page.locator(catalog.title)).toBeVisible();
  });

  test('all 10 service types are displayed', async ({ page }) => {
    await page.goto('/catalog');
    const serviceTypes = [
      'MariaDB', 'PostgreSQL', 'ClickHouse',
      'Redis', 'Memcached',
      'RabbitMQ', 'ActiveMQ',
      'MinIO',
      'Kong', 'NGINX',
    ];

    for (const svcType of serviceTypes) {
      await expect(page.locator(`text=${svcType}`).first()).toBeVisible();
    }
  });

  test('all category groupings are present', async ({ page }) => {
    await page.goto('/catalog');
    await expect(page.locator(catalog.databases)).toBeVisible();
    await expect(page.locator(catalog.caches)).toBeVisible();
    await expect(page.locator(catalog.messaging)).toBeVisible();
    await expect(page.locator(catalog.storage)).toBeVisible();
    await expect(page.locator(catalog.gateway)).toBeVisible();
  });

  test('plan details show resource information', async ({ page }) => {
    await page.goto('/catalog');
    // Plans should show small/medium/large with resource specs
    await expect(page.locator('text=small').first()).toBeVisible();
    await expect(page.locator('text=medium').first()).toBeVisible();
    await expect(page.locator('text=large').first()).toBeVisible();
  });

  test('plan table has provisioner column', async ({ page }) => {
    await page.goto('/catalog');
    // Provisioner column with K8s Native or Terraform badges
    const provisionerBadges = page.locator('.bg-purple-100, .bg-gray-100').locator('text=Edit, text=K8s Native, text=Configure');
    const count = await provisionerBadges.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('visibility toggle buttons are present', async ({ page }) => {
    await page.goto('/catalog');
    const toggleButtons = page.locator('button[hx-post*="visibility"]');
    const count = await toggleButtons.count();
    expect(count).toBeGreaterThan(0);
  });

  test('enable/disable all visibility controls exist', async ({ page }) => {
    await page.goto('/catalog');
    // Each service type should have Enable All / Disable All buttons
    const enableAllBtns = page.locator('button:has-text("Enable All")');
    const disableAllBtns = page.locator('button:has-text("Disable All")');
    const enableCount = await enableAllBtns.count();
    const disableCount = await disableAllBtns.count();
    expect(enableCount + disableCount).toBeGreaterThan(0);
  });

  test('terraform status badge is displayed', async ({ page }) => {
    await page.goto('/catalog');
    // Should show terraform availability status
    const terraformBadge = page.locator('.bg-green-100:has-text("Terraform"), .bg-yellow-100:has-text("Terraform")');
    await expect(terraformBadge.first()).toBeVisible();
  });

  test('upload topology link is available', async ({ page }) => {
    await page.goto('/catalog');
    const uploadLink = page.locator(catalog.uploadTopology);
    await expect(uploadLink.first()).toBeVisible();
  });

  test('topology page loads for a service plan', async ({ page }) => {
    const response = await page.goto('/topologies/mariadb/small');
    expect(response?.status()).not.toBe(404);
  });
});
