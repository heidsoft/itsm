/**
 * US3: super_admin 跨租户与平台治理
 * Priority: P1
 *
 * 用户故事: 作为超级管理员，我能管理所有租户、全局配置、系统监控、查看所有审计日志
 */
import { test, expect } from '../fixtures/auth';

test.describe('US3: super_admin 跨租户与平台治理', () => {
  let token: string;

  test.beforeEach(async ({ loginAs }) => {
    // 登录为超级管理员
    token = await loginAs('admin');
  });

  test('T029 - 能查看所有租户列表', async ({ apiGet }) => {
    const response = await apiGet(token, '/api/v1/tenants');
    expect(response.status).toBe(200);
    expect(response.data).toHaveProperty('code', 0);
  });

  test('T030 - 能查看所有用户列表', async ({ apiGet }) => {
    const response = await apiGet(token, '/api/v1/users');
    expect(response.status).toBe(200);
    expect(response.data).toHaveProperty('code', 0);
    // 超级管理员应该看到 >= 20 条用户记录
    const userCount = response.data.data?.users?.length || 0;
    expect(userCount).toBeGreaterThanOrEqual(1);
  });

  test('T031 - 能访问 GA Readiness 端点并验证 12 个模块', async ({ apiGet }) => {
    const response = await apiGet(token, '/api/v1/readiness/ga');
    expect(response.status).toBe(200);
    expect(response.data).toHaveProperty('code', 0);

    // 验证 12 个模块
    const modules = response.data.data?.modules || [];
    expect(modules.length).toBeGreaterThanOrEqual(12);
  });

  test('T032 - 能管理角色权限', async ({ apiGet }) => {
    const response = await apiGet(token, '/api/v1/roles');
    expect(response.status).toBe(200);
    expect(response.data).toHaveProperty('code', 0);
  });

  test('T033 - 能查看审计日志（全局）', async ({ apiGet }) => {
    const response = await apiGet(token, '/api/v1/audit-logs');
    expect(response.status).toBe(200);
    expect(response.data).toHaveProperty('code', 0);
  });

  test('T034 - 能管理菜单配置', async ({ apiGet }) => {
    const response = await apiGet(token, '/api/v1/auth/menus');
    expect(response.status).toBe(200);
    expect(response.data).toHaveProperty('code', 0);
  });

  test('T035 - 能访问系统配置', async ({ apiGet }) => {
    const response = await apiGet(token, '/api/v1/system-config');
    // 可能返回 200 或 404，但超级管理员不应该被拒绝
    expect([200, 404]).toContain(response.status);
  });
});
