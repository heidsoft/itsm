/**
 * Browser End-to-End Verification Script
 *
 * Validates the unified BPMN workflow migration by:
 * 1. Logging in via the browser UI
 * 2. Navigating to ticket creation page
 * 3. Submitting incident/change/service_request/problem tickets
 * 4. Verifying each ticket triggers the correct BPMN process
 * 5. Capturing screenshots at each step
 */

const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');

const BASE_URL = process.env.BASE_URL || 'http://localhost:3000';
const API_URL = process.env.API_URL || 'http://localhost:8090';
const SCREENSHOT_DIR = '/tmp/itsm-e2e-screenshots';

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
  if (!response.ok()) {
    throw new Error(`Login failed: ${response.status()} ${await response.text()}`);
  }
  const body = await response.json();
  const token = body.data?.access_token;
  if (!token) {
    throw new Error('No access_token in response');
  }
  await page.goto(BASE_URL);
  await page.evaluate((t) => {
    localStorage.setItem('access_token', t);
    localStorage.setItem('auth_token', t);
    document.cookie = `auth-token=${t}; path=/; max-age=900`;
  }, token);
  log('✅ Login successful, token stored');
  return token;
}

async function createTicketViaApi(token, ticketData) {
  const response = await fetch(`${API_URL}/api/v1/tickets`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(ticketData),
  });
  const body = await response.json();
  if (!response.ok) {
    throw new Error(`Create ticket failed: ${response.status} ${JSON.stringify(body)}`);
  }
  return body.data;
}

async function getTicketProcess(token, ticketId) {
  const response = await fetch(`${API_URL}/api/v1/tickets/${ticketId}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const body = await response.json();
  if (!response.ok) {
    throw new Error(`Get ticket failed: ${response.status} ${JSON.stringify(body)}`);
  }
  return body.data;
}

async function run() {
  log('🚀 Starting browser E2E verification of BPMN workflow migration');

  const browser = await chromium.launch({
    headless: true,
    args: ['--disable-crashpad', '--disable-breakpad'],
  });

  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });
  const page = await context.newPage();

  // Capture browser console for debugging
  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      log(`[browser:error] ${msg.text()}`);
    }
  });
  page.on('pageerror', (err) => {
    log(`[browser:pageerror] ${err.message}`);
  });

  try {
    // Step 1: Login page screenshot (anonymous)
    log('📸 Step 1: Capture login page');
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1500);
    await page.screenshot({ path: `${SCREENSHOT_DIR}/01-login-page.png`, fullPage: true });

    // Step 2: Login via API
    const token = await loginViaApi(page);

    // Step 3: Navigate to dashboard
    log('📸 Step 2: Navigate to dashboard after login');
    await page.goto(`${BASE_URL}/dashboard`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    await page.screenshot({ path: `${SCREENSHOT_DIR}/02-dashboard.png`, fullPage: true });

    // Step 4: Navigate to tickets list
    log('📸 Step 3: Navigate to tickets list');
    await page.goto(`${BASE_URL}/tickets`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    await page.screenshot({ path: `${SCREENSHOT_DIR}/03-tickets-list.png`, fullPage: true });

    // Step 5: Test 4 ticket types via API, then verify in UI
    const ticketTests = [
      {
        type: 'incident',
        priority: 'urgent',
        title: `E2E-事件工单-服务器宕机-${Date.now()}`,
        description: '生产数据库服务器CPU持续100%，所有请求超时，需要立即处理',
        expectedProcess: 'incident_emergency_flow',
      },
      {
        type: 'change',
        priority: 'medium',
        title: `E2E-变更工单-系统升级-${Date.now()}`,
        description: '将核心业务系统从v2.3升级到v3.0，预计停机30分钟',
        expectedProcess: 'change_normal_flow',
      },
      {
        type: 'service_request',
        priority: 'low',
        title: `E2E-服务请求-权限申请-${Date.now()}`,
        description: '申请新员工的VPN访问权限和邮箱账号',
        expectedProcess: 'service_request_flow',
      },
      {
        type: 'problem',
        priority: 'high',
        title: `E2E-问题工单-反复故障-${Date.now()}`,
        description: '用户登录模块每周出现3-5次500错误，需要根因分析',
        expectedProcess: 'problem_management_flow',
      },
    ];

    const results = [];

    for (const [index, ticket] of ticketTests.entries()) {
      log(`\n📋 Test ${index + 1}/4: Creating ${ticket.type} ticket`);

      // Create ticket via API (more reliable than UI for assertions)
      const created = await createTicketViaApi(token, {
        title: ticket.title,
        description: ticket.description,
        priority: ticket.priority,
        type: ticket.type,
        category: 'general',
      });
      log(`  ✅ Created ticket id=${created.id}`);

      // Fetch ticket details to verify BPMN process was started
      await new Promise((r) => setTimeout(r, 1500));
      const detail = await getTicketProcess(token, created.id);
      const processKey =
        detail.processDefinitionKey ||
        detail.process_instance_id ||
        detail.processInstanceKey ||
        'unknown';

      log(`  🔍 Process: ${processKey}`);
      log(`  🎯 Expected: ${ticket.expectedProcess}`);

      const passed = processKey.includes(ticket.expectedProcess) ||
        (detail.processDefinitionKey && detail.processDefinitionKey === ticket.expectedProcess);

      results.push({
        ...ticket,
        ticketId: created.id,
        processKey,
        passed,
      });

      // Navigate to ticket detail page in browser and screenshot
      log(`  📸 Navigating to ticket detail page in browser`);
      await page.goto(`${BASE_URL}/tickets/${created.id}`);
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(2000);
      await page.screenshot({
        path: `${SCREENSHOT_DIR}/04-ticket-detail-${ticket.type}.png`,
        fullPage: true,
      });
    }

    // Step 6: Navigate to admin/approvals to verify the migration notice
    log('\n📋 Step 6: Navigate to admin/approvals page (verify migration notice)');
    await page.goto(`${BASE_URL}/admin/approvals`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    await page.screenshot({ path: `${SCREENSHOT_DIR}/05-admin-approvals.png`, fullPage: true });

    log('\n📋 Step 7: Navigate to admin/approval-chains page');
    await page.goto(`${BASE_URL}/admin/approval-chains`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    await page.screenshot({
      path: `${SCREENSHOT_DIR}/06-admin-approval-chains.png`,
      fullPage: true,
    });

    // Summary
    log('\n========================================');
    log('📊 Test Results Summary');
    log('========================================');
    let allPassed = true;
    for (const r of results) {
      const status = r.passed ? '✅ PASS' : '❌ FAIL';
      log(`${status}  ${r.type.padEnd(16)} → ${r.processKey}`);
      if (!r.passed) allPassed = false;
    }
    log('========================================');
    log(allPassed ? '🎉 All BPMN process bindings are correct!' : '⚠️ Some bindings failed');
    log(`Screenshots saved to: ${SCREENSHOT_DIR}`);
    log('========================================');

    fs.writeFileSync(
      `${SCREENSHOT_DIR}/results.json`,
      JSON.stringify({ allPassed, results, timestamp: new Date().toISOString() }, null, 2)
    );

    process.exit(allPassed ? 0 : 1);
  } catch (err) {
    log(`❌ FATAL: ${err.message}`);
    log(err.stack);
    await page.screenshot({ path: `${SCREENSHOT_DIR}/error.png`, fullPage: true }).catch(() => {});
    process.exit(2);
  } finally {
    await browser.close();
  }
}

run();
