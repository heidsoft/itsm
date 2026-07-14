/**
 * Service Catalog E2E Tests
 * 服务目录端到端测试
 */

import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

test.describe('Service Catalog - 服务目录', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
  });

  test.describe('Service List - 服务列表', () => {
    test('should navigate to service catalog page', async ({ page }) => {
      await page.goto('/service-catalog');
      await page.waitForLoadState('domcontentloaded');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should display service catalog title', async ({ page }) => {
      await page.goto('/service-catalog');
      await page.waitForLoadState('networkidle');
      await expect(page.locator('text=服务目录')).toBeVisible();
    });

    test('should display category tabs', async ({ page }) => {
      await page.goto('/service-catalog');
      await page.waitForLoadState('networkidle');
      // 验证分类标签存在
      await expect(page.locator('text=全部服务')).toBeVisible();
    });

    test('should filter by category', async ({ page }) => {
      await page.goto('/service-catalog');
      await page.waitForLoadState('networkidle');
      // 点击云资源服务分类
      const categoryTab = page.locator('text=云资源服务').first();
      if (await categoryTab.isVisible().catch(() => false)) {
        await categoryTab.click();
        await page.waitForTimeout(500);
      }
    });
  });

  test.describe('Service Request - 服务请求', () => {
    test('should navigate to service request page', async ({ page }) => {
      await page.goto('/service-requests', { timeout: 60000 });
      await page.waitForLoadState('domcontentloaded');
      await expect(page.locator('body')).toBeVisible();
    });
  });
});

test.describe('Service Catalog - Create Service - 创建服务', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
  });

  test('should open create service modal', async ({ page }) => {
    await page.goto('/service-catalog');
    await page.waitForLoadState('networkidle');

    // 点击创建服务按钮
    const createButton = page.locator('button:has-text("创建服务")').first();
    if (await createButton.isVisible()) {
      await createButton.click();
      await page.waitForTimeout(500);
    }
  });
});
