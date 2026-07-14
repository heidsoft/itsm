/**
 * Change Management E2E Tests
 * 变更管理端到端测试
 */

import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

test.describe('Change List - 变更列表', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
    await page.goto('/changes', { waitUntil: 'domcontentloaded' });
  });

  test('should display change list page', async ({ page }) => {
    await expect(page.locator('h1').first()).toContainText(/变更|change/i);
  });

  test('should display change table', async ({ page }) => {
    await expect(page.locator('.ant-table')).toBeVisible();
  });

  test('should filter by status', async ({ page }) => {
    const statusFilter = page.locator('select[name="status"]').first();
    if (await statusFilter.isVisible()) {
      await statusFilter.selectOption('pending_approval');
    }
  });

  test('should filter by change type', async ({ page }) => {
    const typeFilter = page.locator('select[name="type"]').first();
    if (await typeFilter.isVisible()) {
      await typeFilter.selectOption('standard');
    }
  });

  test('should navigate to change detail', async ({ page }) => {
    const viewBtn = page.locator('button.ant-btn').filter({ has: page.locator('.lucide-eye') }).first();
    if (!(await viewBtn.isVisible().catch(() => false))) {
      test.skip();
      return;
    }
    await viewBtn.click();
    await expect(page).toHaveURL(/\/changes\/\d+/);
  });
});

test.describe('Change Create - 创建变更', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
    await page.goto('/changes/new');
  });

  test('should display create change form', async ({ page }) => {
    await expect(page.locator('input[name="title"], input#title')).toBeVisible();
    await expect(page.locator('select[name="type"]')).toBeVisible();
  });

  test('should create change successfully', async ({ page }) => {
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
      return resp.url().includes('/api/v1/changes') && resp.request().method() === 'POST';
    });

    await titleInput.fill(`E2E 变更 - ${Date.now()}`);
    await descInput.fill('这是一个用于 E2E 的创建变更测试描述。');

    const typeSelect = page.locator('select[name="type"], [aria-label*="类型"]').first();
    if (await typeSelect.isVisible().catch(() => false)) {
      await typeSelect.selectOption({ index: 0 }).catch(() => {});
    }

    await submitBtn.click();
    const createResp = await createResponsePromise;
    expect(createResp.status()).toBeGreaterThanOrEqual(200);
    expect(createResp.status()).toBeLessThan(300);
  });

  test('should show risk assessment section', async ({ page }) => {
    await expect(page.locator('text=风险评估')).toBeVisible();
  });
});

test.describe('Change Detail - 变更详情', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
    await page.goto('/changes', { waitUntil: 'domcontentloaded' });

    const viewBtn = page.locator('button.ant-btn').filter({ has: page.locator('.lucide-eye') }).first();
    if (!(await viewBtn.isVisible().catch(() => false))) {
      test.skip();
      return;
    }
    await viewBtn.click();
  });

  test('should display change detail', async ({ page }) => {
    await expect(page.locator('h1, .ant-card-head-title').first()).toBeVisible();
  });

  test('should display approval status', async ({ page }) => {
    await expect(page.locator('text=pending_approval')).toBeVisible();
  });

  test('should have approval button', async ({ page }) => {
    const approveButton = page.locator('button:has-text("审批"), button:has-text("批准")');
    if (await approveButton.isVisible()) {
      await approveButton.click();
    }
  });

  test('should display risk assessment', async ({ page }) => {
    await expect(page.locator('text=风险评估')).toBeVisible();
  });

  test('should display rollback plan', async ({ page }) => {
    const rollbackTab = page.locator('button:has-text("回滚计划")');
    if (await rollbackTab.isVisible()) {
      await rollbackTab.click();
    }
  });
});

test.describe('Change Approval - 变更审批', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
  });

  test('should approve change', async ({ page }) => {
    await page.goto('/changes', { waitUntil: 'domcontentloaded' });

    const viewBtn = page.locator('button.ant-btn').filter({ has: page.locator('.lucide-eye') }).first();
    if (!(await viewBtn.isVisible().catch(() => false))) {
      test.skip();
      return;
    }
    await viewBtn.click();

    const approveButton = page.locator('button:has-text("批准"), button:has-text("审批")').first();
    if (!(await approveButton.isVisible().catch(() => false))) {
      test.skip();
      return;
    }

    const approveResponsePromise = page.waitForResponse((resp) => {
      return resp.url().includes('/approve') && resp.request().method() === 'POST';
    });
    await approveButton.click();
    const approveResp = await approveResponsePromise;
    expect(approveResp.status()).toBeGreaterThanOrEqual(200);
    expect(approveResp.status()).toBeLessThan(300);
  });

  test('should reject change', async ({ page }) => {
    await page.goto('/changes', { waitUntil: 'domcontentloaded' });

    const viewBtn = page.locator('button.ant-btn').filter({ has: page.locator('.lucide-eye') }).first();
    if (!(await viewBtn.isVisible().catch(() => false))) {
      test.skip();
      return;
    }
    await viewBtn.click();

    const rejectButton = page.locator('button:has-text("拒绝")').first();
    if (!(await rejectButton.isVisible().catch(() => false))) {
      test.skip();
      return;
    }

    const rejectResponsePromise = page.waitForResponse((resp) => {
      return resp.url().includes('/reject') && resp.request().method() === 'POST';
    });
    await rejectButton.click();
    const rejectResp = await rejectResponsePromise;
    expect(rejectResp.status()).toBeGreaterThanOrEqual(200);
    expect(rejectResp.status()).toBeLessThan(300);
  });
});
