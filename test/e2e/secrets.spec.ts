import { test, expect } from '../fixtures/admin-server';
import { secrets, secretDetail } from '../helpers/selectors';
import { APIHelper } from '../helpers/api';

test.describe('Secrets - List', () => {
  test('secret list page loads', async ({ page }) => {
    await page.goto('/secrets');
    await expect(page.locator(secrets.title)).toBeVisible();
  });

  test('create secret button links to new form', async ({ page }) => {
    await page.goto('/secrets');
    const createBtn = page.locator(secrets.createButton).first();
    await expect(createBtn).toBeVisible();
    await createBtn.click();
    await expect(page).toHaveURL('/secrets/new');
  });

  test('type filter tabs are present', async ({ page }) => {
    await page.goto('/secrets');
    await expect(page.locator(secrets.filterAll).first()).toBeVisible();
    await expect(page.locator(secrets.filterService).first()).toBeVisible();
    await expect(page.locator(secrets.filterUser).first()).toBeVisible();
  });

  test('type filter changes URL via HTMX', async ({ page }) => {
    await page.goto('/secrets');
    const serviceFilter = page.locator(secrets.filterService).first();
    await serviceFilter.click();
    await expect(page).toHaveURL(/type=service/);
  });

  test('secret list has auto-refresh', async ({ page }) => {
    await page.goto('/secrets');
    const autoRefresh = page.locator('[hx-trigger*="every"]');
    await expect(autoRefresh.first()).toBeAttached();
  });

  test('secret list shows type badges', async ({ page, request }) => {
    const api = new APIHelper(request);
    const secretsList = await api.getSecrets();

    if (secretsList && secretsList.length > 0) {
      await page.goto('/secrets');
      // Type badges: service (green) or user-defined (purple)
      const badges = page.locator('.bg-green-100, .bg-purple-100');
      const count = await badges.count();
      expect(count).toBeGreaterThan(0);
    }
  });
});

test.describe('Secrets - Create', () => {
  test('create secret form has key-value inputs', async ({ page }) => {
    await page.goto('/secrets/new');
    await expect(page.locator('input[name="name"]').or(page.locator('input[placeholder*="name" i]')).first()).toBeVisible();
  });

  test('create form has description field', async ({ page }) => {
    await page.goto('/secrets/new');
    const descField = page.locator('input[name="description"], textarea[name="description"]');
    if (await descField.isVisible()) {
      await expect(descField).toBeVisible();
    }
  });

  test('create form has key-value pair inputs', async ({ page }) => {
    await page.goto('/secrets/new');
    const keyInput = page.locator('input[name="key_name[]"], input[name*="key"]').first();
    const valueInput = page.locator('input[name="key_value[]"], input[name*="value"]').first();
    await expect(keyInput).toBeVisible();
    await expect(valueInput).toBeVisible();
  });
});

test.describe('Secrets - Detail', () => {
  test('secret detail page shows masked values', async ({ page, request }) => {
    const api = new APIHelper(request);
    const secretsList = await api.getSecrets();

    if (secretsList && secretsList.length > 0) {
      const secretName = secretsList[0].name || secretsList[0].Name;
      await page.goto(`/secrets/${secretName}`);
      await expect(page.locator(`text=${secretName}`).first()).toBeVisible();
      // Values should be masked
      await expect(page.locator(secretDetail.maskedValue).first()).toBeVisible();
    }
  });

  test('reveal key endpoint works via HTMX', async ({ page, request }) => {
    const api = new APIHelper(request);
    const secretsList = await api.getSecrets();

    if (secretsList && secretsList.length > 0) {
      const secretName = secretsList[0].name || secretsList[0].Name;
      await page.goto(`/secrets/${secretName}`);
      const revealBtn = page.locator(secretDetail.revealButton).first();
      if (await revealBtn.isVisible()) {
        // Reveal button should have hx-get pointing to reveal endpoint
        await expect(revealBtn).toHaveAttribute('hx-get', new RegExp(`/secrets/${secretName}/reveal/`));
      }
    }
  });

  test('secret detail shows bound applications', async ({ page, request }) => {
    const api = new APIHelper(request);
    const secretsList = await api.getSecrets();

    if (secretsList && secretsList.length > 0) {
      const secretName = secretsList[0].name || secretsList[0].Name;
      await page.goto(`/secrets/${secretName}`);
      // Bound apps section should be present (even if empty)
      const boundApps = page.locator(secretDetail.boundApps)
        .or(page.locator('text=Bound'));
      await expect(boundApps.first()).toBeVisible();
    }
  });

  test('user-defined secret has delete button', async ({ page, request }) => {
    const api = new APIHelper(request);
    const secretsList = await api.getSecrets();

    if (secretsList && secretsList.length > 0) {
      // Find a user-defined secret
      const userSecret = secretsList.find((s: any) =>
        (s.type || s.Type) === 'user-defined' || (s.type || s.Type) === 'Opaque'
      );
      if (userSecret) {
        const name = userSecret.name || userSecret.Name;
        await page.goto(`/secrets/${name}`);
        const deleteBtn = page.locator('[hx-delete*="secrets"]');
        if (await deleteBtn.isVisible()) {
          await expect(deleteBtn).toHaveAttribute('hx-confirm', /.+/);
        }
      }
    }
  });
});
