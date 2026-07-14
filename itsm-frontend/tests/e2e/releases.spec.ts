/**
 * Release Management E2E Tests
 * 发布管理模块测试
 */

import { test, expect } from '@playwright/test';

test.describe('Release Management - 发布管理', () => {
  test.beforeEach(async ({ page }) => {
    // 先登录
    await page.goto('/login');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForSelector('.ant-input, input.ant-input', { timeout: 15000 });
    const inputs = page.locator('input.ant-input');
    await inputs.nth(0).fill('admin');
    await inputs.nth(1).fill('admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(dashboard|tickets|incidents)/, { timeout: 20000 });
  });

  test.describe('Release List - 发布列表', () => {
    test('should navigate to release management page', async ({ page }) => {
      await page.goto('/releases');
      await page.waitForURL(/\/releases/);
      await expect(page.locator('h1, h2').first()).toBeVisible();
    });

    test('should display release list page', async ({ page }) => {
      await page.goto('/releases');
      await page.waitForLoadState('networkidle');
      // 检查页面有内容
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Release Create - 创建发布', () => {
    test('should display create release form', async ({ page }) => {
      await page.goto('/releases/new');
      await page.waitForLoadState('networkidle');
      // 表单应该可见
      await expect(page.locator('form, .ant-form').first()).toBeVisible({ timeout: 10000 });
    });
  });
});
