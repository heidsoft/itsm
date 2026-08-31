/**
 * FLOW-9: 多租户零跨租户泄露
 * Priority: P1
 *
 * 完整链路: 用户 A 创建资源 → 用户 B（不同租户）无法访问 → 断言明确为 403/404
 *
 * FIXED: 不再接受 [200, 403] 两种结果，明确断言跨租户访问必须返回 403 或 404，
 * 且响应体不得包含对方租户的数据。
 */
import { test, expect } from '../fixtures/auth';

// 直接用 Playwright request 做原始 API 调用，避免 fixture 类型限制
async function rawDelete(
  request: import('@playwright/test').APIRequestContext,
  token: string,
  path: string,
): Promise<{ status: number; data: unknown }> {
  const apiURL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';
  const resp = await request.delete(`${apiURL}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  return { status: resp.status(), data: resp.ok() ? await resp.json() : await resp.text() };
}

async function rawPut(
  request: import('@playwright/test').APIRequestContext,
  token: string,
  path: string,
  body?: unknown,
): Promise<{ status: number; data: unknown }> {
  const apiURL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';
  const resp = await request.put(`${apiURL}${path}`, {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    data: body,
  });
  return { status: resp.status(), data: resp.ok() ? await resp.json() : await resp.text() };
}

test.describe('FLOW-9: 多租户隔离验证', () => {
  test('T074-FLOW9 - 租户隔离：跨租户 GET/LIST/UPDATE/DELETE 必须拒绝', async ({
    loginAs,
    apiGet,
    apiPost,
    request,
  }) => {
    const tenant1Token = await loginAs('tenant1admin');
    const user1Token = await loginAs('user1'); // 不同角色/租户

    // ── Step 1: tenant1admin 创建工单 ──────────────────────────────
    const createResp = await apiPost(tenant1Token, '/api/v1/tickets', {
      title: `FLOW-9-TENANT1-${Date.now()}`,
      description: '此工单属于 tenant1admin',
      priority: 'low',
      category: 'general',
    });
    expect(createResp.status, 'tenant1admin 必须能创建工单').toBe(200);
    const ticket1Id = createResp.data?.data?.id;
    expect(ticket1Id, '返回的 ticket id 不能为空').toBeTruthy();

    // ── Step 2: user1 GET tenant1admin 的工单（按 ID）→ 必须 403 或 404 ──
    const crossGetResp = await apiGet(user1Token, `/api/v1/tickets/${ticket1Id}`);
    expect(
      [403, 404],
      `user1 GET tenant1admin 工单(${ticket1Id})必须被拒绝，实际状态: ${crossGetResp.status}`,
    ).toContain(crossGetResp.status);
    const crossGetBody = JSON.stringify(crossGetResp.data ?? '');
    expect(
      crossGetBody,
      '跨租户响应体不得泄露 tenant1admin 的工单标题',
    ).not.toContain('FLOW-9-TENANT1');

    // ── Step 3: user1 LIST 工单 → 列表中不得包含 tenant1admin 的工单 ──
    const listResp = await apiGet(user1Token, '/api/v1/tickets?page=1&size=50');
    expect(listResp.status, 'user1 LIST 工单应该成功').toBe(200);
    const ticketList: Array<{ id: number; title?: string }> =
      listResp.data?.data?.items ?? listResp.data?.data ?? [];
    const found = ticketList.find(item => item.id === ticket1Id);
    expect(
      found,
      `user1 工单列表中不应包含 tenant1admin 的工单(id=${ticket1Id})，实际列表: ${JSON.stringify(
        ticketList.map(i => i.id),
      )}`,
    ).toBeUndefined();

    // ── Step 4: user1 UPDATE tenant1admin 的工单 → 必须 403 或 404 ─────
    const crossPutResp = await rawPut(request, user1Token, `/api/v1/tickets/${ticket1Id}`, {
      title: '黑客修改',
    });
    expect(
      [403, 404],
      `user1 UPDATE tenant1admin 工单(${ticket1Id})必须被拒绝，实际状态: ${crossPutResp.status}`,
    ).toContain(crossPutResp.status);

    // ── Step 5: user1 DELETE tenant1admin 的工单 → 必须 403 或 404 ─────
    const crossDeleteResp = await rawDelete(request, user1Token, `/api/v1/tickets/${ticket1Id}`);
    expect(
      [403, 404],
      `user1 DELETE tenant1admin 工单(${ticket1Id})必须被拒绝，实际状态: ${crossDeleteResp.status}`,
    ).toContain(crossDeleteResp.status);

    // ── Step 6: 访问不存在的资源 ID → 必须 403 或 404 ─────────────
    const nonExistentGet = await apiGet(tenant1Token, '/api/v1/tickets/999999');
    expect(
      [403, 404],
      `tenant1admin GET 不存在的工单应返回 403/404，实际: ${nonExistentGet.status}`,
    ).toContain(nonExistentGet.status);

    // ── Step 7: 清理 ───────────────────────────────────────────────
    await rawDelete(request, tenant1Token, `/api/v1/tickets/${ticket1Id}`);
  });

  test('T075-FLOW9 - 租户隔离：LIST 必须按 tenant 过滤，不得返回跨租户数据', async ({
    loginAs,
    apiGet,
    apiPost,
    request,
  }) => {
    const adminToken = await loginAs('admin');
    const user1Token = await loginAs('user1');

    // admin 创建工单
    const respAdmin = await apiPost(adminToken, '/api/v1/tickets', {
      title: `FLOW9-ADMIN-${Date.now()}`,
      description: 'admin 的工单',
      priority: 'low',
      category: 'general',
    });
    expect(respAdmin.status).toBe(200);
    const adminTicketId = respAdmin.data?.data?.id;

    // user1 创建工单
    const respUser = await apiPost(user1Token, '/api/v1/tickets', {
      title: `FLOW9-USER1-${Date.now()}`,
      description: 'user1 的工单',
      priority: 'low',
      category: 'general',
    });
    expect(respUser.status).toBe(200);
    const user1TicketId = respUser.data?.data?.id;

    // admin LIST：不得出现 user1 的工单标题
    const listAdmin = await apiGet(adminToken, '/api/v1/tickets?page=1&size=100');
    expect(listAdmin.status).toBe(200);
    const adminTitles: string[] =
      listAdmin.data?.data?.items?.map((i: { title?: string }) => i.title) ??
      listAdmin.data?.data?.map((i: { title?: string }) => i.title) ??
      [];
    expect(
      adminTitles.some(t => t?.includes('FLOW9-USER1')),
      `admin 列表不应包含 user1 的工单，实际标题: ${JSON.stringify(adminTitles)}`,
    ).toBe(false);
    // 必须包含自己的
    expect(
      adminTitles.some(t => t?.includes('FLOW9-ADMIN')),
      `admin 列表必须包含自己的工单`,
    ).toBe(true);

    // user1 LIST：不得出现 admin 的工单标题
    const listUser = await apiGet(user1Token, '/api/v1/tickets?page=1&size=100');
    expect(listUser.status).toBe(200);
    const user1Titles: string[] =
      listUser.data?.data?.items?.map((i: { title?: string }) => i.title) ??
      listUser.data?.data?.map((i: { title?: string }) => i.title) ??
      [];
    expect(
      user1Titles.some(t => t?.includes('FLOW9-ADMIN')),
      `user1 列表不应包含 admin 的工单，实际标题: ${JSON.stringify(user1Titles)}`,
    ).toBe(false);
    expect(
      user1Titles.some(t => t?.includes('FLOW9-USER1')),
      `user1 列表必须包含自己的工单`,
    ).toBe(true);

    // 清理
    if (adminTicketId) await rawDelete(request, adminToken, `/api/v1/tickets/${adminTicketId}`);
    if (user1TicketId) await rawDelete(request, user1Token, `/api/v1/tickets/${user1TicketId}`);
  });
});
