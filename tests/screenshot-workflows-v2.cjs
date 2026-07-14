const { chromium } = require('playwright');

const BASE_URL = 'http://localhost:3000';
const API_URL = 'http://localhost:8090';
const SCREENSHOT_PATH = '/tmp/itsm-e2e-screenshots-v2/08-admin-workflows.png';

(async () => {
  const browser = await chromium.launch({
    headless: true,
    args: ['--disable-crashpad'],
  });
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();

  const errors = [];
  page.on('pageerror', (err) => {
    errors.push(`pageerror: ${err.message}`);
    console.log(`[pageerror] ${err.message}`);
  });
  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      errors.push(`console.error: ${msg.text()}`);
      console.log(`[console.error] ${msg.text()}`);
    }
  });

  // Login via API
  const res = await page.request.post(`${API_URL}/api/v1/auth/login`, {
    headers: { 'Content-Type': 'application/json' },
    data: { username: 'admin', password: 'admin123' },
  });
  const body = await res.json();
  const token = body.data.access_token;
  console.log('Got token:', token?.slice(0, 20) + '...');

  await page.goto(BASE_URL);
  await page.evaluate((t) => {
    localStorage.setItem('access_token', t);
    localStorage.setItem('auth_token', t);
    document.cookie = `auth-token=${t}; path=/; max-age=900`;
  }, token);

  // Navigate with longer timeout
  console.log('Navigating to /admin/workflows...');
  try {
    await page.goto(`${BASE_URL}/admin/workflows`, { timeout: 90000 });
    await page.waitForLoadState('networkidle', { timeout: 60000 });
    await page.waitForTimeout(5000);
    await page.screenshot({ path: SCREENSHOT_PATH, fullPage: true });
    console.log(`Screenshot saved: ${SCREENSHOT_PATH}`);
  } catch (err) {
    console.log(`Navigation error: ${err.message}`);
    // Try alternative path /workflow
    console.log('Trying /workflow...');
    await page.goto(`${BASE_URL}/workflow`, { timeout: 60000 });
    await page.waitForLoadState('networkidle', { timeout: 30000 });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-e2e-screenshots-v2/08b-workflow.png', fullPage: true });
    console.log('Alternative screenshot saved');
  }

  if (errors.length > 0) {
    console.log('\n=== Errors found ===');
    errors.forEach((e) => console.log(e));
  }

  await browser.close();
})();
