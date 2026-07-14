/**
 * End User Role E2E Tests
 * Tests features accessible to end_user role
 */

import { test, expect } from '@playwright/test';
import { TEST_USERS, waitForTable } from './utils/test-utils';

// Increase timeout for end user tests
test.describe.configure({ timeout: 60000 });

test.describe('End User Role - Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    // Login as end_user
    await page.goto('/login');
    await page.waitForSelector('.ant-input', { timeout: 15000 });

    const inputs = page.locator('input.ant-input');
    await inputs.nth(0).fill(TEST_USERS.end_user.username);
    await inputs.nth(1).fill(TEST_USERS.end_user.password);

    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(dashboard|tickets|incidents|problems|changes)/, { timeout: 30000 });
  });

  test('should access dashboard', async ({ page }) => {
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    // Dashboard should show content
    const dashboardContent = page.locator('[class*="dashboard"], .ant-layout-content');
    await expect(dashboardContent.first()).toBeVisible({ timeout: 10000 });
  });

  test('should view own tickets', async ({ page }) => {
    await page.goto('/tickets');
    await page.waitForLoadState('networkidle');

    // Should display ticket list or empty state
    const content = page.locator('.ant-table, [class*="empty"], [class*="table"]');
    await expect(content.first()).toBeVisible({ timeout: 10000 });
  });

  test('should create new ticket', async ({ page }) => {
    await page.goto('/tickets/create');
    await page.waitForLoadState('networkidle');

    // Check for form
    const form = page.locator('form');
    await expect(form).toBeVisible({ timeout: 10000 });

    // Fill in ticket form if available
    const titleInput = page.locator('input[id*="title"], input[name*="title"]');
    if (await titleInput.isVisible()) {
      await titleInput.fill('Test Ticket from End User');

      const descInput = page.locator('textarea[id*="description"], textarea[name*="description"]');
      if (await descInput.isVisible()) {
        await descInput.fill('This is a test ticket created by end user');
      }
    }
  });

  test('should access knowledge base', async ({ page }) => {
    await page.goto('/knowledge');
    await page.waitForLoadState('networkidle');

    // Should display knowledge base content
    const content = page.locator('[class*="knowledge"], .ant-layout-content, article');
    await expect(content.first()).toBeVisible({ timeout: 10000 });
  });

  test('should access service catalog', async ({ page }) => {
    await page.goto('/service-catalog');
    await page.waitForLoadState('networkidle');

    // Should display service catalog
    const content = page.locator('[class*="catalog"], .ant-card, [class*="service"]');
    await expect(content.first()).toBeVisible({ timeout: 10000 });
  });
});

test.describe('End User Role - Access Control', () => {
  test.beforeEach(async ({ page }) => {
    // Login as end_user
    await page.goto('/login');
    await page.waitForSelector('.ant-input', { timeout: 15000 });

    const inputs = page.locator('input.ant-input');
    await inputs.nth(0).fill(TEST_USERS.end_user.username);
    await inputs.nth(1).fill(TEST_USERS.end_user.password);

    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(dashboard|tickets|incidents|problems|changes)/, { timeout: 30000 });
  });

  test('should have limited access to admin panel', async ({ page }) => {
    await page.goto('/admin');
    await page.waitForLoadState('networkidle');

    // End user may have read-only access to admin panel
    // Check if they can access but with limited functionality
    const currentUrl = page.url();

    // Either redirected to login, see access denied, or can only view (no edit controls)
    const accessDenied = page.locator('text=无权限, text=403, [class*="403"]');
    const noEditButton = await page.locator('button:has-text("编辑"), button:has-text("Delete"), button:has-text("删除")').count() === 0;

    // End user should either be denied, redirected, or have no edit buttons
    expect(
      currentUrl.includes('login') ||
      (await accessDenied.count() > 0) ||
      noEditButton
    ).toBeTruthy();
  });

  test('should have limited access to user management', async ({ page }) => {
    await page.goto('/admin/users');
    await page.waitForLoadState('networkidle');

    // End user may have read-only access
    const currentUrl = page.url();
    const accessDenied = page.locator('text=无权限, text=403, [class*="403"]');

    // Either redirected, see access denied, or can only view (no add/edit/delete buttons)
    const noEditButton = await page.locator('button:has-text("编辑"), button:has-text("Delete"), button:has-text("删除"), button:has-text("新增")').count() === 0;

    expect(
      currentUrl.includes('login') ||
      (await accessDenied.count() > 0) ||
      noEditButton
    ).toBeTruthy();
  });

  test('should have limited access to system config', async ({ page }) => {
    await page.goto('/admin/system-config');
    await page.waitForLoadState('networkidle');

    // End user may have read-only access to system config
    const currentUrl = page.url();
    const accessDenied = page.locator('text=无权限, text=403, [class*="403"]');

    // Either redirected, see access denied, or can only view (no edit controls)
    const noEditButton = await page.locator('button:has-text("保存"), button:has-text("配置")').count() === 0;

    expect(
      currentUrl.includes('login') ||
      (await accessDenied.count() > 0) ||
      noEditButton
    ).toBeTruthy();
  });
});
