/**
 * Four-Domain Browser Journey — Playwright
 *
 * Goal: walk through Ticket / Incident / Problem / Change domains in browser,
 * capture screenshots at each lifecycle step, verify each new entity exists
 * via API.
 *
 * Output:
 *   reports/four-domain-journey/<date>/{tickets,incidents,problems,changes}/*.png
 *   reports/four-domain-journey/<date>/journey-summary.json
 *
 * Reproducible: ADMIN_USER + ADMIN_PASSWORD env vars override default admin.
 */

const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');

const BASE_URL = process.env.BASE_URL || 'http://localhost:3000';
const API_URL = process.env.API_URL || 'http://localhost:8090';
const ADMIN_USER = process.env.ADMIN_USER || 'admin';
const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD || 'admin123';
const RUN_DATE = new Date().toISOString().slice(0, 10);
const SCREENSHOT_ROOT = process.env.SCREENSHOT_DIR ||
  path.join('/Users/heidsoft/Downloads/research/itsm/reports/four-domain-journey', RUN_DATE);

if (!fs.existsSync(SCREENSHOT_ROOT)) {
  fs.mkdirSync(SCREENSHOT_ROOT, { recursive: true });
}

const ts = () => new Date().toISOString().replace(/[:.]/g, '-');
const log = (msg) => console.log(`[${ts()}] ${msg}`);

// 令牌改为 HttpOnly cookie 下发后，前端路由守卫读的是 zustand `auth-storage`
// （localStorage），只注入 cookie 会被重定向回 /login——实测截图全是登录页。
// 因此旅程必须走真实 UI 登录：填表单提交，让 SPA 自己写入 store 与 cookie。
async function loginViaUI(page) {
  log(`Login as ${ADMIN_USER} through the SPA form`);
  await page.goto(`${BASE_URL}/login`, { waitUntil: 'domcontentloaded' });
  await page.waitForSelector('#username', { timeout: 30000 });
  await page.fill('#username', ADMIN_USER);
  await page.fill('#password', ADMIN_PASSWORD);
  await page.click('button[type="submit"]');
  await page
    .waitForFunction(() => !window.location.pathname.startsWith('/login'), null, { timeout: 30000 })
    .catch(() => {
      throw new Error(`login did not leave /login: ${page.url()}`);
    });
  const cookies = await page.context().cookies();
  if (!cookies.some((c) => c.name === 'access_token')) {
    throw new Error('UI login did not set access_token cookie');
  }

  // 写操作走 Double Submit Cookie：取一次 csrf token，让 cookie 与 header 同源。
  const csrfResp = await page.request.get(`${API_URL}/api/v1/csrf-token`);
  if (!csrfResp.ok()) throw new Error(`csrf-token failed: ${csrfResp.status()}`);
  const csrf = (await csrfResp.json()).data?.csrf_token;
  if (!csrf) throw new Error('csrf-token response missing csrf_token');
  log(`Authenticated, landed on ${page.url()}, csrf_len=${csrf.length}`);
}

async function authedFetch(page, url, options = {}) {
  const method = (options.method || 'GET').toUpperCase();
  const cookies = await page.context().cookies();
  const csrf = cookies.find((c) => c.name === 'csrf_token')?.value || '';
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  };
  if (csrf && ['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)) {
    headers['X-CSRF-Token'] = decodeURIComponent(csrf);
  }
  const init = { method, headers };
  if (options.body !== undefined) init.data = options.body;
  const resp = await page.request.fetch(url, init);
  const body = await resp.json().catch(() => ({}));
  return { ok: resp.ok(), status: resp.status(), body };
}

async function shoot(page, dir, name) {
  const fullDir = path.join(SCREENSHOT_ROOT, dir);
  if (!fs.existsSync(fullDir)) fs.mkdirSync(fullDir, { recursive: true });
  const file = path.join(fullDir, `${name}.png`);
  await page.screenshot({ path: file, fullPage: true });
  log(`📸 ${dir}/${name}.png`);
  return file;
}

