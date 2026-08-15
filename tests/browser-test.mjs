import { chromium } from 'playwright';
import fs from 'fs';
import path from 'path';

const BASE = 'http://localhost:3000';
const SCREENSHOT_DIR = '/tmp/itsm-browser-test';
fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });

const browser = await chromium.launch({ headless: false, args: ['--start-maximized'] });
const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
const page = await context.newPage();

let step = 0;
const results = [];

function log(msg) { console.log(`[${++step}] ${msg}`); }
async function snap(name) {
  const file = path.join(SCREENSHOT_DIR, `${String(step).padStart(2,'0')}-${name}.png`);
  await page.screenshot({ path: file, fullPage: true });
  log(`Screenshot: ${file}`);
}

// ========== 1. LOGIN ==========
log('Navigating to login page...');
await page.goto(`${BASE}/login`, { waitUntil: 'networkidle', timeout: 15000 });
await page.waitForTimeout(2000);
await snap('login-page');

// Check login form elements
const usernameInput = page.locator('input[placeholder*="用户名"], input[name="username"], input[type="text"]').first();
const passwordInput = page.locator('input[type="password"]').first();
const submitBtn = page.locator('button[type="submit"], button:has-text("登录")').first();

const hasUsername = await usernameInput.isVisible().catch(() => false);
const hasPassword = await passwordInput.isVisible().catch(() => false);
const hasSubmit = await submitBtn.isVisible().catch(() => false);
log(`Login form: username=${hasUsername}, password=${hasPassword}, submit=${hasSubmit}`);

// Fill and login
if (hasUsername && hasPassword && hasSubmit) {
  await usernameInput.fill('admin');
  await passwordInput.fill(process.env.ADMIN_PASSWORD || 'admin123');
  await snap('login-filled');
  await submitBtn.click();
  await page.waitForTimeout(5000);
  log(`After login URL: ${page.url()}`);
  await snap('after-login');
  
  if (!page.url().includes('/login')) {
    results.push({ page: 'LOGIN', status: '✅ PASS', detail: 'Login successful' });
  } else {
    results.push({ page: 'LOGIN', status: '❌ FAIL', detail: 'Still on login page' });
    // Check for error messages
    const errorText = await page.locator('.ant-message-error, .ant-notification, [class*="error"]').first().textContent().catch(() => '');
    log(`Login error: ${errorText}`);
  }
} else {
  results.push({ page: 'LOGIN', status: '❌ FAIL', detail: 'Login form not found' });
}

// ========== 2. DASHBOARD ==========
log('Testing Dashboard...');
await page.goto(`${BASE}/dashboard`, { waitUntil: 'networkidle', timeout: 15000 });
await page.waitForTimeout(3000);
await snap('dashboard');

const dashUrl = page.url();
if (dashUrl.includes('/login')) {
  results.push({ page: 'DASHBOARD', status: '❌ FAIL', detail: 'Redirected to login' });
} else {
  const sidebarItems = await page.locator('.ant-menu-item, [role="menuitem"], nav a[href]').count();
  const headerBtns = await page.locator('header button').count();
  const pageBtns = await page.locator('main button, [role="main"] button').count();
  log(`Dashboard: sidebar=${sidebarItems}, headerBtns=${headerBtns}, pageBtns=${pageBtns}`);
  results.push({ page: 'DASHBOARD', status: '✅ PASS', detail: `sidebar=${sidebarItems}, header=${headerBtns}, page=${pageBtns}` });
}

// ========== 3-16. Test each module ==========
const modules = [
  { name: 'TICKETS', path: '/tickets', createBtnText: '新建' },
  { name: 'INCIDENTS', path: '/incidents', createBtnText: '新建' },
  { name: 'PROBLEMS', path: '/problems', createBtnText: '新建' },
  { name: 'CHANGES', path: '/changes', createBtnText: '新建' },
  { name: 'CMDB', path: '/cmdb', createBtnText: '新建' },
  { name: 'SERVICE-CATALOG', path: '/service-catalog', createBtnText: '新建' },
  { name: 'KNOWLEDGE', path: '/knowledge', createBtnText: '新建' },
  { name: 'SLA', path: '/sla', createBtnText: '新建' },
  { name: 'WORKFLOW', path: '/workflow', createBtnText: '新建' },
  { name: 'SYSTEM-USERS', path: '/system/users', createBtnText: '新建' },
  { name: 'RELEASES', path: '/releases', createBtnText: '新建' },
  { name: 'STANDARD-CHANGES', path: '/standard-changes', createBtnText: '新建' },
  { name: 'SERVICE-REQUESTS', path: '/service-requests', createBtnText: '新建' },
  { name: 'AI-CHAT', path: '/ai/chat', createBtnText: '发送' },
];

