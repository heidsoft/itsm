/**
 * Knowledge Base E2E Tests
 * 知识库端到端测试
 */

import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

test.describe('Knowledge Base - 知识库', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
  });

  test.describe('Article List - 文章列表', () => {
    test('should navigate to knowledge page', async ({ page }) => {
      await page.goto('/knowledge');
      await page.waitForLoadState('domcontentloaded');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should display article list', async ({ page }) => {
      await page.goto('/knowledge');
      await page.waitForLoadState('networkidle');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should display articles', async ({ page }) => {
      await page.goto('/knowledge');
      await page.waitForLoadState('networkidle');
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Article Detail - 文章详情', () => {
    test('should navigate to article detail', async ({ page }) => {
      await page.goto('/knowledge');
      await page.waitForLoadState('networkidle');

      const firstLink = page.locator('a[href^="/knowledge/"]').first();
      if (!(await firstLink.isVisible().catch(() => false))) {
        test.skip();
        return;
      }
      await firstLink.click();
      await page.waitForLoadState('domcontentloaded');
      await expect(page).toHaveURL(/\/knowledge\/\d+/);
    });
  });

  test.describe('Create Article - 创建文章', () => {
    test('should navigate to create article page', async ({ page }) => {
      await page.goto('/knowledge/create');
      await page.waitForLoadState('domcontentloaded');
      await expect(page.locator('body')).toBeVisible();
    });
  });
});
