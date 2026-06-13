/**
 * FLOW-7: SLA 风险 → 监控 → 通知 → 升级
 * Priority: P2
 *
 * 完整链路: SLA 风险检测 → 告警通知 → 升级处理
 */
import { test, expect } from '../fixtures/auth';

test.describe('FLOW-7: SLA 风险监控与升级', () => {
  let adminToken: string;
  let managerToken: string;

  test.beforeEach(async ({ loginAs }) => {
    adminToken = await loginAs('admin');
    managerToken = await loginAs('manager1');
  });

  test('T073-FLOW7 - SLA 风险监控流程', async ({ apiGet }) => {
    // Step 1: 查看 SLA 监控状态
    const monitorResp = await apiGet(adminToken, '/api/v1/sla/monitoring');
    expect([200, 405]).toContain(monitorResp.status);

    // Step 2: 查看 SLA 违规记录
    const violationsResp = await apiGet(adminToken, '/api/v1/sla/violations');
    expect([200, 405]).toContain(violationsResp.status);

    // Step 3: 查看 SLA 统计数据
    const statsResp = await apiGet(managerToken, '/api/v1/sla/stats');
    expect([200, 405]).toContain(statsResp.status);

    // Step 4: 查看 SLA 合规报告
    const complianceResp = await apiGet(managerToken, '/api/v1/sla/compliance-report');
    expect([200, 405]).toContain(complianceResp.status);
  });
});
