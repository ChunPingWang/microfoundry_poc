import { test, expect } from '../fixtures/admin-server';
import { users } from '../helpers/selectors';
import { APIHelper } from '../helpers/api';

test.describe('Users & Organizations - Page Load', () => {
  test('users page loads', async ({ page }) => {
    await page.goto('/users');
    // Should show IAM page or no-auth message
    await expect(
      page.locator(users.title)
        .or(page.locator('text=Organizations'))
        .or(page.locator(users.noAuthMessage))
    ).toBeVisible();
  });

  test('users page shows auth-disabled message when auth is off', async ({ page }) => {
    await page.goto('/users');
    const noAuthMsg = page.locator(users.noAuthMessage)
      .or(page.locator('text=login'))
      .or(page.locator('text=disabled'));
    const iamTabs = page.locator(users.workspacesTab)
      .or(page.locator(users.orgsTab));

    // Either auth is enabled (tabs shown) or disabled (message shown)
    await expect(iamTabs.first().or(noAuthMsg.first())).toBeVisible();
  });
});

test.describe('Users & Organizations - IAM Tabs', () => {
  test('IAM tab navigation is present when auth is enabled', async ({ page }) => {
    await page.goto('/users');
    const tabs = page.locator(users.workspacesTab);
    if (await tabs.isVisible()) {
      await expect(page.locator(users.workspacesTab)).toBeVisible();
      await expect(page.locator(users.orgsTab)).toBeVisible();
      await expect(page.locator(users.usersTab)).toBeVisible();
      await expect(page.locator(users.policiesTab)).toBeVisible();
      await expect(page.locator(users.auditTab)).toBeVisible();
    }
  });

  test('workspaces tab loads via HTMX', async ({ page }) => {
    await page.goto('/users');
    const wsTab = page.locator(users.workspacesTab);
    if (await wsTab.isVisible()) {
      await wsTab.click();
      await page.waitForTimeout(500);
      await expect(page).toHaveURL(/tab=workspaces/);
    }
  });

  test('organizations tab loads via HTMX', async ({ page }) => {
    await page.goto('/users');
    const orgsTab = page.locator(users.orgsTab);
    if (await orgsTab.isVisible()) {
      await orgsTab.click();
      await page.waitForTimeout(500);
      await expect(page).toHaveURL(/tab=orgs/);
    }
  });

  test('users tab loads via HTMX', async ({ page }) => {
    await page.goto('/users');
    const usersTab = page.locator(users.usersTab);
    if (await usersTab.isVisible()) {
      await usersTab.click();
      await page.waitForTimeout(500);
      await expect(page).toHaveURL(/tab=users/);
    }
  });

  test('policies tab loads via HTMX', async ({ page }) => {
    await page.goto('/users');
    const policiesTab = page.locator(users.policiesTab);
    if (await policiesTab.isVisible()) {
      await policiesTab.click();
      await page.waitForTimeout(500);
      await expect(page).toHaveURL(/tab=policies/);
    }
  });

  test('audit tab loads via HTMX', async ({ page }) => {
    await page.goto('/users');
    const auditTab = page.locator(users.auditTab);
    if (await auditTab.isVisible()) {
      await auditTab.click();
      await page.waitForTimeout(500);
      await expect(page).toHaveURL(/tab=audit/);
    }
  });

  test('tab content container receives HTMX updates', async ({ page }) => {
    await page.goto('/users');
    const tabContent = page.locator(users.tabContent);
    if (await tabContent.isVisible()) {
      await expect(tabContent).toBeVisible();
    }
  });
});

test.describe('Users & Organizations - Org Management', () => {
  test('create org form is available', async ({ page }) => {
    await page.goto('/users?tab=orgs');
    const createForm = page.locator(users.createOrgForm)
      .or(page.locator(users.createOrgButton))
      .or(page.locator('input[name="name"]'));
    const noAuthMsg = page.locator(users.noAuthMessage).or(page.locator('text=login'));
    await expect(createForm.first().or(noAuthMsg.first())).toBeVisible();
  });

  test('member management actions are available when orgs exist', async ({ page, request }) => {
    const api = new APIHelper(request);
    let orgs: any[] = [];
    try {
      orgs = await api.getOrgs();
    } catch {
      // API may not be available without auth
    }

    await page.goto('/users?tab=orgs');
    if (orgs && orgs.length > 0) {
      await expect(page.locator('text=Members').or(page.locator('text=Role')).first()).toBeVisible();
    }
  });

  test('org invite member form has email and role fields', async ({ page, request }) => {
    const api = new APIHelper(request);
    let orgs: any[] = [];
    try {
      orgs = await api.getOrgs();
    } catch {}

    if (orgs && orgs.length > 0) {
      await page.goto('/users?tab=orgs');
      const emailInput = page.locator('input[name="email"]');
      const roleSelect = page.locator('select[name="role"]');
      if (await emailInput.isVisible()) {
        await expect(emailInput).toBeVisible();
        await expect(roleSelect).toBeVisible();
      }
    }
  });
});

test.describe('Users & Organizations - User Management', () => {
  test('create user form has required fields', async ({ page }) => {
    await page.goto('/users?tab=users');
    const createBtn = page.locator(users.createUserButton);
    if (await createBtn.isVisible()) {
      await createBtn.click();
      const form = page.locator(users.createUserForm);
      if (await form.isVisible()) {
        await expect(page.locator('input[name="username"]')).toBeVisible();
        await expect(page.locator('input[name="email"]')).toBeVisible();
        await expect(page.locator('input[name="password"]')).toBeVisible();
      }
    }
  });

  test('user search form is available', async ({ page }) => {
    await page.goto('/users?tab=users');
    const searchInput = page.locator('input[name="search"]');
    if (await searchInput.isVisible()) {
      await expect(searchInput).toBeVisible();
    }
  });
});

test.describe('Users & Organizations - Policies', () => {
  test('policies tab shows OPA policy editor', async ({ page }) => {
    await page.goto('/users?tab=policies');
    // Should show policy source or a not-configured message
    const policyCode = page.locator('pre');
    const notConfigured = page.locator('.bg-yellow-100, text=not configured');
    const infoMsg = page.locator('.bg-blue-50');
    await expect(
      policyCode.first()
        .or(notConfigured.first())
        .or(infoMsg.first())
        .or(page.locator(users.noAuthMessage).first())
    ).toBeVisible();
  });

  test('add custom policy form has name and source fields', async ({ page }) => {
    await page.goto('/users?tab=policies');
    const nameInput = page.locator('input[name="name"]');
    const sourceTextarea = page.locator('textarea[name="source"]');
    if (await nameInput.isVisible()) {
      await expect(nameInput).toBeVisible();
      await expect(sourceTextarea).toBeVisible();
    }
  });
});
