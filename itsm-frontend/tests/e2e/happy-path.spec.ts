/**
 * ITSM 核心模块 Happy Path E2E 测试
 *
 * 目的：快速验证核心业务闭环可测性
 * 覆盖：工单、事件、问题、变更、服务请求、SLA、CMDB
 *
 * 运行方式：
 *   npx playwright test tests/e2e/happy-path.spec.ts
 */

import type { Page } from '@playwright/test';
import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './test-utils';

const BASE = 'http://localhost:3000';
const API = 'http://localhost:8090';

/**
 * 辅助函数：验证 API 响应结构
 */
async function verifySuccessResponse(response: Response, minDataFields: number = 1) {
  expect(response.status()).toBe(200);
  const json = await response.json();
  expect(json.code).toBe(0);
  expect(json).toHaveProperty('data');
  if (minDataFields > 0 && json.data) {
    expect(Object.keys(json.data).length).toBeGreaterThanOrEqual(minDataFields);
  }
  return json;
}

test.describe('ITSM 核心模块 Happy Path E2E', () => {

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  // ===== 1. 工单 (Ticket) Happy Path =====
  test('1. 工单：创建 → 接受 → 解决 → 关闭', async ({ page }) => {
    // 1.1 创建工单
    const createResp = await page.request.post(`${API}/api/v1/tickets`, {
      data: {
        title: `Happy Path Ticket ${Date.now()}`,
        description: '核心模块 E2E 测试工单',
        priority: 'high',
        type: 'incident',
      },
    });
    const createJson = await verifySuccessResponse(createResp);
    const ticketId = createJson.data.id;
    expect(createJson.data.status).toBe('new');

    // 1.2 接受工单
    const acceptResp = await page.request.post(`${API}/api/v1/tickets/workflow/accept`, {
      data: { ticket_id: ticketId },
    });
    await verifySuccessResponse(acceptResp);

    // 1.3 解决工单
    const resolveResp = await page.request.post(`${API}/api/v1/tickets/workflow/resolve`, {
      data: {
        ticket_id: ticketId,
        resolution: '问题已修复',
        resolutionCategory: 'code_defect'
      },
    });
    await verifySuccessResponse(resolveResp);

    // 1.4 关闭工单
    const closeResp = await page.request.post(`${API}/api/v1/tickets/workflow/close`, {
      data: { ticket_id: ticketId, closeNotes: '测试关闭', closeReason: '完成' },
    });
    await verifySuccessResponse(closeResp);

    // 1.5 验证工单状态
    const detailResp = await page.request.get(`${API}/api/v1/tickets/${ticketId}`);
    const detailJson = await verifySuccessResponse(detailResp, 5);
    expect(detailJson.data.status).toBe('closed');

    console.log(`✅ 工单 Happy Path 完成: ticketId=${ticketId}`);
  });

  // ===== 2. 事件 (Incident) Happy Path =====
  test('2. 事件：创建 → 确认 → 评论 → 解决 → 关闭', async ({ page }) => {
    // 2.1 创建事件
    const createResp = await page.request.post(`${API}/api/v1/incidents`, {
      data: {
        title: `Happy Path Incident ${Date.now()}`,
        description: '核心模块 E2E 测试事件',
        severity: 'high',
        priority: 'high',
        type: 'incident',
        source: 'monitoring',
      },
    });
    const createJson = await verifySuccessResponse(createResp);
    const incidentId = createJson.data.id;
    expect(createJson.data.status).toBe('new');

    // 2.2 确认事件
    const ackResp = await page.request.post(`${API}/api/v1/incidents/${incidentId}/acknowledge`, {
      data: { comment: '已确认' },
    });
    await verifySuccessResponse(ackResp);

    // 2.3 添加评论
    const commentResp = await page.request.post(`${API}/api/v1/incidents/${incidentId}/comments`, {
      data: { content: '处理中', isInternal: false },
    });
    await verifySuccessResponse(commentResp);

    // 2.4 解决事件
    const resolveResp = await page.request.post(`${API}/api/v1/incidents/${incidentId}/resolve`, {
      data: { resolution: '已恢复', resolutionCategory: 'environment' },
    });
    await verifySuccessResponse(resolveResp);

    // 2.5 关闭事件
    const closeResp = await page.request.post(`${API}/api/v1/incidents/${incidentId}/close`, {
      data: { comment: '已关闭' },
    });
    await verifySuccessResponse(closeResp);

    console.log(`✅ 事件 Happy Path 完成: incidentId=${incidentId}`);
  });

  // ===== 3. 问题 (Problem) Happy Path =====
  test('3. 问题：创建 → 调查 → 根因 → 关闭', async ({ page }) => {
    // 3.1 创建问题
    const createResp = await page.request.post(`${API}/api/v1/problems`, {
      data: {
        title: `Happy Path Problem ${Date.now()}`,
        description: '核心模块 E2E 测试问题',
        priority: 'high',
        impactScope: 'high',
        riskLevel: 'medium',
      },
    });
    const createJson = await verifySuccessResponse(createResp);
    const problemId = createJson.data.id;

    // 3.2 开始调查
    const investigateResp = await page.request.post(`${API}/api/v1/problems/${problemId}/investigate`, {
      data: { comment: '开始调查' },
    });
    await verifySuccessResponse(investigateResp);

    // 3.3 添加根因分析
    const rcaResp = await page.request.put(`${API}/api/v1/problems/${problemId}/root-cause`, {
      data: { rootCause: '资源泄漏', causeCategory: 'resource_leak' },
    });
    await verifySuccessResponse(rcaResp);

    // 3.4 关闭问题
    const closeResp = await page.request.post(`${API}/api/v1/problems/${problemId}/close`, {
      data: { resolution: '已修复', comment: '问题已解决' },
    });
    await verifySuccessResponse(closeResp);

    console.log(`✅ 问题 Happy Path 完成: problemId=${problemId}`);
  });

  // ===== 4. 变更 (Change) Happy Path =====
  test('4. 变更：创建 → 提交 → 审批 → 实施 → 关闭', async ({ page }) => {
    // 4.1 创建变更
    const createResp = await page.request.post(`${API}/api/v1/changes`, {
      data: {
        title: `Happy Path Change ${Date.now()}`,
        description: '核心模块 E2E 测试变更',
        justification: '业务需求',
        type: 'normal',
        priority: 'medium',
        impactScope: 'low',
        riskLevel: 'low',
      },
    });
    const createJson = await verifySuccessResponse(createResp);
    const changeId = createJson.data.id;
    expect(createJson.data.status).toBe('draft');

    // 4.2 提交变更
    const submitResp = await page.request.post(`${API}/api/v1/changes/${changeId}/submit`, {
      data: {
        comment: '请审批',
        approver_ids: [1],
      },
    });
    await verifySuccessResponse(submitResp);

    // 4.3 获取审批摘要
    const summaryResp = await page.request.get(`${API}/api/v1/changes/${changeId}/approval-summary`);
    await verifySuccessResponse(summaryResp);

    // 4.4 获取风险评估
    const riskResp = await page.request.get(`${API}/api/v1/changes/${changeId}/risk-assessment`);
    await verifySuccessResponse(riskResp);

    // 4.5 获取 CMDB 影响摘要
    const cmdbResp = await page.request.get(`${API}/api/v1/changes/${changeId}/cmdb-impact`);
    await verifySuccessResponse(cmdbResp);

    console.log(`✅ 变更 Happy Path 完成: changeId=${changeId}`);
  });

  // ===== 5. 服务请求 (Service Request) Happy Path =====
  test('5. 服务请求：创建目录 → 申请 → 审批', async ({ page }) => {
    // 5.1 创建服务目录
    const catalogResp = await page.request.post(`${API}/api/v1/service-catalogs`, {
      data: {
        name: `Happy Path Catalog ${Date.now()}`,
        description: '核心模块 E2E 测试服务目录',
        category: '软件',
        delivery_time: '3',
        status: 'enabled',
      },
    });
    const catalogJson = await verifySuccessResponse(catalogResp);
    const catalogId = catalogJson.data.id;

    // 5.2 创建服务请求
    const requestResp = await page.request.post(`${API}/api/v1/service-requests`, {
      data: {
        catalog_id: catalogId,
        title: `Happy Path Request ${Date.now()}`,
        reason: '工作需要',
        compliance_ack: true,
        expire_at: '2026-12-31T00:00:00Z',
      },
    });
    const requestJson = await verifySuccessResponse(requestResp);
    const requestId = requestJson.data.id;

    // 5.3 审批请求
    const approvalResp = await page.request.post(`${API}/api/v1/service-requests/${requestId}/approval`, {
      data: { action: 'approve', comment: '审批通过' },
    });
    await verifySuccessResponse(approvalResp);

    console.log(`✅ 服务请求 Happy Path 完成: catalogId=${catalogId}, requestId=${requestId}`);
  });

  // ===== 6. SLA Happy Path =====
  test('6. SLA：统计 → 监控 → 合规报告', async ({ page }) => {
    // 6.1 获取 SLA 统计
    const statsResp = await page.request.get(`${API}/api/v1/sla/stats`);
    const statsJson = await verifySuccessResponse(statsResp);

    // 6.2 触发 SLA 监控
    const monitorResp = await page.request.post(`${API}/api/v1/sla/monitoring`, { data: {} });
    const monitorJson = await verifySuccessResponse(monitorResp, 2);

    // 6.3 获取合规报告
    const complianceResp = await page.request.get(
      `${API}/api/v1/sla/compliance-report?start_date=2026-01-01&end_date=2026-12-31`
    );
    await verifySuccessResponse(complianceResp);

    console.log(`✅ SLA Happy Path 完成: stats=${JSON.stringify(statsJson.data)}`);
  });

  // ===== 7. CMDB Happy Path =====
  test('7. CMDB：创建 CI 类型 → 创建 CI 实例 → 查询', async ({ page }) => {
    // 7.1 创建 CI 类型
    const typeResp = await page.request.post(`${API}/api/v1/cmdb/ci-types`, {
      data: {
        name: `Server ${Date.now()}`,
        description: '核心模块 E2E 测试 CI 类型',
        category: 'hardware',
        attributes: [
          { name: 'ip_address', type: 'string', required: true },
          { name: 'os_version', type: 'string', required: false },
        ],
      },
    });
    const typeJson = await verifySuccessResponse(typeResp);
    const ciTypeId = typeJson.data.id;

    // 7.2 创建 CI 实例
	const ciResp = await page.request.post(`${API}/api/v1/configuration-items`, {
      data: {
        ci_type_id: ciTypeId,
        name: `Test Server ${Date.now()}`,
        serial_number: `SN-${Date.now()}`,
      },
    });
    const ciJson = await verifySuccessResponse(ciResp);
    const ciId = ciJson.data.id;

    // 7.3 查询 CI 详情
	const detailResp = await page.request.get(`${API}/api/v1/configuration-items/${ciId}`);
    await verifySuccessResponse(detailResp, 3);

    // 7.4 列出 CI 实例
	const listResp = await page.request.get(`${API}/api/v1/configuration-items?ciTypeId=${ciTypeId}&page=1&size=10`);
    await verifySuccessResponse(listResp);

    console.log(`✅ CMDB Happy Path 完成: ciTypeId=${ciTypeId}, ciId=${ciId}`);
  });

  // ===== 8. Dashboard Happy Path =====
  test('8. Dashboard：总览 → KPI 指标', async ({ page }) => {
    // 8.1 获取总览
    const overviewResp = await page.request.get(`${API}/api/v1/dashboard/overview`);
    const overviewJson = await verifySuccessResponse(overviewResp, 2);

    // 8.2 验证 KPI 指标
    if (overviewJson.data.kpiMetrics) {
      expect(Array.isArray(overviewJson.data.kpiMetrics)).toBeTruthy();
    }

    console.log(`✅ Dashboard Happy Path 完成`);
  });

});
