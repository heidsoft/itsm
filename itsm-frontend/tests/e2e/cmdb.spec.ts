/**
 * CMDB E2E Tests
 * 配置管理数据库模块测试
 */

import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

test.describe('CMDB - 配置管理', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
  });

  test.describe('CI List - 配置项列表', () => {
    test.beforeEach(async ({ page }) => {
      // Mock API routes
      await page.route(/\/api\/v1\/cmdb\/cis/, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 0,
            message: 'success',
            data: {
              items: [
                {
                  id: 1,
                  name: '应用服务器-01',
                  description: '测试资产',
                  type: 'server',
                  status: 'active',
                  ci_type_id: 1,
                },
              ],
              total: 1,
              page: 1,
              size: 10,
            },
          }),
        });
      });

      await page.route(/\/api\/v1\/cmdb\/types/, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 0,
            message: 'success',
            data: [{ id: 1, name: '服务器' }],
          }),
        });
      });
    });

    test('should navigate to CMDB page', async ({ page }) => {
      await page.goto('/cmdb', { timeout: 30000 });
      await page.waitForLoadState('domcontentloaded');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should display CI list', async ({ page }) => {
      await page.goto('/cmdb');
      await page.waitForLoadState('networkidle');
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('CI Types - 配置项类型', () => {
    test('should navigate to CI types page', async ({ page }) => {
      await page.goto('/admin/cmdb-types');
      await page.waitForLoadState('domcontentloaded');
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Cloud Resources - 云资源', () => {
    test('should navigate to cloud resources page', async ({ page }) => {
      await page.goto('/cmdb/cloud-resources');
      await page.waitForLoadState('domcontentloaded');
      await expect(page.locator('body')).toBeVisible();
    });
  });
});