async function visitList(page, dir, url) {
  log(`Visit ${url}`);
  await page.goto(`${BASE_URL}${url}`, { waitUntil: 'domcontentloaded' });
  await page.waitForLoadState('networkidle').catch(() => {});
  await page.waitForTimeout(2000);
  await shoot(page, dir, '01-list');
}

async function visitDetail(page, dir, url) {
  log(`Visit ${url}`);
  await page.goto(`${BASE_URL}${url}`, { waitUntil: 'domcontentloaded' });
  await page.waitForLoadState('networkidle').catch(() => {});
  await page.waitForTimeout(2000);
  await shoot(page, dir, '03-detail');
}

// --------- Domain journeys ---------

async function ticketJourney(page, summary) {
  log('=== Ticket 域旅程开始 ===');
  await visitList(page, 'tickets', '/tickets');

  const title = `prod-stack-rls-pilot-ticket-${Date.now()}`;
  const created = await authedFetch(page, `${API_URL}/api/v1/tickets`, {
    method: 'POST',
    body: JSON.stringify({
      title,
      description: 'Pilot rebuild ticket created by automated four-domain journey. >=10 chars.',
      type: 'incident',
      priority: 'medium',
      categoryId: 8,
    }),
  });
  if (!created.ok)
    throw new Error(
      `create ticket failed: http=${created.status} body=${JSON.stringify(created.body).slice(0, 300)}`
    );
  const ticketId = created.body.data?.id || created.body.data?.ticket?.id;
  summary.tickets = { id: ticketId, title, created: true };

  await page.goto(`${BASE_URL}/tickets/create`, { waitUntil: 'domcontentloaded' });
  await page.waitForLoadState('networkidle').catch(() => {});
  await page.waitForTimeout(2000);
  await shoot(page, 'tickets', '02-create-form');

  if (ticketId) {
    await visitDetail(page, 'tickets', `/tickets/${ticketId}`);
    const detail = await authedFetch(page, `${API_URL}/api/v1/tickets/${ticketId}`);
    summary.tickets.detail_visible = detail.ok;
    summary.tickets.detail_status = detail.body?.data?.status;
  }
  log('=== Ticket 域旅程完成 ===');
}

async function incidentJourney(page, summary) {
  log('=== Incident 域旅程开始 ===');
  await visitList(page, 'incidents', '/incidents');

  const title = `prod-stack-rls-pilot-incident-${Date.now()}`;
  const created = await authedFetch(page, `${API_URL}/api/v1/incidents`, {
    method: 'POST',
    body: JSON.stringify({
      title,
      description: 'Pilot rebuild incident created by automated four-domain journey.',
      type: 'incident',
      priority: 'high',
      severity: 'high',
      impact: 'medium',
      urgency: 'medium',
      category: 'outage',
      source: 'manual',
    }),
  });
  if (!created.ok) {
    log(`incident create failed (non-fatal): ${JSON.stringify(created).slice(0, 200)}`);
    summary.incidents = { created: false, error: created.body?.message };
  } else {
    const incidentId = created.body.data?.id || created.body.data?.incident?.id;
    summary.incidents = { id: incidentId, title, created: true };
    await page.goto(`${BASE_URL}/incidents/create`, { waitUntil: 'domcontentloaded' });
    await page.waitForLoadState('networkidle').catch(() => {});
    await page.waitForTimeout(2000);
    await shoot(page, 'incidents', '02-create-form');

    if (incidentId) {
      await visitDetail(page, 'incidents', `/incidents/${incidentId}`);
      const detail = await authedFetch(page, `${API_URL}/api/v1/incidents/${incidentId}`);
      summary.incidents.detail_visible = detail.ok;
      summary.incidents.detail_status = detail.body?.data?.status;
    }
  }
  log('=== Incident 域旅程完成 ===');
}

