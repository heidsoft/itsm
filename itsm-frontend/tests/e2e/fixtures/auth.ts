import { test as base, Page } from '@playwright/test';

/**
 * 角色测试账号映射
 * 与 seeder.go 中 seedRoleTestAccounts 保持一致
 */
export const TEST_ACCOUNTS = {
  admin: { username: 'admin', password: 'AdminProd2026!', role: 'admin' },
  user1: { username: 'user1', password: 'user123456', role: 'end_user' },
  security1: { username: 'security1', password: 'security123456', role: 'security' },
  engineer1: { username: 'engineer1', password: 'eng123456', role: 'technician' },
  manager1: { username: 'manager1', password: 'mgr123456', role: 'manager' },
  tenant1admin: { username: 'tenant1admin', password: 'ta123456', role: 'admin' },
} as const;

// 密码最短长度（后端 RegisterRequest.Password min=6）
const MIN_PASSWORD_LEN = 6;

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
      const apiURL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost';

      // 确保密码满足后端最小长度要求
      const password =
        account.password.length >= MIN_PASSWORD_LEN
          ? account.password
          : account.password + '1'.repeat(MIN_PASSWORD_LEN - account.password.length);

      const login = async (): Promise<any> =>
        request.post(`${apiURL}/api/v1/auth/login`, {
          data: { username: account.username, password },
        });

      const register = async (): Promise<any> =>
        request.post(`${apiURL}/api/v1/auth/register`, {
          data: {
            username: account.username,
            password,
            role: account.role,
            email: `${account.username}@example.com`,
            fullName: account.username,
            tenantId: 1,
          },
        });

      // Token 缓存，避免同一 role 重复登录触发 rate limit
      const cacheKey = `token_cache_${role}`;
      const cachedToken = (globalThis as any)[cacheKey] as string | undefined;
      if (cachedToken) {
        const validateRes = await request.get(`${apiURL}/api/v1/auth/me`, {
          headers: { Authorization: `Bearer ${cachedToken}` },
        });
        if (validateRes.ok()) {
          return cachedToken;
        }
      }

      // 登录，带重试（处理登录限流 429/403）
      let response = await login();
      if (!response.ok()) {
        const bodyText: string = await response.text();
        const status = response.status();
        const isRateLimited = status === 429 || (status === 403 && bodyText.includes('过于频繁'));
        if (isRateLimited && account.role !== 'admin') {
          // 限流时等 3 秒再试
          await new Promise(r => setTimeout(r, 3000));
          response = await login();
        }
      }

      // 非管理员账号若登录失败，尝试自动注册后再登录
      if (!response.ok() && account.role !== 'admin') {
        const regRes = await register();
        // 注册可能因用户已存在而失败（400），此时直接重试登录
        if (regRes.ok()) {
          response = await login();
        } else {
          // 注册失败（用户已存在等），直接重试登录
          response = await login();
        }
      }

      if (!response.ok()) {
        const errBody = await response.text();
        throw new Error(`Login failed for ${role}: ${response.status()} ${errBody}`);
      }

      const json = await response.json();
      const token = json.data?.accessToken || json.data?.access_token;

      if (!token) {
        throw new Error(`No access token for ${role}`);
      }

      // 缓存 token
      (globalThis as any)[cacheKey] = token;

      // 纯 API 测试不需要页面 cookie，token 直接通过 apiPost/apiGet 使用
      return token;
    });
  },

  apiGet: async ({ request }, use) => {
    await use(async (token: string, path: string) => {
      const apiURL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost';
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
      const apiURL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost';
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
