/**
 * Playwright 认证工具
 * 用于 E2E 测试中的登录状态管理
 */

import type { Page} from '@playwright/test';
import { test as base, expect } from '@playwright/test';

/**
 * 执行登录并返回认证后的页面
 */
export async function loginAndReturn(page: Page, username: string = 'admin', password: string = 'admin123') {
  const appURL = process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3000';
  // 清除之前的认证状态
  await page.context().clearCookies();

  // Ant Design v5 inputs can miss React form updates in browser automation.
  // Use the real login API, then seed the same browser auth state the app uses.
  const loginResponse = await page.request.post('http://localhost:8090/api/v1/auth/login', {
    data: { username, password },
    timeout: 30_000,
  });
  if (!loginResponse.ok()) {
    throw new Error(`登录失败: HTTP ${loginResponse.status()}`);
  }

  const loginJson = await loginResponse.json();
  const data = loginJson.data || {};
  const accessToken = data.accessToken || data.access_token;
  const refreshToken = data.refreshToken || data.refresh_token;
  const user = data.user || {};
  if (!accessToken) {
    throw new Error('登录失败: 响应中缺少 access_token');
  }

  const tenantId = Number(user.tenantId || user.tenant_id || 1);
  const tenant = {
    id: tenantId,
    name: '默认租户',
    code: tenantId === 1 ? 'default' : `tenant-${tenantId}`,
    type: 'standard',
    status: 'active',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  };

  await page.context().addCookies([
    {
      name: 'auth-token',
      value: accessToken,
      url: appURL,
      httpOnly: false,
      sameSite: 'Lax',
      expires: Math.floor(Date.now() / 1000) + 900,
    },
    {
      name: 'access_token',
      value: accessToken,
      url: appURL,
      httpOnly: true,
      sameSite: 'Lax',
      expires: Math.floor(Date.now() / 1000) + 900,
    },
    ...(refreshToken
      ? [
          {
            name: 'refresh_token',
            value: refreshToken,
            url: appURL,
            httpOnly: true,
            sameSite: 'Lax' as const,
            expires: Math.floor(Date.now() / 1000) + 604800,
          },
        ]
      : []),
  ]);

  await page.goto('/site.webmanifest', { waitUntil: 'domcontentloaded' });
  await page.evaluate(
    ({ token, currentUser, currentTenant }) => {
      const normalizedUser = {
        id: Number(currentUser.id || 0),
        username: String(currentUser.username || ''),
        email: String(currentUser.email || ''),
        name: String(currentUser.name || currentUser.full_name || ''),
        role: String(currentUser.role || 'end_user'),
        tenantId: currentUser.tenant_id
          ? Number(currentUser.tenant_id)
          : currentUser.tenantId
            ? Number(currentUser.tenantId)
            : undefined,
        department: currentUser.department,
        permissions: currentUser.permissions,
        createdAt: currentUser.created_at || currentUser.createdAt,
        updatedAt: currentUser.updated_at || currentUser.updatedAt,
      };

      localStorage.setItem('access_token', token);
      localStorage.setItem('current_tenant_id', String(currentTenant.id));
      localStorage.setItem('current_tenant_code', currentTenant.code);
      localStorage.setItem(
        'auth-storage',
        JSON.stringify({
          state: {
            user: normalizedUser,
            token: null,
            currentTenant,
            isAuthenticated: true,
          },
          version: 0,
        })
      );
    },
    { token: accessToken, currentUser: user, currentTenant: tenant }
  );

  await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });

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
