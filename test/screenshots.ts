import { chromium, Browser } from '@playwright/test';
import * as path from 'path';
import * as fs from 'fs';

const BASE_URL = 'https://admin.cf-local.dev:8443';
const OUTPUT_DIR = path.resolve(__dirname, '../docs/images');

// Keycloak credentials (platform-admin)
const KC_USER = 'myadmin';
const KC_PASS = 'admin';

// All pages to screenshot (super admin view — full menu)
const pages = [
  // Operations
  { name: 'dashboard',          url: '/',                      title: 'Dashboard' },
  { name: 'apps-list',          url: '/apps',                  title: 'Applications' },
  { name: 'services',           url: '/services',              title: 'Services' },
  { name: 'secrets',            url: '/secrets',               title: 'Secrets' },
  // Settings
  { name: 'users-iam',          url: '/users',                 title: 'Users & IAM' },
  { name: 'workspaces',         url: '/workspaces',            title: 'Workspaces' },
  { name: 'clusters',           url: '/clusters',              title: 'Clusters' },
  { name: 'catalog',            url: '/catalog',               title: 'Service Catalog' },
  { name: 'settings-registry',  url: '/settings/registry',     title: 'Registry Settings' },
  { name: 'settings-webhooks',  url: '/settings/webhooks',     title: 'Webhook Settings' },
  { name: 'settings-smtp',      url: '/settings/smtp',         title: 'SMTP Settings' },
  { name: 'settings-endpoints', url: '/settings/endpoints',    title: 'Endpoint Settings' },
  { name: 'monitoring',         url: '/monitoring',            title: 'Monitoring & Alerts' },
  { name: 'config',             url: '/config',                title: 'Platform Config' },
  { name: 'docs',               url: '/docs',                  title: 'Documentation' },
];

// Full super-admin walkthrough — every menu and sub-tab
const walkthrough = [
  // Operations
  { url: '/',                      wait: 2500, desc: 'Dashboard' },
  { url: '/apps',                  wait: 2000, desc: 'Applications' },
  { url: '/services',              wait: 2000, desc: 'Services' },
  { url: '/secrets',               wait: 2000, desc: 'Secrets' },

  // IAM — walk through all tabs
  { url: '/users',                 wait: 2000, desc: 'Users & IAM (Workspaces tab)' },
  { url: '/users?tab=orgs',       wait: 2000, desc: 'IAM — Organizations' },
  { url: '/users?tab=users',      wait: 2000, desc: 'IAM — Users' },
  { url: '/users?tab=policies',   wait: 2000, desc: 'IAM — Policies' },
  { url: '/users?tab=audit',      wait: 2000, desc: 'IAM — Audit Log' },

  // Workspaces
  { url: '/workspaces',            wait: 2000, desc: 'Workspaces' },

  // Settings
  { url: '/clusters',              wait: 2000, desc: 'Clusters' },
  { url: '/catalog',               wait: 2500, scroll: true, desc: 'Service Catalog' },
  { url: '/settings/registry',     wait: 1500, desc: 'Registry Settings' },
  { url: '/settings/webhooks',     wait: 1500, desc: 'Webhook Settings' },
  { url: '/settings/smtp',         wait: 1500, desc: 'SMTP Settings' },
  { url: '/settings/endpoints',    wait: 2000, desc: 'Endpoint Settings' },
  { url: '/monitoring',            wait: 2000, desc: 'Monitoring & Alerts' },
  { url: '/config',                wait: 2000, desc: 'Platform Config' },
  { url: '/docs',                  wait: 2000, desc: 'Documentation (User Manual)' },
  { url: '/docs?tab=admin',        wait: 2000, desc: 'Documentation (Admin Guide)' },
  { url: '/docs?tab=architecture', wait: 2000, desc: 'Documentation (Architecture)' },

  // Return to dashboard
  { url: '/',                      wait: 2000, desc: 'Back to Dashboard' },
];

