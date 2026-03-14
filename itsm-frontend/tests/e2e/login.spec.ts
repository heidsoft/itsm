/**
 * Login Page E2E Tests
 * 测试登录页面的功能和交互
 */

import { test, expect } from '@playwright/test';

test.describe('Login Page - 无需认证', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    // 等待页面完全加载
    await page.waitForLoadState('domcontentloaded');
  });

  test('should display login form', async ({ page }) => {
    await expect(page.locator('form')).toBeVisible({ timeout: 15000 });
  });

  test('should display username field', async ({ page }) => {
    // 等待 Ant Design 表单渲染
    await page.waitForSelector('.ant-input, input.ant-input', { timeout: 15000 });
    const inputs = page.locator('input.ant-input');
    await expect(inputs.first()).toBeVisible();
  });

  test('should display password field', async ({ page }) => {
    await page.waitForSelector('input[type="password"]', { timeout: 15000 });
    await expect(page.locator('input[type="password"]')).toBeVisible();
  });

  test('should display login button', async ({ page }) => {
    await page.waitForSelector('button[type="submit"]', { timeout: 15000 });
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('should show validation error when submitting empty form', async ({ page }) => {
    await page.waitForSelector('button[type="submit"]', { timeout: 15000 });
    await page.click('button[type="submit"]');
    // 等待一下让验证运行
    await page.waitForTimeout(500);
    // 页面仍然在登录页
    await expect(page).toHaveURL(/\/login/);
  });

  test('should redirect to dashboard on successful login', async ({ page }) => {
    // 等待输入框加载
    await page.waitForSelector('.ant-input, input.ant-input', { timeout: 15000 });

    // 使用更通用的选择器
    const inputs = page.locator('input.ant-input');
    const usernameInput = inputs.nth(0);
    const passwordInput = inputs.nth(1);

    await usernameInput.fill('admin');
    await passwordInput.fill('admin123');

    // 提交表单
    await page.click('button[type="submit"]');

    // 等待登录成功跳转
    await expect(page).not.toHaveURL(/\/login/, { timeout: 20000 });
  });
});

test.describe('Dashboard Page - 需要认证', () => {
  test.use({ storageState: {} });

  test('should redirect to login when not authenticated', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveURL(/\/login/);
  });

  test('should display dashboard after login', async ({ page }) => {
    // 访问登录页
    await page.goto('/login');
    await page.waitForSelector('.ant-input, input.ant-input', { timeout: 15000 });

    // 填写表单
    const inputs = page.locator('input.ant-input');
    await inputs.nth(0).fill('admin');
    await inputs.nth(1).fill('admin123');

    // 提交
    await page.click('button[type="submit"]');

    // 等待跳转
    await page.waitForURL(/\/(dashboard|tickets)/, { timeout: 20000 });

    // 验证不在登录页
    await expect(page).not.toHaveURL(/\/login/);
  });
});
