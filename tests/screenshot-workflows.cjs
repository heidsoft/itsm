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

  // Login via API to get token
  const res = await page.request.post(`${API_URL}/api/v1/auth/login`, {
    headers: { 'Content-Type': 'application/json' },
    data: { username: 'admin', password: 'admin123' },
  });
  const body = await res.json();
  const token = body.data.access_token;

  // Persist token in localStorage
  await page.goto(BASE_URL);
  await page.evaluate((t) => {
    localStorage.setItem('access_token', t);
    localStorage.setItem('auth_token', t);
    document.cookie = `auth-token=${t}; path=/; max-age=900`;
  }, token);

  // Navigate to /admin/workflows
  await page.goto(`${BASE_URL}/admin/workflows`, {
    waitUntil: 'domcontentloaded',
    timeout: 60000,
  });
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(3000);
  await page.screenshot({ path: SCREENSHOT_PATH, fullPage: true });
  console.log(`Screenshot saved: ${SCREENSHOT_PATH}`);

  await browser.close();
})();
