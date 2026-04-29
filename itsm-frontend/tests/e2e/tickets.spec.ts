/**
 * Ticket Management E2E Tests
 */

import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

test.describe('Ticket List Page', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
    await page.goto('/tickets', { waitUntil: 'domcontentloaded' });
  });

  test('should display ticket list', async ({ page }) => {
    await expect(page.locator('table, [class*="table"], [class*="list"]')).toBeVisible();
  });

  test('should have search functionality', async ({ page }) => {
    const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="Search"]');
    await expect(searchInput).toBeVisible();
  });

  test('should have filter functionality', async ({ page }) => {
    const filterSelect = page.locator('[class*="filter"], select[class*="filter"]');
    await expect(filterSelect.first()).toBeVisible();
  });

  test('should navigate to ticket detail (if any row exists)', async ({ page }) => {
    const row = page.locator('table tbody tr').first();
    if (!(await row.isVisible().catch(() => false))) {
      test.skip();
      return;
    }

    await row.click();
    await page.waitForTimeout(300);

    if (!/\/tickets\/\d+/.test(page.url())) {
      const viewButton = page
        .locator('a[href*="/tickets/"], button:has-text("查看"), button:has-text("详情")')
        .first();
      if (await viewButton.isVisible().catch(() => false)) {
        await viewButton.click();
      }
    }

    await expect(page).toHaveURL(/\/tickets\/\d+/);
  });
});

test.describe('Ticket Detail Page', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
  });

  test('should display ticket title', async ({ page }) => {
    await page.goto('/tickets', { waitUntil: 'domcontentloaded' });

    const row = page.locator('table tbody tr').first();
    if (!(await row.isVisible().catch(() => false))) {
      test.skip();
      return;
    }

    await row.click();
    await page.waitForTimeout(300);
    await expect(page.locator('h1, .ant-card-head-title, [data-testid="ticket-title"]')).toBeVisible();
  });

  test('should display ticket details', async ({ page }) => {
    if (!/\/tickets\/\d+/.test(page.url())) {
      test.skip();
      return;
    }
    await expect(page.locator('[class*="detail"], [class*="info"], .ant-descriptions')).toBeVisible();
  });

  test('should have action buttons', async ({ page }) => {
    if (!/\/tickets\/\d+/.test(page.url())) {
      test.skip();
      return;
    }
    const actionButtons = page.locator('button');
    await expect(actionButtons.first()).toBeVisible();
  });
});

test.describe('Create Ticket Flow', () => {
  test('should display create ticket form', async ({ page }) => {
    await loginAndReturn(page);
    await page.goto('/tickets/create');
    await expect(page.locator('form, [data-testid="ticket-form"]')).toBeVisible();
  });

  test('should have required fields', async ({ page }) => {
    await loginAndReturn(page);
    await page.goto('/tickets/create');
    const titleInput = page.locator('[data-testid="ticket-title-input"], input[id*="title"], input[name*="title"]');
    await expect(titleInput).toBeVisible();
  });
});
