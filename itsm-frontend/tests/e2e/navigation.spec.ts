/**
 * Navigation E2E Tests
 */

import { test, expect } from '@playwright/test';

test.describe('Main Navigation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should have main navigation menu', async ({ page }) => {
    const nav = page.locator('nav, header, [role="navigation"]');
    await expect(nav.first()).toBeVisible();
  });

  test('should navigate to tickets page', async ({ page }) => {
    await page.goto('/');
    const ticketsLink = page.getByText(/工单|Tickets/i).first();
    if (await ticketsLink.isVisible()) {
      await ticketsLink.click();
      await expect(page).toHaveURL(/tickets/);
    }
  });

  test('should navigate to dashboard', async ({ page }) => {
    await page.goto('/tickets');
    const dashboardLink = page.getByText(/仪表盘|Dashboard/i).first();
    if (await dashboardLink.isVisible()) {
      await dashboardLink.click();
      await expect(page).toHaveURL('/');
    }
  });
});

test.describe('Page Loading', () => {
  test('should load pages without errors', async ({ page }) => {
    const errors: string[] = [];

    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });

    page.on('pageerror', (error) => {
      errors.push(error.message);
    });

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Filter out known non-critical errors
    const criticalErrors = errors.filter(
      (e) => !e.includes('favicon') && !e.includes('404')
    );

    expect(criticalErrors).toHaveLength(0);
  });

  test('should have proper page title', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle(/ITSM|仪表盘|Dashboard/i);
  });
});

test.describe('Responsive Design', () => {
  test('should display correctly on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    // Should not have horizontal scroll
    const bodyWidth = await page.locator('body').evaluate((el) => el.scrollWidth);
    const viewport = page.viewportSize();
    await expect(bodyWidth).toBeLessThanOrEqual(viewport?.width ?? 375);
  });

  test('should display correctly on tablet', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto('/');
    await expect(page.locator('nav')).toBeVisible();
  });

  test('should display correctly on desktop', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.goto('/');
    await expect(page.locator('nav')).toBeVisible();
  });
});
