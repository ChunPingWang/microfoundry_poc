import { chromium, Browser, Page } from '@playwright/test';
import * as path from 'path';
import * as fs from 'fs';

const BASE_URL = 'http://localhost:8080';
const OUTPUT_DIR = path.resolve(__dirname, '../docs/images');

// Pages to screenshot
const pages = [
  { name: 'dashboard',       url: '/',                   title: 'Dashboard' },
  { name: 'apps-list',       url: '/apps',               title: 'Applications' },
  { name: 'services',        url: '/services',           title: 'Services' },
  { name: 'catalog',         url: '/catalog',            title: 'Service Catalog' },
  { name: 'clusters',        url: '/clusters',           title: 'Clusters' },
  { name: 'monitoring',      url: '/monitoring',         title: 'Monitoring' },
  { name: 'secrets',         url: '/secrets',            title: 'Secrets' },
  { name: 'users-iam',       url: '/users',              title: 'Users & IAM' },
  { name: 'settings',        url: '/settings/registry',  title: 'Settings' },
  { name: 'config',          url: '/config',             title: 'Platform Config' },
];

// Pages to navigate for the walkthrough video
const walkthrough = [
  { url: '/',                  wait: 2000, desc: 'Dashboard' },
  { url: '/apps',              wait: 2000, desc: 'Applications List' },
  { url: '/services',          wait: 2000, desc: 'Services' },
  { url: '/catalog',           wait: 2500, scroll: true, desc: 'Service Catalog' },
  { url: '/clusters',          wait: 2000, desc: 'Clusters' },
  { url: '/monitoring',        wait: 2000, desc: 'Monitoring' },
  { url: '/secrets',           wait: 2000, desc: 'Secrets' },
  { url: '/users',             wait: 2000, desc: 'Users & IAM' },
  { url: '/settings/registry', wait: 1500, desc: 'Registry Settings' },
  { url: '/settings/webhooks', wait: 1500, desc: 'Webhook Settings' },
  { url: '/settings/smtp',     wait: 1500, desc: 'SMTP Settings' },
  { url: '/config',            wait: 2000, desc: 'Platform Config' },
  { url: '/',                  wait: 2000, desc: 'Back to Dashboard' },
];

async function takeScreenshots(browser: Browser) {
  console.log('\n📸 Taking screenshots...\n');
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 2,
  });
  const page = await context.newPage();

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
  console.log('\n🎬 Recording walkthrough video...\n');
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 1,
    recordVideo: {
      dir: OUTPUT_DIR,
      size: { width: 1440, height: 900 },
    },
  });
  const page = await context.newPage();

  for (const step of walkthrough) {
    console.log(`  🎥 ${step.desc}...`);
    await page.goto(`${BASE_URL}${step.url}`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(300);

    if (step.scroll) {
      // Smooth scroll to show full page
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

  // Find the recorded video file
  const files = fs.readdirSync(OUTPUT_DIR).filter(f => f.endsWith('.webm'));
  if (files.length > 0) {
    const latestVideo = files.sort().pop()!;
    const src = path.join(OUTPUT_DIR, latestVideo);
    const dest = path.join(OUTPUT_DIR, 'dashboard-walkthrough.webm');
    fs.renameSync(src, dest);
    console.log(`\n🎬 Walkthrough video saved → docs/images/dashboard-walkthrough.webm\n`);
  }
}

async function main() {
  // Ensure output dir exists
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