async function signIn(page: import('@playwright/test').Page) {
  console.log('  🔐 Signing in as platform-admin...');

  // Navigate to admin — will redirect to login page
  await page.goto(BASE_URL, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1000);

  // Click "Sign in with Keycloak" button
  const signInBtn = page.locator('text=Sign in with Keycloak');
  if (await signInBtn.isVisible()) {
    await signInBtn.click();
    await page.waitForTimeout(2000);
  } else {
    // Try the Sign In button in header
    const headerSignIn = page.locator('text=Sign In');
    if (await headerSignIn.isVisible()) {
      await headerSignIn.click();
      await page.waitForTimeout(2000);
    }
  }

  // Fill Keycloak login form
  const usernameField = page.locator('#username');
  if (await usernameField.isVisible({ timeout: 5000 })) {
    await usernameField.fill(KC_USER);
    await page.locator('#password').fill(KC_PASS);
    await page.locator('#kc-login').click();
    await page.waitForTimeout(3000);
    console.log('  ✅ Signed in as platform-admin');
  } else {
    console.log('  ⚠️ Could not find Keycloak login form — may already be signed in');
  }
}

async function takeScreenshots(browser: Browser) {
  console.log('\n📸 Taking screenshots (super admin)...\n');
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 2,
    ignoreHTTPSErrors: true,
  });
  const page = await context.newPage();

  // Sign in first
  await signIn(page);

  for (const p of pages) {
    const filepath = path.join(OUTPUT_DIR, `${p.name}.png`);
    await page.goto(`${BASE_URL}${p.url}`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(500);
    await page.screenshot({ path: filepath, fullPage: false });
    console.log(`  ✅ ${p.title} → ${p.name}.png`);
  }

  await context.close();
  console.log(`\n📸 ${pages.length} screenshots saved to docs/images/\n`);
}

async function recordWalkthrough(browser: Browser) {
  console.log('\n🎬 Recording full walkthrough (super admin)...\n');
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 1,
    ignoreHTTPSErrors: true,
    recordVideo: {
      dir: OUTPUT_DIR,
      size: { width: 1440, height: 900 },
    },
  });
  const page = await context.newPage();

  // Sign in first
  await signIn(page);

  for (const step of walkthrough) {
    console.log(`  🎥 ${step.desc}...`);
    await page.goto(`${BASE_URL}${step.url}`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(300);

    if (step.scroll) {
      await page.evaluate(() => {
        return new Promise<void>((resolve) => {
          let totalHeight = 0;
          const distance = 200;
          const timer = setInterval(() => {
            window.scrollBy(0, distance);
            totalHeight += distance;
            if (totalHeight >= document.body.scrollHeight - window.innerHeight) {
              clearInterval(timer);
              setTimeout(() => {
                window.scrollTo(0, 0);
                resolve();
              }, 500);
            }
          }, 100);
        });
      });
    }

    await page.waitForTimeout(step.wait);
  }

  await page.close();
  await context.close();

  // Find the recorded video file and rename it
  const files = fs.readdirSync(OUTPUT_DIR).filter((f: string) => f.endsWith('.webm') && f !== 'dashboard-walkthrough.webm');
  if (files.length > 0) {
    const latestVideo = files.sort().pop()!;
    const src = path.join(OUTPUT_DIR, latestVideo);
    const dest = path.join(OUTPUT_DIR, 'dashboard-walkthrough.webm');
    // Remove old video first if it exists
    if (fs.existsSync(dest)) {
      fs.unlinkSync(dest);
    }
    fs.renameSync(src, dest);
    console.log(`\n🎬 Walkthrough video saved → docs/images/dashboard-walkthrough.webm\n`);
  }
}

async function main() {
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });

  const browser = await chromium.launch({ headless: true });

  try {
    await takeScreenshots(browser);
    await recordWalkthrough(browser);
  } finally {
    await browser.close();
  }

  console.log('✨ Done! All screenshots and video are in docs/images/');
}

main().catch(console.error);
