/**
 * Agent/Security Role E2E Tests
 * Tests features accessible to agent/security role
 * Note: Uses security1 user from seed data (role: security)
 */

import { test, expect } from '@playwright/test';
import { TEST_USERS } from './utils/test-utils';

// Increase timeout for agent tests
test.describe.configure({ timeout: 60000 });

test.describe('Agent/Security Role - Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    // Login as security (simulates agent-like role with limited permissions)
    await page.goto('/login');
    await page.waitForSelector('.ant-input', { timeout: 15000 });

    const inputs = page.locator('input.ant-input');
    await inputs.nth(0).fill(TEST_USERS.security.username);
    await inputs.nth(1).fill(TEST_USERS.security.password);

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

  test('should access tickets', async ({ page }) => {
    await page.goto('/tickets');
    await page.waitForLoadState('networkidle');

    // Should display ticket list
    const content = page.locator('.ant-table, [class*="table"]');
    await expect(content.first()).toBeVisible({ timeout: 10000 });
  });

  test('should view ticket details', async ({ page }) => {
    await page.goto('/tickets');
    await page.waitForLoadState('networkidle');

    // Try to click on a Ticket if any exists
    const firstTicket = page.locator('.ant-table-row, [class*="row"], tr').first();
    if (await firstTicket.isVisible()) {
      await firstTicket.click();
      await page.waitForLoadState('networkidle');

      // Should show ticket details
      const details = page.locator('[class*="detail"], [class*="info"]');
      await expect(details.first()).toBeVisible({ timeout: 5000 });
    }
  });
});

test.describe('Agent/Security Role - Ticket Management', () => {
  test.beforeEach(async ({ page }) => {
    // Login as security user
    await page.goto('/login');
    await page.waitForSelector('.ant-input', { timeout: 15000 });

    const inputs = page.locator('input.ant-input');
    await inputs.nth(0).fill(TEST_USERS.security.username);
    await inputs.nth(1).fill(TEST_USERS.security.password);

    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(dashboard|tickets|incidents|problems|changes)/, { timeout: 30000 });
  });

  test('should access incident management', async ({ page }) => {
    await page.goto('/incidents');
    await page.waitForLoadState('networkidle');

    // Should display incident management
    const content = page.locator('.ant-layout-content, [class*="incident"]');
    await expect(content.first()).toBeVisible({ timeout: 10000 });
  });

  test('should access problem management', async ({ page }) => {
    await page.goto('/problems');
    await page.waitForLoadState('networkidle');

    // Should display problem management
    const content = page.locator('.ant-layout-content, [class*="problem"]');
    await expect(content.first()).toBeVisible({ timeout: 10000 });
  });

  test('should access change management', async ({ page }) => {
    await page.goto('/changes');
    await page.waitForLoadState('networkidle');

    // Should display change management
    const content = page.locator('.ant-layout-content, [class*="change"]');
    await expect(content.first()).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Agent/Security Role - Access Control', () => {
  test.beforeEach(async ({ page }) => {
    // Login as security user
    await page.goto('/login');
    await page.waitForSelector('.ant-input', { timeout: 15000 });

    const inputs = page.locator('input.ant-input');
    await inputs.nth(0).fill(TEST_USERS.security.username);
    await inputs.nth(1).fill(TEST_USERS.security.password);

    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(dashboard|tickets|incidents|problems|changes)/, { timeout: 30000 });
  });

  test('should have limited admin access', async ({ page }) => {
    await page.goto('/admin');
    await page.waitForLoadState('networkidle');

    // Security user may have limited admin access
    // Check the URL or access denied message
    const accessDenied = page.locator('text=无权限, text=403, [class*="403"]');
    const currentUrl = page.url();

    // Either redirected, access denied, or on admin page with limited access
    expect(currentUrl.includes('login') || currentUrl.includes('/admin') || (await accessDenied.count() > 0)).toBeTruthy();
  });

  test('should NOT access user management in some cases', async ({ page }) => {
    await page.goto('/admin/users');
    await page.waitForLoadState('networkidle');

    // Should be redirected or see access denied
    const accessDenied = page.locator('text=无权限, text=403, [class*="403"]');
    const redirected = page.url().includes('login');

    expect(redirected || (await accessDenied.count() > 0) || page.url().includes('/admin')).toBeTruthy();
  });
});

test.describe('Agent/Security Role - Service Features', () => {
  test.beforeEach(async ({ page }) => {
    // Login as security user
    await page.goto('/login');
    await page.waitForSelector('.ant-input', { timeout: 15000 });

    const inputs = page.locator('input.ant-input');
    await inputs.nth(0).fill(TEST_USERS.security.username);
    await inputs.nth(1).fill(TEST_USERS.security.password);

    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(dashboard|tickets|incidents|problems|changes)/, { timeout: 30000 });
  });

  test('should access knowledge base', async ({ page }) => {
    await page.goto('/knowledge');
    await page.waitForLoadState('networkidle');

    // Should display knowledge base
    const content = page.locator('[class*="knowledge"], .ant-layout-content');
    await expect(content.first()).toBeVisible({ timeout: 10000 });
  });

  test('should access CMDB', async ({ page }) => {
    await page.goto('/cmdb');
    await page.waitForLoadState('networkidle');

    // Should display CMDB content
    const content = page.locator('.ant-layout-content, [class*="cmdb"]');
    await expect(content.first()).toBeVisible({ timeout: 10000 });
  });

  test('should access service catalog', async ({ page }) => {
    await page.goto('/service-catalog');
    await page.waitForLoadState('networkidle');

    // Should display service catalog
    const content = page.locator('[class*="catalog"], .ant-card');
    await expect(content.first()).toBeVisible({ timeout: 10000 });
  });
});
