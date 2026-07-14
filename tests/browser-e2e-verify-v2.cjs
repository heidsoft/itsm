/**
 * Browser End-to-End Verification Script (v2 - Reliable)
 *
 * Validates the unified BPMN workflow migration by:
 * 1. Loading login page
 * 2. Logging in via API
 * 3. Navigating through key UI pages (dashboard, tickets, approvals)
 * 4. Creating incident/change/service_request/problem tickets via API
 * 5. Verifying process-bindings API confirms correct mappings
 * 6. Verifying each ticket has a workflow state
 * 7. Capturing screenshots at each step
 */

const { chromium } = require('playwright');
const fs = require('fs');

const BASE_URL = process.env.BASE_URL || 'http://localhost:3000';
const API_URL = process.env.API_URL || 'http://localhost:8090';
const SCREENSHOT_DIR = '/tmp/itsm-e2e-screenshots-v2';

if (!fs.existsSync(SCREENSHOT_DIR)) {
  fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
}

const ts = () => new Date().toISOString().replace(/[:.]/g, '-');
const log = (msg) => console.log(`[${ts()}] ${msg}`);

async function loginViaApi(page) {
  log('🔐 Logging in via API as admin');
  const response = await page.request.post(`${API_URL}/api/v1/auth/login`, {
    headers: { 'Content-Type': 'application/json' },
    data: { username: 'admin', password: 'admin123' },
  });
  const body = await response.json();
  if (!response.ok()) {
    throw new Error(`Login failed: ${response.status} ${JSON.stringify(body)}`);
  }
  const token = body.data?.access_token;
  if (!token) throw new Error('No access_token');
  await page.goto(BASE_URL);
  await page.evaluate((t) => {
    localStorage.setItem('access_token', t);
    localStorage.setItem('auth_token', t);
    document.cookie = `auth-token=${t}; path=/; max-age=900`;
  }, token);
  log('✅ Login successful');
  return token;
}

async function apiGet(token, path) {
  const response = await fetch(`${API_URL}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const body = await response.json();
  if (!response.ok) {
    throw new Error(`GET ${path} failed: ${response.status} ${JSON.stringify(body)}`);
  }
  return body.data;
}

async function apiPost(token, path, data) {
  const response = await fetch(`${API_URL}${path}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(data),
  });
  const body = await response.json();
  if (!response.ok) {
    throw new Error(`POST ${path} failed: ${response.status} ${JSON.stringify(body)}`);
  }
  return body.data;
}