async function problemJourney(page, summary) {
  log('=== Problem 域旅程开始 ===');
  await visitList(page, 'problems', '/problems');

  const title = `prod-stack-rls-pilot-problem-${Date.now()}`;
  const created = await authedFetch(page, `${API_URL}/api/v1/problems`, {
    method: 'POST',
    body: JSON.stringify({
      title,
      description: 'Pilot rebuild problem created by automated four-domain journey with > 10 chars.',
      priority: 'medium',
      category: 'root_cause_analysis',
      impact: 'medium',
      impactScope: 'limited',
    }),
  });
  if (!created.ok) {
    log(`problem create failed (non-fatal): ${JSON.stringify(created).slice(0, 200)}`);
    summary.problems = { created: false, error: created.body?.message };
  } else {
    const problemId = created.body.data?.id || created.body.data?.problem?.id;
    summary.problems = { id: problemId, title, created: true };

    await page.goto(`${BASE_URL}/problems/new`, { waitUntil: 'domcontentloaded' });
    await page.waitForLoadState('networkidle').catch(() => {});
    await page.waitForTimeout(2000);
    await shoot(page, 'problems', '02-create-form');

    if (problemId) {
      await visitDetail(page, 'problems', `/problems/${problemId}`);
      const detail = await authedFetch(page, `${API_URL}/api/v1/problems/${problemId}`);
      summary.problems.detail_visible = detail.ok;
      summary.problems.detail_status = detail.body?.data?.status;
    }
  }
  log('=== Problem 域旅程完成 ===');
}

async function changeJourney(page, summary) {
  log('=== Change 域旅程开始 ===');
  await visitList(page, 'changes', '/changes');

  const title = `prod-stack-rls-pilot-change-${Date.now()}`;
  const created = await authedFetch(page, `${API_URL}/api/v1/changes`, {
    method: 'POST',
    body: JSON.stringify({
      title,
      description: 'Pilot rebuild change created by automated four-domain journey.',
      justification: 'Validating four-domain browser journey with real PG RLS pilot.',
      type: 'normal',
      priority: 'medium',
      impactScope: 'limited',
      riskLevel: 'medium',
    }),
  });
  if (!created.ok) {
    log(`change create failed (non-fatal): ${JSON.stringify(created).slice(0, 200)}`);
    summary.changes = { created: false, error: created.body?.message };
  } else {
    const changeId = created.body.data?.id || created.body.data?.change?.id;
    summary.changes = { id: changeId, title, created: true };

    await page.goto(`${BASE_URL}/changes/new`, { waitUntil: 'domcontentloaded' });
    await page.waitForLoadState('networkidle').catch(() => {});
    await page.waitForTimeout(2000);
    await shoot(page, 'changes', '02-create-form');

    if (changeId) {
      await visitDetail(page, 'changes', `/changes/${changeId}`);
      const detail = await authedFetch(page, `${API_URL}/api/v1/changes/${changeId}`);
      summary.changes.detail_visible = detail.ok;
      summary.changes.detail_status = detail.body?.data?.status;
    }
  }
  log('=== Change 域旅程完成 ===');
}

// --------- Main ---------

async function run() {
  log('=== 四域浏览器旅程开始 ===');
  const summary = { date: RUN_DATE, base_url: BASE_URL, api_url: API_URL };

  const browser = await chromium.launch({
    headless: true,
    args: ['--disable-crashpad', '--disable-breakpad'],
  });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();

  page.on('pageerror', (err) => log(`[browser:pageerror] ${err.message}`));

  try {
    await loginViaUI(page);
    summary.auth = 'SPA form login + httpOnly cookie + csrf_token';

    await ticketJourney(page, summary);
    await incidentJourney(page, summary);
    await problemJourney(page, summary);
    await changeJourney(page, summary);
  } catch (e) {
    summary.error = e.message;
    log(`[fatal] ${e.message}`);
  }

  fs.writeFileSync(
    path.join(SCREENSHOT_ROOT, 'journey-summary.json'),
    JSON.stringify(summary, null, 2)
  );
  log('journey-summary.json written');
  await browser.close();
  log('=== 四域浏览器旅程完成 ===');
}

run().catch((e) => {
  console.error(e);
  process.exit(1);
});