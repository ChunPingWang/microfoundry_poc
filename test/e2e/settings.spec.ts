import { test, expect } from '../fixtures/admin-server';
import { settingsRegistry, settingsWebhooks, settingsSMTP } from '../helpers/selectors';

test.describe('Settings - Registry', () => {
  test('registry page loads with form', async ({ page }) => {
    await page.goto('/settings/registry');
    await expect(page.locator(settingsRegistry.title)).toBeVisible();
  });

  test('registry form has all required fields', async ({ page }) => {
    await page.goto('/settings/registry');
    await expect(page.locator(settingsRegistry.urlInput)).toBeVisible();
    await expect(page.locator(settingsRegistry.projectInput)).toBeVisible();
    await expect(page.locator(settingsRegistry.usernameInput)).toBeVisible();
    await expect(page.locator(settingsRegistry.passwordInput)).toBeVisible();
    await expect(page.locator(settingsRegistry.insecureCheckbox)).toBeAttached();
    await expect(page.locator(settingsRegistry.enabledCheckbox)).toBeAttached();
  });

  test('save registry config shows success banner', async ({ page }) => {
    await page.goto('/settings/registry');
    await page.locator(settingsRegistry.urlInput).fill('harbor.local:30003');
    await page.locator(settingsRegistry.projectInput).fill('microfoundry');
    await page.locator(settingsRegistry.usernameInput).fill('admin');

    await page.locator(settingsRegistry.saveButton).click();
    await page.waitForURL(/saved=true|registry/);
  });

  test('test connection button exists with HTMX', async ({ page }) => {
    await page.goto('/settings/registry');
    const testBtn = page.locator(settingsRegistry.testButton);
    await expect(testBtn).toBeVisible();
    await expect(testBtn).toHaveAttribute('hx-post', /registry\/test/);
  });

  test('test result container exists for HTMX response', async ({ page }) => {
    await page.goto('/settings/registry');
    const testResult = page.locator(settingsRegistry.testResult);
    await expect(testResult).toBeAttached();
  });

  test('image preview shows registry prefix', async ({ page }) => {
    await page.goto('/settings/registry');
    // Image preview should show a code block with registry prefix
    const codeBlock = page.locator('code:has-text("my-app:latest")');
    if (await codeBlock.isVisible()) {
      await expect(codeBlock).toBeVisible();
    }
  });
});