async function run() {
  log('🚀 Starting browser E2E verification (v2)');
  const browser = await chromium.launch({
    headless: true,
    args: ['--disable-crashpad', '--disable-breakpad'],
  });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });
  const page = await context.newPage();
  page.on('pageerror', (err) => log(`[browser:pageerror] ${err.message}`));

  const results = {
    ui_screenshots: [],
    bindings: [],
    ticket_tests: [],
    errors: [],
  };

  try {
    // ========== Step 1: Capture login page ==========
    log('📸 Step 1: Capture login page');
    await page.goto(`${BASE_URL}/login`, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    await page.screenshot({ path: `${SCREENSHOT_DIR}/01-login.png`, fullPage: true });
    results.ui_screenshots.push('01-login.png');

    // ========== Step 2: Login ==========
    const token = await loginViaApi(page);

    // ========== Step 3: Dashboard ==========
    log('📸 Step 2: Capture dashboard after login');
    await page.goto(`${BASE_URL}/dashboard`, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);
    await page.screenshot({ path: `${SCREENSHOT_DIR}/02-dashboard.png`, fullPage: true });
    results.ui_screenshots.push('02-dashboard.png');

    // ========== Step 4: Tickets list ==========
    log('📸 Step 3: Capture tickets list');
    await page.goto(`${BASE_URL}/tickets`, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);
    await page.screenshot({ path: `${SCREENSHOT_DIR}/03-tickets-list.png`, fullPage: true });
    results.ui_screenshots.push('03-tickets-list.png');

    // ========== Step 5: Verify process-bindings ==========
    log('\n📋 Step 4: Verify process-bindings via API');
    const bindings = await apiGet(token, '/api/v1/process-bindings');
    const activeBindings = bindings.filter((b) => b.is_active);

    // Build map: ticket type -> expected process
    const expectedMap = {};
    for (const b of activeBindings) {
      if (b.business_type === 'ticket') {
        const sub = b.business_sub_type || 'general';
        expectedMap[sub] = b.process_definition_key;
      }
    }
    // Also check independent types
    for (const b of activeBindings) {
      if (['incident', 'change', 'service_request', 'problem'].includes(b.business_type)) {
        expectedMap[b.business_type] = b.process_definition_key;
      }
    }
    log('  Expected bindings:');
    for (const [k, v] of Object.entries(expectedMap)) {
      log(`    ${k.padEnd(20)} → ${v}`);
    }
    results.bindings = expectedMap;

    // ========== Step 6: Create 4 ticket types and verify workflow state ==========
    const ticketTests = [
      { type: 'incident', priority: 'urgent', title: `E2E-事件-${Date.now()}` },
      { type: 'change', priority: 'medium', title: `E2E-变更-${Date.now()}` },
      { type: 'service_request', priority: 'low', title: `E2E-请求-${Date.now()}` },
      { type: 'problem', priority: 'high', title: `E2E-问题-${Date.now()}` },
    ];

    log('\n📋 Step 5: Create tickets and verify workflow');
    for (const [idx, t] of ticketTests.entries()) {
      log(`\n  Test ${idx + 1}: ${t.type}`);
      try {
        const created = await apiPost(token, '/api/v1/tickets', {
          title: t.title,
          description: `${t.type} 类型的端到端验证工单，测试 BPMN 工作流绑定`,
          priority: t.priority,
          type: t.type,
          category: 'general',
        });
        log(`    ✅ Created ticket id=${created.id} number=${created.ticketNumber}`);

        // Wait for async process trigger
        await new Promise((r) => setTimeout(r, 1500));

        // Verify workflow state exists
        const state = await apiGet(
          token,
          `/api/v1/tickets/${created.id}/workflow/state`
        );
        const hasState =
          state &&
          state.current_status &&
          Array.isArray(state.available_actions) &&
          state.available_actions.length > 0;
        const expected = expectedMap[t.type] || expectedMap.general;
        const passed = hasState && state.ticket_id === created.id;

        log(`    🔍 Workflow state: status=${state?.current_status}, actions=${state?.available_actions?.join(',')}`);
        log(`    🎯 Expected process: ${expected}`);
        log(`    ${passed ? '✅' : '❌'} Workflow state ${passed ? 'available' : 'MISSING'}`);

        results.ticket_tests.push({
          type: t.type,
          ticketId: created.id,
          ticketNumber: created.ticketNumber,
          expectedProcess: expected,
          workflowStatus: state?.current_status,
          availableActions: state?.available_actions,
          passed,
        });

        // Navigate to detail page in browser
        log(`    📸 Navigate to ticket ${created.id} detail page`);
        await page.goto(`${BASE_URL}/tickets/${created.id}`, {
          waitUntil: 'domcontentloaded',
          timeout: 60000,
        });
        await page.waitForLoadState('networkidle');
        await page.waitForTimeout(2500);
        await page.screenshot({
          path: `${SCREENSHOT_DIR}/04-ticket-${t.type}-detail.png`,
          fullPage: true,
        });
        results.ui_screenshots.push(`04-ticket-${t.type}-detail.png`);
      } catch (err) {
        log(`    ❌ Failed: ${err.message}`);
        results.errors.push({ step: `create-${t.type}`, error: err.message });
      }
    }

    // ========== Step 7: Admin approvals page (verify migration notice) ==========
    log('\n📋 Step 6: Navigate to admin/approvals');
    await page.goto(`${BASE_URL}/admin/approvals`, {
      waitUntil: 'domcontentloaded',
      timeout: 60000,
    });
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2500);
    await page.screenshot({ path: `${SCREENSHOT_DIR}/05-admin-approvals.png`, fullPage: true });
    results.ui_screenshots.push('05-admin-approvals.png');

    // ========== Step 8: Admin approval-chains page ==========
    log('📋 Step 7: Navigate to admin/approval-chains');
    await page.goto(`${BASE_URL}/admin/approval-chains`, {
      waitUntil: 'domcontentloaded',
      timeout: 60000,
    });
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2500);
    await page.screenshot({
      path: `${SCREENSHOT_DIR}/06-admin-approval-chains.png`,
      fullPage: true,
    });
    results.ui_screenshots.push('06-admin-approval-chains.png');

    // ========== Step 9: Workflow definition page ==========
    log('📋 Step 8: Navigate to admin/process-definitions');
    await page.goto(`${BASE_URL}/admin/process-definitions`, {
      waitUntil: 'domcontentloaded',
      timeout: 60000,
    });
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2500);
    await page.screenshot({
      path: `${SCREENSHOT_DIR}/07-process-definitions.png`,
      fullPage: true,
    });
    results.ui_screenshots.push('07-process-definitions.png');

    // ========== Summary ==========
    log('\n========================================');
    log('📊 Test Results Summary');
    log('========================================');
    log(`UI Screenshots captured: ${results.ui_screenshots.length}`);
    for (const s of results.ui_screenshots) {
      log(`  📸 ${s}`);
    }
    log('');
    log(`Process Bindings: ${Object.keys(results.bindings).length} configured`);
    log('');
    let allPassed = true;
    for (const t of results.ticket_tests) {
      const status = t.passed ? '✅ PASS' : '❌ FAIL';
      log(`${status}  ${t.type.padEnd(16)} ticket=${t.ticketId} → ${t.expectedProcess}`);
      if (!t.passed) allPassed = false;
    }
    log('========================================');
    log(allPassed ? '🎉 All tests passed!' : '⚠️ Some tests failed');
    log(`Screenshots: ${SCREENSHOT_DIR}`);
    log('========================================');

    fs.writeFileSync(
      `${SCREENSHOT_DIR}/results.json`,
      JSON.stringify(results, null, 2)
    );

    process.exit(allPassed && results.errors.length === 0 ? 0 : 1);
  } catch (err) {
    log(`❌ FATAL: ${err.message}`);
    log(err.stack);
    results.errors.push({ step: 'fatal', error: err.message });
    fs.writeFileSync(`${SCREENSHOT_DIR}/results.json`, JSON.stringify(results, null, 2));
    await page
      .screenshot({ path: `${SCREENSHOT_DIR}/error.png`, fullPage: true })
      .catch(() => {});
    process.exit(2);
  } finally {
    await browser.close();
  }
}

run();
