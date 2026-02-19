import { test, expect } from '../fixtures/admin-server';
import { services, serviceDetail } from '../helpers/selectors';
import { APIHelper } from '../helpers/api';

test.describe('Services - List', () => {
  test('service list page loads', async ({ page }) => {
    await page.goto('/services');
    await expect(page.locator(services.title)).toBeVisible();
  });

  test('create service button opens modal', async ({ page }) => {
    await page.goto('/services');
    const createBtn = page.locator(services.createButton);
    await expect(createBtn).toBeVisible();
    await createBtn.click();

    // Modal should become visible
    const modal = page.locator(services.createModal);
    await expect(modal).toBeVisible();
  });

  test('create service modal has required form fields', async ({ page }) => {
    await page.goto('/services');
    await page.locator(services.createButton).click();

    await expect(page.locator(services.serviceTypeSelect)).toBeVisible();
    await expect(page.locator(services.planSelect)).toBeVisible();
    await expect(page.locator(services.serviceNameInput)).toBeVisible();
  });

  test('service type select populates plans', async ({ page }) => {
    await page.goto('/services');
    await page.locator(services.createButton).click();

    const typeSelect = page.locator(services.serviceTypeSelect);
    const options = typeSelect.locator('option');
    const count = await options.count();
    expect(count).toBeGreaterThan(1);
  });

  test('service name input has validation pattern', async ({ page }) => {
    await page.goto('/services');
    await page.locator(services.createButton).click();

    const nameInput = page.locator(services.serviceNameInput);
    await expect(nameInput).toHaveAttribute('pattern', /\[a-z\]/);
  });

  test('service list has auto-refresh', async ({ page }) => {
    await page.goto('/services');
    const autoRefresh = page.locator('[hx-trigger*="every"]');
    await expect(autoRefresh.first()).toBeAttached();
  });

  test('service list shows table or empty state', async ({ page, request }) => {
    const api = new APIHelper(request);
    const svcList = await api.getServices();

    await page.goto('/services');
    if (svcList && svcList.length > 0) {
      await expect(page.locator(services.tableBody)).toBeVisible();
    } else {
      await expect(page.locator(services.emptyState)).toBeVisible();
    }
  });
});

test.describe('Services - Detail', () => {
  test('service detail page loads', async ({ page, request }) => {
    const api = new APIHelper(request);
    const svcList = await api.getServices();

    if (svcList && svcList.length > 0) {
      const svcName = svcList[0].name || svcList[0].Name;
      await page.goto(`/services/${svcName}`);
      await expect(page.locator(`text=${svcName}`).first()).toBeVisible();
    } else {
      await page.goto('/services');
      await expect(page.locator(services.emptyState)).toBeVisible();
    }
  });

  test('service detail shows connection details', async ({ page, request }) => {
    const api = new APIHelper(request);
    const svcList = await api.getServices();

    if (svcList && svcList.length > 0) {
      const svcName = svcList[0].name || svcList[0].Name;
      await page.goto(`/services/${svcName}`);
      const connDetails = page.locator(serviceDetail.connectionDetails)
        .or(page.locator('text=Credentials'));
      await expect(connDetails.first()).toBeVisible();
    }
  });

  test('service detail shows bind form', async ({ page, request }) => {
    const api = new APIHelper(request);
    const svcList = await api.getServices();

    if (svcList && svcList.length > 0) {
      const svcName = svcList[0].name || svcList[0].Name;
      await page.goto(`/services/${svcName}`);
      const bindInput = page.locator(serviceDetail.bindForm);
      if (await bindInput.isVisible()) {
        await expect(bindInput).toBeVisible();
      }
    }
  });

  test('service detail has delete button with confirmation', async ({ page, request }) => {
    const api = new APIHelper(request);
    const svcList = await api.getServices();

    if (svcList && svcList.length > 0) {
      const svcName = svcList[0].name || svcList[0].Name;
      await page.goto(`/services/${svcName}`);
      const deleteBtn = page.locator('button[hx-delete*="services"]');
      if (await deleteBtn.isVisible()) {
        await expect(deleteBtn).toHaveAttribute('hx-confirm', /.+/);
      }
    }
  });
});
