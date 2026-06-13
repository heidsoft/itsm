import { test as base, Page } from '@playwright/test';

/**
 * 角色测试账号映射
 * 与 seeder.go 中 seedRoleTestAccounts 保持一致
 */
export const TEST_ACCOUNTS = {
  admin: { username: 'admin', password: 'admin123', role: 'admin' },
  user1: { username: 'user1', password: 'user123', role: 'end_user' },
  security1: { username: 'security1', password: 'security123', role: 'security' },
  engineer1: { username: 'engineer1', password: 'eng123', role: 'technician' },
  manager1: { username: 'manager1', password: 'mgr123', role: 'manager' },
  tenant1admin: { username: 'tenant1admin', password: 'ta123', role: 'admin' },
} as const;

export type TestRole = keyof typeof TEST_ACCOUNTS;

// 扩展 Playwright test 类型
interface TestFixtures {
  loginAs: (role: TestRole) => Promise<string>;
  apiGet: (token: string, path: string) => Promise<any>;
  apiPost: (token: string, path: string, body?: any) => Promise<any>;
}

export const test = base.extend<TestFixtures>({
  loginAs: async ({ page, request }, use) => {
    await use(async (role: TestRole) => {
      const account = TEST_ACCOUNTS[role];
      const baseURL = process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3000';
      const apiURL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';

      // 调用登录 API
      const response = await request.post(`${apiURL}/api/v1/auth/login`, {
        data: {
          username: account.username,
          password: account.password,
        },
      });

      if (!response.ok()) {
        throw new Error(`Login failed for ${role}: ${response.status()}`);
      }

      const json = await response.json();
      const token = json.data?.access_token;

      if (!token) {
        throw new Error(`No access token for ${role}`);
      }

      // 设置页面 cookie（用于 UI 测试）
      await page.goto(`${baseURL}/login`);
      await page.evaluate(
        (t) => {
          document.cookie = `token=${t}; path=/`;
        },
        token
      );

      return token;
    });
  },

  apiGet: async ({ request }, use) => {
    await use(async (token: string, path: string) => {
      const apiURL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';
      const response = await request.get(`${apiURL}${path}`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      return {
        status: response.status(),
        data: response.ok() ? await response.json() : await response.text(),
      };
    });
  },

  apiPost: async ({ request }, use) => {
    await use(async (token: string, path: string, body?: any) => {
      const apiURL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';
      const response = await request.post(`${apiURL}${path}`, {
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        data: body,
      });
      return {
        status: response.status(),
        data: response.ok() ? await response.json() : await response.text(),
      };
    });
  },
});

export { expect } from '@playwright/test';
