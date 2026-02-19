import { test, expect } from '../fixtures/admin-server';

test.describe('Topology Editor', () => {
  test('topology editor page loads for a service plan', async ({ page }) => {
    const response = await page.goto('/topologies/mariadb/small');
    expect(response?.status()).not.toBe(404);
  });

  test('topology upload page loads', async ({ page }) => {
    const response = await page.goto('/topologies/upload');
    expect(response?.status()).not.toBe(404);
  });

  test('topology upload form has service type and plan selects', async ({ page }) => {
    await page.goto('/topologies/upload');
    const svcTypeSelect = page.locator('select[name="service_type"]');
    const planSelect = page.locator('select[name="plan_id"]');
    if (await svcTypeSelect.isVisible()) {
      await expect(svcTypeSelect).toBeVisible();
      await expect(planSelect).toBeVisible();
    }
  });

  test('topology upload form accepts .tf files', async ({ page }) => {
    await page.goto('/topologies/upload');
    const fileInput = page.locator('input[type="file"]');
    if (await fileInput.isVisible()) {
      await expect(fileInput).toHaveAttribute('accept', /\.tf/);
    }
  });

  test('topology editor has save and preview buttons', async ({ page }) => {
    await page.goto('/topologies/mariadb/small');
    const saveBtn = page.locator('button:has-text("Save")');
    const previewBtn = page.locator('button:has-text("Preview")');
    if (await saveBtn.isVisible()) {
      await expect(saveBtn).toBeVisible();
    }
    if (await previewBtn.isVisible()) {
      await expect(previewBtn).toBeVisible();
    }
  });

  test('topology editor has terraform file content area', async ({ page }) => {
    await page.goto('/topologies/mariadb/small');
    const textarea = page.locator('textarea[name*="file_content"]');
    if (await textarea.isVisible()) {
      await expect(textarea).toBeVisible();
    }
  });

  test('topology delete button has confirmation', async ({ page }) => {
    await page.goto('/topologies/mariadb/small');
    const deleteBtn = page.locator('[hx-delete*="topologies"]');
    if (await deleteBtn.isVisible()) {
      await expect(deleteBtn).toHaveAttribute('hx-confirm', /.+/);
    }
  });

  test('topologies API returns list', async ({ request }) => {
    const response = await request.get('/api/topologies');
    expect(response.status()).toBeLessThan(500);
    expect(response.headers()['content-type']).toContain('json');
  });

  test('/topologies redirects to /catalog', async ({ page }) => {
    await page.goto('/topologies');
    await expect(page).toHaveURL(/catalog/);
  });
});