for (const mod of modules) {
  log(`Testing ${mod.name} (${mod.path})...`);
  await page.goto(`${BASE}${mod.path}`, { waitUntil: 'networkidle', timeout: 15000 });
  await page.waitForTimeout(3000);
  await snap(mod.name.toLowerCase());

  const currentUrl = page.url();
  if (currentUrl.includes('/login')) {
    results.push({ page: mod.name, status: '❌ FAIL', detail: 'Redirected to login' });
    continue;
  }

  // Count visible buttons
  const allBtns = await page.locator('button:visible').all();
  const btnTexts = [];
  for (const btn of allBtns) {
    const text = (await btn.textContent())?.trim();
    if (text && text.length < 30) btnTexts.push(text);
  }
  log(`  ${mod.name} buttons: [${btnTexts.join(', ')}]`);

  // Check for table rows
  const rows = await page.locator('table tbody tr, .ant-table-row').count();
  log(`  ${mod.name} table rows: ${rows}`);

  // Check for create button
  const createBtn = page.locator(`button:has-text("${mod.createBtnText}")`).first();
  const hasCreate = await createBtn.isVisible({ timeout: 3000 }).catch(() => false);

  results.push({
    page: mod.name,
    status: hasCreate ? '✅ PASS' : '⚠️ WARN',
    detail: `buttons=${allBtns.length}, rows=${rows}, createBtn=${hasCreate}, btns=[${btnTexts.slice(0, 8).join(', ')}]`
  });

  // Test create button click if visible
  if (hasCreate) {
    await createBtn.click();
    await page.waitForTimeout(2000);
    await snap(`${mod.name.toLowerCase()}-create-click`);
    // Close modal/drawer if opened
    const closeBtn = page.locator('.ant-modal-close, .ant-drawer-close, button:has-text("取消"), button:has-text("关闭")').first();
    if (await closeBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await closeBtn.click();
      await page.waitForTimeout(500);
      log(`  ${mod.name} create modal/drawer opened and closed OK`);
    } else {
      log(`  ${mod.name} create button clicked (no modal detected, may navigate)`);
    }
  }
}

// ========== TICKET DETAIL ==========
log('Testing ticket detail page...');
await page.goto(`${BASE}/tickets`, { waitUntil: 'networkidle', timeout: 15000 });
await page.waitForTimeout(3000);
const firstRow = page.locator('table tbody tr, .ant-table-row').first();
if (await firstRow.isVisible().catch(() => false)) {
  await firstRow.click();
  await page.waitForTimeout(3000);
  await snap('ticket-detail');
  const detailUrl = page.url();
  if (!detailUrl.includes('/login')) {
    const detailBtns = await page.locator('button:visible').all();
    const detailBtnTexts = [];
    for (const btn of detailBtns) {
      const t = (await btn.textContent())?.trim();
      if (t && t.length < 30) detailBtnTexts.push(t);
    }
    results.push({ page: 'TICKET-DETAIL', status: '✅ PASS', detail: `buttons=[${detailBtnTexts.join(', ')}]` });
    log(`  Ticket detail buttons: [${detailBtnTexts.join(', ')}]`);
  } else {
    results.push({ page: 'TICKET-DETAIL', status: '❌ FAIL', detail: 'Redirected to login' });
  }
}

// ========== SUMMARY ==========
console.log('\n' + '='.repeat(80));
console.log('ITSM PRODUCTION BUTTON-LEVEL TEST REPORT');
console.log('='.repeat(80));
console.log(`Time: ${new Date().toISOString()}`);
console.log(`URL: ${BASE}`);
console.log('-'.repeat(80));
for (const r of results) {
  console.log(`${r.status}  ${r.page.padEnd(20)}  ${r.detail}`);
}
console.log('-'.repeat(80));
const passed = results.filter(r => r.status.includes('PASS')).length;
const warned = results.filter(r => r.status.includes('WARN')).length;
const failed = results.filter(r => r.status.includes('FAIL')).length;
console.log(`Total: ${results.length} | Pass: ${passed} | Warn: ${warned} | Fail: ${failed}`);
console.log('='.repeat(80));

await browser.close();
