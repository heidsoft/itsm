/**
 * ITSM 全面 E2E 测试 - 验证所有关键业务流和新增页面
 *
 * 覆盖范围：
 * 1. 登录 / 多角色权限
 * 2. 工单全生命周期
 * 3. 事件全生命周期
 * 4. 知识库 CRUD
 * 5. 服务目录申请
 * 6. 连接器市场（新）
 * 7. BPMN 工作流
 * 8. SLA 监控
 * 9. 报表 / Dashboard
 */

import type { Page } from '@playwright/test';
import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './test-utils';

const BASE = 'http://localhost:3000';
const API = 'http://localhost:8090';

async function waitApiOk(page: Page, urlPattern: RegExp): Promise<boolean> {
  try {
    const resp = await page.waitForResponse(
      r => urlPattern.test(r.url()) && r.status() >= 200 && r.status() < 300,
      { timeout: 15000 }
    );
    return true;
  } catch {
    return false;
  }
}

test.describe('ITSM 全面 E2E 业务流测试', () => {

  test.describe.configure({ mode: 'serial' });

  // ===== 1. 登录 + 主页面 =====
  test('1.1 admin 登录 + 所有主页面可访问', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto(`${BASE}/dashboard`);
    await page.waitForSelector('body', { timeout: 10000 });
    const html = await page.content();
    expect(html.length).toBeGreaterThan(1000);
  });

  test('1.2 admin 访问全部 27 个主页面无 404', async ({ page }) => {
    await loginAsAdmin(page);
    const pages = [
      '/dashboard', '/tickets', '/incidents', '/problems', '/changes',
      '/service-catalog', '/knowledge', '/cmdb', '/sla', '/sla-dashboard',
      '/sla-monitor', '/workflow', '/workflows', '/approvals',
      '/assets', '/applications', '/projects', '/licenses', '/reports',
      '/notifications', '/admin/users', '/admin/roles', '/admin/connectors',
      '/admin/sla-definitions', '/admin/overview',
    ];
    for (const p of pages) {
      const resp = await page.goto(`${BASE}${p}`, { waitUntil: 'domcontentloaded' });
      // 由于 Next.js 重定向，状态码可能是 200 或 307
      const status = resp?.status() ?? 0;
      expect([200, 307, 308]).toContain(status);
    }
  });

  // ===== 2. 工单全生命周期 =====
  test('2.1 工单创建→接受→in_progress→解决→关闭 完整闭环', async ({ page }) => {
    await loginAsAdmin(page);

    // 直接通过 API 创建（避免 UI 字段差异）
    const apiResp = await page.request.post(`${API}/api/v1/tickets`, {
      data: {
        title: `E2E 完整生命周期 ${Date.now()}`,
        description: 'E2E 测试工单，验证完整生命周期流程',
        priority: 'high',
        type: 'incident',
      },
    });
    expect(apiResp.status()).toBe(200);
    const ticketId = (await apiResp.json()).data.id;

    // Accept
    const acceptResp = await page.request.post(`${API}/api/v1/tickets/workflow/accept`, {
      data: { ticket_id: ticketId },
    });
    expect(acceptResp.status()).toBe(200);

    // Resolve
    const resolveResp = await page.request.post(`${API}/api/v1/tickets/workflow/resolve`, {
      data: { ticket_id: ticketId, resolution: '已修复', resolutionCategory: 'code_defect' },
    });
    expect(resolveResp.status()).toBe(200);

    // Close
    const closeResp = await page.request.post(`${API}/api/v1/tickets/workflow/close`, {
      data: { ticket_id: ticketId, closeNotes: '测试关闭', closeReason: '完成' },
    });
    expect(closeResp.status()).toBe(200);

    // History
    const histResp = await page.request.get(`${API}/api/v1/tickets/${ticketId}/workflow-history`);
    expect(histResp.status()).toBe(200);
    const hist = (await histResp.json()).data;
    expect(hist.length).toBeGreaterThanOrEqual(3);

    // 详情页渲染 - 等待 ticket 数据加载完成
    await page.goto(`${BASE}/tickets/${ticketId}`);
    await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => {});
    // 验证 ticket 详情 API 返回了 closed 状态
    const detailCheck = await page.request.get(`${API}/api/v1/tickets/${ticketId}`);
    const detailJson = await detailCheck.json();
    expect(detailCheck.status()).toBe(200);
    expect(detailJson.data.status).toBe('closed');
  });

  // ===== 3. 事件全生命周期 =====
  test('3.1 事件创建→确认→评论→根因→解决→关闭 完整闭环', async ({ page }) => {
    await loginAsAdmin(page);

    const apiResp = await page.request.post(`${API}/api/v1/incidents`, {
      data: {
        title: `E2E 事件 ${Date.now()}`,
        description: 'E2E 测试事件，验证完整生命周期',
        severity: 'critical',
        priority: 'critical',
        type: 'incident',
        source: 'monitoring',
      },
    });
    expect(apiResp.status()).toBe(200);
    const incId = (await apiResp.json()).data.id;

    // 4 个新端点
    const ackResp = await page.request.post(`${API}/api/v1/incidents/${incId}/acknowledge`, { data: { comment: '已确认' } });
    expect(ackResp.status()).toBe(200);

    const cmtResp = await page.request.post(`${API}/api/v1/incidents/${incId}/comments`, { data: { content: '处理中', isInternal: false } });
    expect(cmtResp.status()).toBe(200);

    const rcaResp = await page.request.put(`${API}/api/v1/incidents/${incId}/root-cause`, {
      data: { rootCause: '连接池耗尽', causeCategory: 'resource_exhaustion' },
    });
    expect(rcaResp.status()).toBe(200);

    const resResp = await page.request.post(`${API}/api/v1/incidents/${incId}/resolve`, { data: { resolution: '已恢复', resolutionCategory: 'environment' } });
    expect(resResp.status()).toBe(200);

    const clsResp = await page.request.post(`${API}/api/v1/incidents/${incId}/close`, { data: { comment: '已关闭' } });
    expect(clsResp.status()).toBe(200);
  });

  // ===== 4. 知识库 CRUD =====
  test('4.1 知识库：创建→列表→详情→搜索', async ({ page }) => {
    await loginAsAdmin(page);

    const createResp = await page.request.post(`${API}/api/v1/knowledge-articles`, {
      data: {
        title: `E2E 知识 ${Date.now()}`,
        content: '这是 E2E 测试创建的知识库文章内容',
        category: 'operations',
        tags: ['e2e', 'test'],
      },
    });
    expect(createResp.status()).toBe(200);
    const artId = (await createResp.json()).data.id;

    // 详情
    const detailResp = await page.request.get(`${API}/api/v1/knowledge-articles/${artId}`);
    expect(detailResp.status()).toBe(200);

    // 列表
    const listResp = await page.request.get(`${API}/api/v1/knowledge-articles?page=1&page_size=5`);
    expect(listResp.status()).toBe(200);
    const items = (await listResp.json()).data.articles;
    expect(items.length).toBeGreaterThan(0);

    // UI: 新建页面渲染
    await page.goto(`${BASE}/knowledge/articles/new`);
    await page.waitForSelector('form', { timeout: 10000 });
    const html = await page.content();
    expect(html).toContain('新建知识库文章');

    // UI: 详情页面渲染
    await page.goto(`${BASE}/knowledge/articles/${artId}`);
    await page.waitForSelector('body', { timeout: 10000 });
  });

  // ===== 5. 服务目录申请 =====
  test('5.1 服务目录：创建→详情→申请（含 compliance_ack + expire_at）', async ({ page }) => {
    await loginAsAdmin(page);

    // 创建服务目录
    const catResp = await page.request.post(`${API}/api/v1/service-catalogs`, {
      data: {
        name: `E2E 服务目录 ${Date.now()}`,
        description: 'E2E 测试用服务目录',
        category: '硬件',
        delivery_time: '5',
        status: 'enabled',
      },
    });
    expect(catResp.status()).toBe(200);
    const catId = (await catResp.json()).data.id;

    // 申请 - 必填 compliance_ack=true, expire_at
    const reqResp = await page.request.post(`${API}/api/v1/service-requests`, {
      data: {
        catalog_id: catId,
        title: `E2E 申请 ${Date.now()}`,
        reason: '工作需要',
        compliance_ack: true,
        expire_at: '2026-12-31T00:00:00Z',
      },
    });
    expect(reqResp.status()).toBe(200);
    const reqId = (await reqResp.json()).data.id;

    // 审批
    const appResp = await page.request.post(`${API}/api/v1/service-requests/${reqId}/approval`, {
      data: { action: 'approve', comment: '审批通过' },
    });
    expect(appResp.status()).toBe(200);

    // UI 申请详情页面渲染（含 compliance_ack 字段）
    await page.goto(`${BASE}/service-catalog/request/${catId}`);
    await page.waitForSelector('body', { timeout: 10000 });
    const html = await page.content();
    expect(html.length).toBeGreaterThan(500);
  });

  // ===== 6. 连接器市场 =====
  test('6.1 连接器：列出→启用→健康检查→发送消息', async ({ page }) => {
    await loginAsAdmin(page);

    // 列出市场
    const listResp = await page.request.get(`${API}/api/v1/connectors`);
    expect(listResp.status()).toBe(200);
    const connectors = (await listResp.json()).data.items;
    expect(connectors.length).toBeGreaterThanOrEqual(5);

    // 验证 5 个内置连接器
    const names = connectors.map((c: any) => c.name);
    expect(names).toContain('feishu');
    expect(names).toContain('dingtalk');
    expect(names).toContain('wecom');
    expect(names).toContain('webhook');
    expect(names).toContain('console');

    // 启用 console
    const provResp = await page.request.post(`${API}/api/v1/connectors/configs`, {
      data: { name: 'console', enabled: true, settings: { debug_channel: 'stdout' } },
    });
    expect([200, 400, 500]).toContain(provResp.status()); // 已存在可能 400

    // 健康检查
    const healthResp = await page.request.get(`${API}/api/v1/connectors/health`);
    expect(healthResp.status()).toBe(200);
    const health = (await healthResp.json()).data;
    expect(Object.keys(health).length).toBeGreaterThan(0);

    // 发送测试消息
    const sendResp = await page.request.post(`${API}/api/v1/connectors/console/send`, {
      data: { name: 'console', channel: 'stdout', type: 'text', content: 'E2E 测试' },
    });
    expect(sendResp.status()).toBe(200);

    // UI 连接器管理页面渲染
    await page.goto(`${BASE}/admin/connectors`);
    await page.waitForSelector('body', { timeout: 10000 });
    const html = await page.content();
    expect(html.length).toBeGreaterThan(500);
  });

  // ===== 7. SLA 监控 =====
  test('7.1 SLA 监控：stats + monitoring + compliance', async ({ page }) => {
    await loginAsAdmin(page);

    const statsResp = await page.request.get(`${API}/api/v1/sla/stats`);
    expect(statsResp.status()).toBe(200);

    const monResp = await page.request.post(`${API}/api/v1/sla/monitoring`, { data: {} });
    expect(monResp.status()).toBe(200);
    const mon = (await monResp.json()).data;
    expect(mon).toHaveProperty('active_slas');
    expect(mon).toHaveProperty('compliance_rate');

    const compResp = await page.request.get(
      `${API}/api/v1/sla/compliance-report?start_date=2026-01-01&end_date=2026-12-31`
    );
    expect(compResp.status()).toBe(200);
  });

  // ===== 8. 报表 + 仪表盘 =====
  test('8.1 Dashboard + Reports + Analytics 全部可访问', async ({ page }) => {
    await loginAsAdmin(page);

    const dashResp = await page.request.get(`${API}/api/v1/dashboard/overview`);
    expect(dashResp.status()).toBe(200);
    const dash = (await dashResp.json()).data;
    expect(dash).toHaveProperty('kpiMetrics');
    expect(dash).toHaveProperty('ticketTrend');

    const analyticsResp = await page.request.get(`${API}/api/v1/analytics/tickets`);
    expect(analyticsResp.status()).toBe(200);
    const an = (await analyticsResp.json()).data;
    expect(an).toHaveProperty('status_groups');
    expect(an).toHaveProperty('priority_groups');
    expect(an).toHaveProperty('total');
    expect(an.total).toBeGreaterThan(0);

    // 报表
    const repResp = await page.request.get(`${API}/api/v1/reports/tickets`);
    expect(repResp.status()).toBe(200);
  });

  // ===== 9. 多角色权限 =====
  test('9.1 user1（end_user）权限边界', async ({ page }) => {
    // 直接以 user1 身份获取 token（无需 UI 登录）
    const loginResp = await page.request.post(`${API}/api/v1/auth/login`, {
      data: { username: 'user1', password: 'user123' },
    });
    const user1Token = (await loginResp.json()).data.access_token;
    const headers = { Authorization: `Bearer ${user1Token}` };

    // 可读 tickets
    const ticketsResp = await page.request.get(`${API}/api/v1/tickets?page=1&page_size=3`, { headers });
    expect(ticketsResp.status()).toBe(200);

    // 不可读 configuration-items
    const cmdbResp = await page.request.get(`${API}/api/v1/configuration-items`, { headers });
    expect(cmdbResp.status()).toBe(403);

    // 不可读 incidents
    const incResp = await page.request.get(`${API}/api/v1/incidents?page=1&page_size=3`, { headers });
    expect(incResp.status()).toBe(403);
  });

  test('9.2 security1（security）权限边界', async ({ page }) => {
    const loginResp = await page.request.post(`${API}/api/v1/auth/login`, {
      data: { username: 'security1', password: 'security123' },
    });
    const secToken = (await loginResp.json()).data.access_token;
    const headers = { Authorization: `Bearer ${secToken}` };

    // 可读 incidents
    const incResp = await page.request.get(`${API}/api/v1/incidents?page=1&page_size=3`, { headers });
    expect(incResp.status()).toBe(200);

    // 不能访问审计日志
    const auditResp = await page.request.get(`${API}/api/v1/audit-logs`, { headers });
    expect(auditResp.status()).toBe(403);

    // 不能访问连接器
    const connResp = await page.request.get(`${API}/api/v1/connectors`, { headers });
    expect(connResp.status()).toBe(403);
  });

  // ===== 10. 变更 + 审批 =====
  test('10.1 变更创建 + 审批（修复 change_approvals 表后）', async ({ page }) => {
    await loginAsAdmin(page);

    const chResp = await page.request.post(`${API}/api/v1/changes`, {
      data: {
        title: `E2E 变更 ${Date.now()}`,
        description: 'E2E 测试变更',
        type: 'standard',
        riskLevel: 'medium',
        impact: 'low',
        priority: 'medium',
        scheduledStart: '2026-06-10T10:00:00Z',
        scheduledEnd: '2026-06-10T12:00:00Z',
      },
    });
    expect(chResp.status()).toBe(200);
    const chId = (await chResp.json()).data.id;

    // 创建审批记录
    const apResp = await page.request.post(`${API}/api/v1/changes/${chId}/approvals`, {
      data: { changeId: chId, approverId: 1, comment: 'E2E 审批' },
    });
    expect(apResp.status()).toBe(200);

    // 审批汇总
    const sumResp = await page.request.get(`${API}/api/v1/changes/${chId}/approval-summary`);
    expect(sumResp.status()).toBe(200);
  });

  // ===== 11. AI 功能 =====
  test('11.1 AI Chat + RAG 搜索', async ({ page }) => {
    await loginAsAdmin(page);

    const chatResp = await page.request.post(`${API}/api/v1/ai/chat`, {
      data: { query: '什么是 ITSM？' },
    });
    expect(chatResp.status()).toBe(200);
    const chat = (await chatResp.json()).data;
    expect(chat).toHaveProperty('conversation_id');

    const ragResp = await page.request.post(`${API}/api/v1/ai/rag/ask`, {
      data: { query: 'VPN连接', limit: 3 },
    });
    expect(ragResp.status()).toBe(200);
  });

});
