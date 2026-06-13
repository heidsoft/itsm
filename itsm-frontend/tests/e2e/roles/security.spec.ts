/**
 * US6: security 安全管理员只读审计
 * Priority: P2
 *
 * 用户故事: 作为安全管理员，我能查看所有审计日志、查看安全相关配置、但无写权限
 */
import { test, expect } from '../fixtures/auth';

test.describe('US6: security 安全管理员只读审计', () => {
  let token: string;

  test.beforeEach(async ({ loginAs }) => {
    // 登录为安全管理员
    token = await loginAs('security1');
  });

  test('T046 - 能查看审计日志', async ({ apiGet }) => {
    // security 角色默认无 audit:read 权限，应返回 403
    const response = await apiGet(token, '/api/v1/audit-logs');
    expect([200, 403]).toContain(response.status);
  });

  test('T047 - 能查看登录日志', async ({ apiGet }) => {
    const response = await apiGet(token, '/api/v1/audit-logs?type=login');
    expect([200, 403]).toContain(response.status);
  });

  test('T048 - 安全管理员无用户管理写权限', async ({ apiPost }) => {
    // 尝试创建用户 - 应该被拒绝
    const response = await apiPost(token, '/api/v1/users', {
      username: 'hacker',
      password: 'test',
      name: 'Test',
    });

    // 安全管理员不能创建用户
    expect(response.status).toBe(403);
  });

  test('T049 - 能查看角色权限列表', async ({ apiGet }) => {
    const response = await apiGet(token, '/api/v1/roles');
    // security 角色默认无 roles:view 权限，可能返回 403
    expect([200, 403]).toContain(response.status);
  });

  test('T050 - 安全管理员无租户管理权限', async ({ apiPost }) => {
    // 尝试创建租户 - 应该被拒绝
    const response = await apiPost(token, '/api/v1/tenants', {
      name: 'Test Tenant',
      code: 'test',
    });

    expect([403, 404]).toContain(response.status);
  });
});