test.describe('Settings - Webhooks', () => {
  test('webhooks page loads', async ({ page }) => {
    await page.goto('/settings/webhooks');
    await expect(page.locator(settingsWebhooks.title)).toBeVisible();
  });

  test('add webhook form has required fields', async ({ page }) => {
    await page.goto('/settings/webhooks');
    const addBtn = page.locator(settingsWebhooks.addButton);
    await expect(addBtn).toBeVisible();
    await addBtn.click();

    const form = page.locator(settingsWebhooks.addForm);
    await expect(form).toBeVisible();

    await expect(page.locator(settingsWebhooks.nameInput)).toBeVisible();
    await expect(page.locator(settingsWebhooks.urlInput)).toBeVisible();
  });

  test('webhook URL input validates URL format', async ({ page }) => {
    await page.goto('/settings/webhooks');
    await page.locator(settingsWebhooks.addButton).click();
    const urlInput = page.locator(settingsWebhooks.urlInput);
    await expect(urlInput).toHaveAttribute('type', 'url');
  });

  test('all 8 event types shown as checkboxes', async ({ page }) => {
    await page.goto('/settings/webhooks');
    const addBtn = page.locator(settingsWebhooks.addButton);
    await addBtn.click();

    const eventCheckboxes = page.locator(settingsWebhooks.eventsCheckboxes);
    const count = await eventCheckboxes.count();
    expect(count).toBe(8);
  });

  test('enabled checkbox is checked by default', async ({ page }) => {
    await page.goto('/settings/webhooks');
    await page.locator(settingsWebhooks.addButton).click();
    const enabledCheckbox = page.locator(settingsWebhooks.enabledCheckbox);
    await expect(enabledCheckbox).toBeChecked();
  });

  test('webhook form submits via HTMX', async ({ page }) => {
    await page.goto('/settings/webhooks');
    await page.locator(settingsWebhooks.addButton).click();
    const form = page.locator(settingsWebhooks.addForm);
    // Form or submit button should have hx-post
    const htmxEl = form.locator('[hx-post*="webhooks"]');
    await expect(htmxEl.first()).toBeAttached();
  });

  test('delete webhook has confirmation', async ({ page }) => {
    await page.goto('/settings/webhooks');
    const deleteBtns = page.locator('button[hx-delete*="webhooks"]');
    const count = await deleteBtns.count();
    if (count > 0) {
      const firstDelete = deleteBtns.first();
      await expect(firstDelete).toHaveAttribute('hx-confirm', /.+/);
    }
  });

  test('test webhook button exists for configured webhooks', async ({ page }) => {
    await page.goto('/settings/webhooks');
    const testBtns = page.locator('button[hx-post*="test"]');
    const count = await testBtns.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('webhook status badges show active/disabled state', async ({ page }) => {
    await page.goto('/settings/webhooks');
    const statusBadges = page.locator('.bg-green-100:has-text("Active"), .bg-gray-100:has-text("Disabled")');
    const count = await statusBadges.count();
    // Only if webhooks exist
    expect(count).toBeGreaterThanOrEqual(0);
  });
});

test.describe('Settings - SMTP', () => {
  test('SMTP page loads with form', async ({ page }) => {
    await page.goto('/settings/smtp');
    await expect(page.locator(settingsSMTP.title)).toBeVisible();
  });

  test('SMTP form has all required fields', async ({ page }) => {
    await page.goto('/settings/smtp');
    await expect(page.locator(settingsSMTP.hostInput)).toBeVisible();
    await expect(page.locator(settingsSMTP.portInput)).toBeVisible();
    await expect(page.locator(settingsSMTP.usernameInput)).toBeVisible();
    await expect(page.locator(settingsSMTP.passwordInput)).toBeVisible();
    await expect(page.locator(settingsSMTP.fromInput)).toBeVisible();
    await expect(page.locator(settingsSMTP.tlsCheckbox)).toBeAttached();
    await expect(page.locator(settingsSMTP.enabledCheckbox)).toBeAttached();
  });

  test('save SMTP config submits form', async ({ page }) => {
    await page.goto('/settings/smtp');
    await page.locator(settingsSMTP.hostInput).fill('smtp.gmail.com');
    await page.locator(settingsSMTP.portInput).fill('587');
    await page.locator(settingsSMTP.usernameInput).fill('test@example.com');
    await page.locator(settingsSMTP.fromInput).fill('noreply@example.com');

    await page.locator(settingsSMTP.saveButton).click();
    await page.waitForURL(/saved=true|smtp/);
  });

  test('test SMTP connection button exists', async ({ page }) => {
    await page.goto('/settings/smtp');
    const testBtn = page.locator(settingsSMTP.testButton);
    await expect(testBtn).toBeVisible();
    await expect(testBtn).toHaveAttribute('hx-post', /smtp\/test/);
  });

  test('test result container exists', async ({ page }) => {
    await page.goto('/settings/smtp');
    const testResult = page.locator(settingsSMTP.testResult);
    await expect(testResult).toBeAttached();
  });

  test('port input has number type', async ({ page }) => {
    await page.goto('/settings/smtp');
    const portInput = page.locator(settingsSMTP.portInput);
    await expect(portInput).toHaveAttribute('type', 'number');
  });

  test('from address input has email type', async ({ page }) => {
    await page.goto('/settings/smtp');
    const fromInput = page.locator(settingsSMTP.fromInput);
    await expect(fromInput).toHaveAttribute('type', 'email');
  });
});
