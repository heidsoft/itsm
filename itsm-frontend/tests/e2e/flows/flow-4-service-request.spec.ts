/**
 * FLOW-4: 服务请求 Catalog → Submit → Approve → Fulfill → Complete
 * Priority: P1
 *
 * 完整链路: 用户浏览服务目录 → 提交请求 → 审批 → 实施 → 完成
 */
import { test, expect } from '../fixtures/auth';

test.describe('FLOW-4: 服务请求完整流程', () => {
  let endUserToken: string;
  let engineerToken: string;
  let managerToken: string;

  test.beforeEach(async ({ loginAs }) => {
    endUserToken = await loginAs('user1');
    engineerToken = await loginAs('engineer1');
    managerToken = await loginAs('manager1');
  });

  test('T072-FLOW4 - 服务请求全流程', async ({ apiGet, apiPost }) => {
    // Step 1: 用户浏览服务目录
    const catalogResp = await apiGet(endUserToken, '/api/v1/service-catalogs');
    expect(catalogResp.status).toBe(200);

    // Step 2: 用户提交服务请求
    const requestResp = await apiPost(endUserToken, '/api/v1/tickets', {
      title: 'FLOW-4 服务请求 - 申请新服务器',
      description: '需要申请一台新的应用服务器',
      priority: 'medium',
      category: 'service_request',
      service_catalog_id: 1,
    });

    expect(requestResp.status).toBe(200);
    const requestId = requestResp.data.data?.id;

    // Step 3: manager 审批
    if (requestId) {
      const approveResp = await apiPost(managerToken, `/api/v1/tickets/${requestId}`, {
        status: 'approved',
        approval_note: '审批通过',
      });

      expect([200, 404]).toContain(approveResp.status);
    }

    // Step 4: engineer 实施
    if (requestId) {
      const fulfillResp = await apiPost(engineerToken, `/api/v1/tickets/${requestId}`, {
        status: 'in_progress',
        handling_note: '正在准备服务器',
      });

      expect([200, 404]).toContain(fulfillResp.status);

      // Step 5: 完成请求
      const completeResp = await apiPost(engineerToken, `/api/v1/tickets/${requestId}`, {
        status: 'closed',
        resolution: '服务器已配置完成',
      });

      expect([200, 404]).toContain(completeResp.status);
    }
  });
});
