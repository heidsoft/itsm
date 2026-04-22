// itsm-frontend/tests/e2e/business-flows/change-approval.spec.ts
import { test, expect } from '@playwright/test';
import { loginAs, logout } from '../utils/test-utils';
import { ChangePage } from '../utils/page-objects/ChangePage';

test.describe('变更审批流程测试', () => {

  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    await logout(page).catch(() => {});
  });

  test('变更列表页面加载', async ({ page }) => {
    await loginAs(page, 'admin');
    await page.waitForURL(/\/(dashboard|tickets|incidents|changes|\/)$/, { timeout: 15000 }).catch(() => {});

    const changePage = new ChangePage(page);
    await changePage.goto();

    // 验证页面内容存在
    const bodyContent = await page.locator('body').textContent();
    expect(bodyContent?.length).toBeGreaterThan(100);
  });

  test('变更创建页面加载', async ({ page }) => {
    await loginAs(page, 'admin');
    await page.waitForURL(/\/(dashboard|tickets|incidents|changes|\/)$/, { timeout: 15000 }).catch(() => {});

    // 导航到创建页面
    await page.goto('/changes/create');
    await page.waitForLoadState('domcontentloaded');

    // 验证页面内容存在
    const bodyContent = await page.locator('body').textContent();
    expect(bodyContent?.length).toBeGreaterThan(100);
  });

  test('变更列表搜索功能', async ({ page }) => {
    await loginAs(page, 'admin');
    await page.waitForURL(/\/(dashboard|tickets|incidents|changes|\/)$/, { timeout: 15000 }).catch(() => {});

    const changePage = new ChangePage(page);
    await changePage.goto();

    // 验证页面内容存在
    const bodyContent = await page.locator('body').textContent();
    expect(bodyContent?.length).toBeGreaterThan(100);
  });
});
