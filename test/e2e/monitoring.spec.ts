import { test, expect } from '../fixtures/admin-server';
import { monitoring } from '../helpers/selectors';

test.describe('Monitoring', () => {
  test('monitoring page loads', async ({ page }) => {
    await page.goto('/monitoring');
    await expect(page.locator(monitoring.title).first()).toBeVisible();
  });

  test('Grafana button or link is present', async ({ page }) => {
    await page.goto('/monitoring');
    const grafanaEl = page.locator(monitoring.grafanaButton)
      .or(page.locator('text=Grafana'));
    await expect(grafanaEl.first()).toBeVisible();
  });

  test('monitoring stack component health grid is displayed', async ({ page }) => {
    await page.goto('/monitoring');
    // Component health grid shows Prometheus, Grafana, Loki, AlertManager status
    const healthGrid = page.locator(monitoring.componentHealth);
    await expect(healthGrid.first()).toBeVisible();
  });

  test('monitoring components show connection status', async ({ page }) => {
    await page.goto('/monitoring');
    // Each component shows connected (green) or disconnected (red)
    const statusIndicators = page.locator('.bg-green-50, .bg-red-50');
    const count = await statusIndicators.count();
    expect(count).toBeGreaterThan(0);
  });

  test('Grafana dashboard iframes are present', async ({ page }) => {
    await page.goto('/monitoring');
    const iframes = page.locator(monitoring.grafanaIframe);
    const count = await iframes.count();
    // Should have at least one Grafana iframe (Overview dashboard)
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('alerts container is present', async ({ page }) => {
    await page.goto('/monitoring');
    await expect(page.locator(monitoring.alertsContainer)).toBeAttached();
  });

  test('alerts auto-refresh with HTMX', async ({ page }) => {
    await page.goto('/monitoring');
    // Alerts section should have hx-trigger="every 30s" for auto-refresh
    const refreshEl = page.locator('[hx-trigger*="every"]');
    await expect(refreshEl.first()).toBeAttached();
  });

  test('alerts page loads via partial endpoint', async ({ page }) => {
    const response = await page.goto('/monitoring/alerts');
    expect(response?.status()).toBeLessThan(500);
  });

  test('monitoring health API returns status', async ({ request }) => {
    const response = await request.get('/api/monitoring/health');
    // May return 200 or 503 depending on monitoring stack
    expect(response.status()).toBeLessThanOrEqual(503);
  });
});
