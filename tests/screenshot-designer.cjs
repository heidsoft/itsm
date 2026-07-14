const { chromium } = require('playwright');

const BASE_URL = 'http://localhost:3000';
const API_URL = 'http://localhost:8090';
const SCREENSHOT_DIR = '/tmp/itsm-workflow-designer';

(async () => {
  const fs = require('fs');
  if (!fs.existsSync(SCREENSHOT_DIR)) {
    fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
  }

  const browser = await chromium.launch({ headless: true });
  const ctx = await browser.newContext({ viewport: { width: 1600, height: 1000 } });
  const page = await ctx.newPage();

  const errors = [];
  page.on('pageerror', (err) => {
    errors.push(`pageerror: ${err.message}`);
  });
  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      errors.push(`console.error: ${msg.text().slice(0, 200)}`);
    }
  });

  // Login via API
  const res = await page.request.post(`${API_URL}/api/v1/auth/login`, {
    headers: { 'Content-Type': 'application/json' },
    data: { username: 'admin', password: 'admin123' },
  });
  const body = await res.json();
  const token = body.data.access_token;

  await page.goto(BASE_URL);
  await page.evaluate((t) => {
    localStorage.setItem('access_token', t);
    localStorage.setItem('auth_token', t);
    document.cookie = `auth-token=${t}; path=/; max-age=900`;
  }, token);

  // Navigate to workflow designer
  console.log('Navigating to /workflow/designer...');
  await page.goto(`${BASE_URL}/workflow/designer`, { timeout: 90000 });
  await page.waitForLoadState('networkidle', { timeout: 60000 });
  await page.waitForTimeout(5000);
  await page.screenshot({ path: `${SCREENSHOT_DIR}/01-designer-empty.png`, fullPage: true });
  console.log('Screenshot 1 saved');

  // Navigate to /admin/groups
  console.log('Navigating to /admin/groups...');
  await page.goto(`${BASE_URL}/admin/groups`, { timeout: 60000 });
  await page.waitForLoadState('networkidle', { timeout: 30000 });
  await page.waitForTimeout(3000);
  await page.screenshot({ path: `${SCREENSHOT_DIR}/02-groups-page.png`, fullPage: true });
  console.log('Screenshot 2 saved');

  if (errors.length > 0) {
    console.log('\n=== Errors ===');
    errors.slice(0, 10).forEach((e) => console.log(e));
  } else {
    console.log('\n✅ No errors detected');
  }

  await browser.close();
})();
