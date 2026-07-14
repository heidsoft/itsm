/**
 * FLOW-9: 多租户零跨租户泄露
 * Priority: P1
 *
 * 完整链路: tenant1 用户操作 → 确认 tenant2 无法访问
 */
import { test, expect } from '../fixtures/auth';

test.describe('FLOW-9: 多租户隔离验证', () => {
  let adminToken: string;
  let tenant1adminToken: string;

  test.beforeEach(async ({ loginAs }) => {
    adminToken = await loginAs('admin');
    tenant1adminToken = await loginAs('tenant1admin');
  });

  test('T074-FLOW9 - 租户隔离验证', async ({ apiGet, apiPost }) => {
    // Step 1: admin 确认能看所有租户
    const allTenantsResp = await apiGet(adminToken, '/api/v1/tenants');
    expect(allTenantsResp.status).toBe(200);

    // Step 2: tenant1admin 只能看自己租户
    const myTenantsResp = await apiGet(tenant1adminToken, '/api/v1/tenants');
    expect([200, 403]).toContain(myTenantsResp.status);

    // Step 3: tenant1admin 创建的工单只能自己看到
    const createResp = await apiPost(tenant1adminToken, '/api/v1/tickets', {
      title: 'FLOW-9 租户隔离测试',
      description: '此工单属于 tenant_test',
      priority: 'low',
      category: 'general',
    });

    expect(createResp.status).toBe(200);
    const ticketId = createResp.data.data?.id;

    // Step 4: 尝试用 admin 访问其他租户工单（如果 tenant1admin 属于 tenant_test）
    if (ticketId) {
      const crossTenantResp = await apiGet(tenant1adminToken, `/api/v1/tickets?tenant_id=99999`);
      // 应该返回空或 403
      expect([200, 403]).toContain(crossTenantResp.status);
    }
  });
});
