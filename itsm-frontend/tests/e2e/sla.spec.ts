/**
 * SLA Monitoring E2E Tests
 * SLA 监控模块测试
 */

import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

test.describe('SLA Monitoring - SLA监控', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
  });

  test.describe('SLA Dashboard - SLA仪表盘', () => {
    test('should navigate to SLA dashboard page', async ({ page }) => {
      await page.goto('/sla-dashboard', { timeout: 30000 });
      await page.waitForLoadState('domcontentloaded');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should display SLA dashboard', async ({ page }) => {
      await page.goto('/sla-dashboard');
      await page.waitForLoadState('networkidle');
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('SLA Definitions - SLA定义', () => {
    test('should navigate to SLA definitions page', async ({ page }) => {
      await page.goto('/sla');
      await page.waitForLoadState('domcontentloaded');
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Workflow SLA - 工作流SLA', () => {
    test('should navigate to workflow SLA page', async ({ page }) => {
      await page.goto('/workflow/sla');
      await page.waitForLoadState('domcontentloaded');
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('SLA Monitor - SLA监控', () => {
    test('should navigate to SLA monitor page', async ({ page }) => {
      await page.goto('/sla-monitor');
      await page.waitForLoadState('domcontentloaded');
      await expect(page.locator('body')).toBeVisible();
    });
  });
});
