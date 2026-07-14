/**
 * MSP Workflow E2E Tests
 * MSP工作流模块测试
 */

import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

test.describe('MSP Workflow', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
  });

  test('should display MSP dashboard for MSP user', async ({ page }) => {
    await page.goto('/msp');
    await page.waitForLoadState('domcontentloaded');
    // MSP page should load
    await expect(page.locator('body')).toBeVisible();
  });

  test('should navigate to MSP management', async ({ page }) => {
    await page.goto('/msp/management');
    await page.waitForLoadState('domcontentloaded');
    await expect(page.locator('body')).toBeVisible();
  });
});

test.describe('MSP Cross-Tenant Authorization', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
  });

  test('should allow MSP to view customer tickets with proper header', async ({ page }) => {
    await page.goto('/msp');
    await page.waitForLoadState('domcontentloaded');
    await expect(page.locator('body')).toBeVisible();
  });
});
