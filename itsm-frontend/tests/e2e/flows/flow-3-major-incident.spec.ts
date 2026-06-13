/**
 * FLOW-3: 重大事件多角色协作
 * Priority: P1
 *
 * 完整链路: 重大事件 → 升级 → 多团队协作 → 解决
 */
import { test, expect } from '../fixtures/auth';

test.describe('FLOW-3: 重大事件多角色协作', () => {
  let endUserToken: string;
  let engineerToken: string;
  let managerToken: string;

  test.beforeEach(async ({ loginAs }) => {
    endUserToken = await loginAs('user1');
    engineerToken = await loginAs('engineer1');
    managerToken = await loginAs('manager1');
  });

  test('T071-FLOW3 - 重大事件处理流程', async ({ apiPost, apiGet }) => {
    // Step 1: 创建重大事件
    const incidentResp = await apiPost(endUserToken, '/api/v1/incidents', {
      title: 'FLOW-3 重大事件 - 核心系统故障',
      description: '核心交易系统宕机，影响所有用户',
      priority: 'critical',
      category: 'major_incident',
    });

    expect(incidentResp.status).toBe(200);
    const incidentId = incidentResp.data.data?.id;

    // Step 2: manager 确认为重大事件
    if (incidentId) {
      const escalateResp = await apiPost(managerToken, `/api/v1/incidents/${incidentId}`, {
        status: 'escalated',
        escalation_note: '确认为重大事件',
      });

      expect([200, 404]).toContain(escalateResp.status);
    }

    // Step 3: engineer 处理事件
    if (incidentId) {
      const handleResp = await apiPost(engineerToken, `/api/v1/incidents/${incidentId}`, {
        status: 'in_progress',
        handling_note: '正在排查问题',
      });

      expect(handleResp.status).toBe(200);

      // Step 4: 事件解决
      const resolveResp = await apiPost(engineerToken, `/api/v1/incidents/${incidentId}`, {
        status: 'resolved',
        resolution: '已修复核心问题，系统恢复',
      });

      expect(resolveResp.status).toBe(200);
    }
  });
});
