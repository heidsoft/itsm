/**
 * Navigation E2E Tests
 * 测试导航功能和页面访问
 */

import { test, expect } from '@playwright/test';

test.describe('公开页面 - 无需认证', () => {
  test('should show login page', async ({ page }) => {
    await page.goto('/login');
    await expect(page.locator('form')).toBeVisible();
  });

  test('should redirect root to login', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveURL(/\/login/);
  });
});

test.describe('导航测试 - 需要登录', () => {
  test.beforeEach(async ({ page }) => {
    // 登录
    await page.goto('/login');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForSelector('.ant-input, input.ant-input', { timeout: 15000 });
    const inputs = page.locator('input.ant-input');
    await inputs.nth(0).fill('admin');
    await inputs.nth(1).fill('admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(dashboard|tickets|incidents)/, { timeout: 20000 });
  });

  test('should have sidebar navigation menu', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // 查找侧边栏
    const sidebar = page.locator('.ant-layout-sider');
    await expect(sidebar).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to tickets page', async ({ page }) => {
    // 直接通过 URL 导航到工单页面
    await page.goto('/tickets');
    await page.waitForURL(/\/tickets/);
    await expect(page.locator('h1, h2')).toBeVisible();
  });

  test('should navigate to dashboard', async ({ page }) => {
    // 直接通过 URL 导航到仪表盘页面
    await page.goto('/dashboard');
    await page.waitForURL(/\/dashboard/);
    await expect(page.locator('h1, h2')).toBeVisible();
  });

  test('should have proper page title', async ({ page }) => {
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    // 检查页面有标题
    const title = page.locator('h1, h2, [class*="title"]');
    await expect(title.first()).toBeVisible();
  });
});

test.describe('响应式设计 - 需要登录', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForSelector('.ant-input, input.ant-input', { timeout: 15000 });
    const inputs = page.locator('input.ant-input');
    await inputs.nth(0).fill('admin');
    await inputs.nth(1).fill('admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(dashboard|tickets|incidents)/, { timeout: 20000 });
  });

  test('should display correctly on desktop', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    // 桌面端应该显示完整侧边栏
    const sidebar = page.locator('.ant-layout-sider');
    await expect(sidebar).toBeVisible();
  });

  test('should display correctly on tablet', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    // 平板端应该可以折叠侧边栏
    const content = page.locator('.ant-layout-content');
    await expect(content).toBeVisible();
  });
});
