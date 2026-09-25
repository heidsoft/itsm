/**
 * Playwright 认证工具
 * 用于 E2E 测试中的登录状态管理
 */

import type { Page} from '@playwright/test';
import { test as base, expect } from '@playwright/test';

/**
 * 执行登录并返回认证后的页面
 *
 * 鉴权自令牌 Cookie 化（仅 HttpOnly cookie）后，登录响应不再返回 access_token：
 * 这里只依赖 page.request 与页面共享的 cookie 罐，不再手工种 cookie/localStorage。
 * 口令必须来自环境变量，禁止在仓库里硬编码凭据。
 */
export async function loginAndReturn(
  page: Page,
  username: string = 'admin',
  password?: string,
  landingPath: string = '/dashboard'
) {
  const secret = password || process.env.E2E_ADMIN_PASSWORD || process.env.ADMIN_PASSWORD;
  if (!secret) {
    throw new Error('缺少 E2E_ADMIN_PASSWORD（或 ADMIN_PASSWORD）：e2e 不接受硬编码口令');
  }
  await page.context().clearCookies();

  // 相对路径走页面同源；page.request 与浏览器上下文共享 cookie 罐。
  const loginResponse = await page.request.post('/api/v1/auth/login', {
    data: { username, password: secret },
    timeout: 30_000,
  });
  if (!loginResponse.ok()) {
    throw new Error(`登录失败: HTTP ${loginResponse.status()} ${await loginResponse.text()}`);
  }

  // 兼容仍返回令牌的旧形态（仅用于填充本地会话元数据，缺失不报错）。
  const loginJson = await loginResponse.json().catch(() => ({ data: {} }));
  const data = loginJson.data || {};
  const user = data.user || {};
  const tenantId = Number(user.tenantId || user.tenant_id || 1);
  await page.goto(landingPath, { waitUntil: 'networkidle' });
  if (page.url().includes('/login')) {
    throw new Error(`登录后被重定向到 ${page.url()}：会话未生效`);
  }
  // 供依赖本地会话元数据的旧代码读取；token 字段留空，真源是 HttpOnly cookie。
  await page.evaluate(
    ({ currentUser, currentTenant }) => {
      localStorage.setItem('current_tenant_id', String(currentTenant.id));
      localStorage.setItem('current_tenant_code', currentTenant.code);
      localStorage.setItem(
        'auth-storage',
        JSON.stringify({
          state: {
            user: currentUser,
            token: null,
            currentTenant,
            isAuthenticated: true,
          },
          version: 0,
        })
      );
    },
    {
      currentUser: { ...user, tenantId },
      currentTenant: {
        id: tenantId,
        name: '默认租户',
        code: tenantId === 1 ? 'default' : `tenant-${tenantId}`,
        type: 'standard',
        status: 'active',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      },
    }
  );

  return page;
}

/**
 * 创建带认证的测试配置
 */
export const test = base.extend<{
  authenticatedPage: Page;
}>({
  authenticatedPage: async ({ page }, use) => {
    await loginAndReturn(page);
    await use(page);
  },
});

/**
 * 导出 expect 来自 playwright
 */
export { expect };
