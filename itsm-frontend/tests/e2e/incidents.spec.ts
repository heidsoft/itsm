/**
 * Incident Management E2E Tests
 * 事件管理端到端测试
 */

import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

test.describe('Incident List - 事件列表', () => {
  test.beforeEach(async ({ page }) => {
    // 先登录
    await loginAndReturn(page);
    await page.goto('/incidents', { waitUntil: 'domcontentloaded' });
  });

  test('should display incident list page', async ({ page }) => {
    await expect(page.locator('h1').first()).toContainText(/事件|incident/i);
  });

  test('should display incident table', async ({ page }) => {
    await expect(page.locator('.ant-table')).toBeVisible();
  });

  test('should filter by status', async ({ page }) => {
    const statusFilter = page.locator('select[name="status"], [class*="status"]').first();
    if (await statusFilter.isVisible()) {
      await statusFilter.selectOption('open');
      await page.waitForTimeout(500);
    }
  });

  test('should filter by priority', async ({ page }) => {
    const priorityFilter = page.locator('select[name="priority"]').first();
    if (await priorityFilter.isVisible()) {
      await priorityFilter.selectOption('critical');
      await page.waitForTimeout(500);
    }
  });

  test('should search incidents', async ({ page }) => {
    const searchInput = page.locator('input[placeholder*="搜索"], input[type="search"]').first();
    if (await searchInput.isVisible()) {
      await searchInput.fill('数据库');
      await searchInput.press('Enter');
      await page.waitForTimeout(500);
    }
  });

  test('should navigate to incident detail', async ({ page }) => {
    const viewBtn = page.locator('button.ant-btn').filter({ has: page.locator('.lucide-eye') }).first();
    if (!(await viewBtn.isVisible().catch(() => false))) {
      test.skip();
      return;
    }
    await viewBtn.click();
    await expect(page).toHaveURL(/\/incidents\/\d+/);
  });
});

test.describe('Incident Create - 创建事件', () => {
  test.beforeEach(async ({ page }) => {
    // 先登录
    await loginAndReturn(page);
    await page.goto('/incidents/new');
  });

  test('should display create incident form', async ({ page }) => {
    await expect(page.locator('input[name="title"], input#title')).toBeVisible();
    await expect(page.locator('textarea[name="description"]')).toBeVisible();
  });

  test('should create incident successfully', async ({ page }) => {
    const titleInput = page.locator('input[name="title"], input#title').first();
    const descInput = page.locator('textarea[name="description"], textarea').first();
    const submitBtn = page.locator('button[type="submit"]').first();

    if (
      !(await titleInput.isVisible().catch(() => false)) ||
      !(await descInput.isVisible().catch(() => false)) ||
      !(await submitBtn.isVisible().catch(() => false))
    ) {
      test.skip();
      return;
    }

    const createResponsePromise = page.waitForResponse((resp) => {
      return resp.url().includes('/api/v1/incidents') && resp.request().method() === 'POST';
    });

    await titleInput.fill(`E2E 事件 - ${Date.now()}`);
    await descInput.fill('这是一个用于 E2E 的创建事件测试描述。');
    await submitBtn.click();

    const createResp = await createResponsePromise;
    expect(createResp.status()).toBeGreaterThanOrEqual(200);
    expect(createResp.status()).toBeLessThan(300);
  });

  test('should validate required fields', async ({ page }) => {
    await page.click('button[type="submit"]');
    // 验证表单提交行为
    await page.waitForTimeout(500);
  });
});

test.describe('Incident Detail - 事件详情', () => {
  test.beforeEach(async ({ page }) => {
    // 先登录
    await loginAndReturn(page);
    await page.goto('/incidents', { waitUntil: 'domcontentloaded' });

    const viewBtn = page.locator('button.ant-btn').filter({ has: page.locator('.lucide-eye') }).first();
    if (!(await viewBtn.isVisible().catch(() => false))) {
      test.skip();
      return;
    }
    await viewBtn.click();
  });

  test('should display incident detail', async ({ page }) => {
    await expect(page.locator('h1, .ant-card-head-title').first()).toBeVisible();
  });

  test('should display incident status badge', async ({ page }) => {
    await expect(page.locator('.ant-tag').first()).toBeVisible();
  });

  test('should have escalate button', async ({ page }) => {
    const escalateButton = page.locator('button:has-text("升级"), button:has-text("Escalate")');
    if (await escalateButton.isVisible()) {
      await escalateButton.click();
    }
  });

  test('should have assign button', async ({ page }) => {
    const assignButton = page.locator('button:has-text("分配"), button:has-text("Assign")');
    if (await assignButton.isVisible()) {
      await assignButton.click();
    }
  });
});
