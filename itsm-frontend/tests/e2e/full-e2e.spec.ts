/**
 * ITSM 全流程 E2E 测试
 * 包含登录、导航、核心业务功能测试
 */

import { test, expect } from '@playwright/test';

/**
 * 辅助函数：执行登录
 */
async function login(page: any) {
  await page.goto('/login');
  await page.waitForLoadState('domcontentloaded');
  await page.waitForSelector('.ant-input, input.ant-input', { timeout: 15000 });
  const inputs = page.locator('input.ant-input');
  await inputs.nth(0).fill('admin');
  await inputs.nth(1).fill('admin123');
  await page.click('button[type="submit"]');

  // 等待页面跳转或 dashboard 出现
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(1000);

  await page.waitForURL(/\/(dashboard|tickets|incidents)/, { timeout: 20000 }).catch(() => {
    // 如果超时，检查是否仍在登录页
    if (page.url().includes('/login')) {
      throw new Error('登录失败，仍在登录页');
    }
  });
}

test.describe('ITSM 全流程测试 - 需要登录', () => {
  // 使用 beforeEach 进行登录，确保每个测试都是独立认证状态
  test.beforeEach(async ({ page }) => {
    // 先导航到登录页，确保在正确的上下文中
    await page.goto('/login');
    await page.waitForLoadState('domcontentloaded');

    // 清除之前的认证状态
    await page.context().clearCookies();
    await page.evaluate(() => {
      try {
        localStorage.clear();
        document.cookie.split(';').forEach(c => {
          const name = c.replace(/^ +/, '').replace(/=.*/, '');
          if (name) {
            document.cookie = name + '=;expires=' + new Date().toUTCString() + ';path=/';
          }
        });
      } catch (e) {
        // Ignore storage errors
      }
    });
    await login(page);
  });

  test('should login successfully and redirect to dashboard', async ({ page }) => {
    // 验证已登录（不在登录页）
    await expect(page).not.toHaveURL(/\/login/);
    // 验证 URL 包含主要页面路径
    await expect(page).toHaveURL(/\/(dashboard|tickets|incidents)/);
  });

  test('should display dashboard with menu', async ({ page }) => {
    // 等待页面加载完成
    await page.waitForLoadState('networkidle');

    // 检查侧边栏菜单是否存在（使用 Ant Design 的类名）
    const sidebar = page.locator('.ant-layout-sider, [class*="sider"]');
    await expect(sidebar.first()).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to tickets page', async ({ page }) => {
    // 直接导航到工单页面
    await page.goto('/tickets');
    await page.waitForURL(/\/tickets/);
    await expect(page.locator('h1, h2')).toBeVisible();
  });

  test('should navigate to incidents page', async ({ page }) => {
    // 直接导航到事件页面
    await page.goto('/incidents');
    await page.waitForURL(/\/incidents/);
    await expect(page.locator('h1, h2')).toBeVisible();
  });

  test('should navigate to CMDB page', async ({ page }) => {
    // 直接导航到 CMDB 页面
    await page.goto('/cmdb');
    await page.waitForURL(/\/cmdb/);
    // 找到任意一个标题即可
    await expect(page.locator('h1, h2').first()).toBeVisible();
  });

  test('should navigate to knowledge base', async ({ page }) => {
    // 直接导航到知识库页面
    await page.goto('/knowledge');
    await page.waitForURL(/\/knowledge/);
    await expect(page.locator('h1, h2')).toBeVisible();
  });

  test('should navigate to admin page', async ({ page }) => {
    // 直接导航到管理页面
    await page.goto('/admin');
    await page.waitForURL(/\/admin/);
    await expect(page.locator('h1, h2').first()).toBeVisible();
  });
});

test.describe('ITSM 公开页面 - 无需登录', () => {
  test('should show login page', async ({ page }) => {
    await page.goto('/login');
    await page.waitForLoadState('domcontentloaded');
    await expect(page.locator('form')).toBeVisible({ timeout: 15000 });
    // 使用 Ant Design 输入框选择器
    await expect(page.locator('input.ant-input').first()).toBeVisible();
  });

  test('should redirect root to login', async ({ page }) => {
    await page.goto('/');
    // 根路径会重定向到登录页
    await expect(page).toHaveURL(/\/login/);
  });
});

test.describe('ITSM 权限测试', () => {
  test('should redirect to login when accessing protected route', async ({ page }) => {
    // 直接访问受保护的路由
    await page.goto('/tickets');
    await expect(page).toHaveURL(/\/login/);
  });

  test('should redirect to login when accessing admin route without auth', async ({ page }) => {
    await page.goto('/admin');
    await expect(page).toHaveURL(/\/login/);
  });
});
