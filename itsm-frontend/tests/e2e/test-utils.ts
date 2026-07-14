/**
 * Playwright 测试配置和辅助函数
 * 统一登录管理和选择器
 */

import type { Page } from '@playwright/test';
import { test as base } from '@playwright/test';

/**
 * 统一的登录辅助函数
 * 在多个测试文件中使用
 */
export async function loginAsAdmin(page: Page) {
  await page.goto('/login');
  await page.waitForLoadState('domcontentloaded');

  // 等待 Ant Design 表单加载
  await page.waitForSelector('.ant-input, input.ant-input', { timeout: 15000 });

  const inputs = page.locator('input.ant-input');
  await inputs.nth(0).fill('admin');
  await inputs.nth(1).fill('admin123');

  await page.click('button[type="submit"]');
  await page.waitForURL(/\/(dashboard|tickets|incidents)/, { timeout: 20000 });
}

/**
 * 等待页面加载完成
 */
export async function waitForPageLoad(page: Page) {
  await page.waitForLoadState('domcontentloaded');
  await page.waitForLoadState('networkidle');
}

/**
 * 扩展的 test 类型
 */
export const test = base.extend<{
  authenticatedPage: Page;
}>({
  authenticatedPage: async ({ page }, use) => {
    await loginAsAdmin(page);
    await use(page);
  },
});

export { expect } from '@playwright/test';
