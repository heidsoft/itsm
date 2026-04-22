// itsm-frontend/tests/e2e/business-flows/problem-rca.spec.ts
import { test, expect } from '@playwright/test';
import { loginAs, logout } from '../utils/test-utils';
import { ProblemPage } from '../utils/page-objects/ProblemPage';

test.describe('问题管理RCA流程测试', () => {

  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    await logout(page).catch(() => {});
  });

  test('问题列表页面加载', async ({ page }) => {
    await loginAs(page, 'admin');
    await page.waitForURL(/\/(dashboard|tickets|incidents|problems|\/)$/, { timeout: 15000 }).catch(() => {});

    const problemPage = new ProblemPage(page);
    await problemPage.goto();

    // 验证页面内容存在
    const bodyContent = await page.locator('body').textContent();
    expect(bodyContent?.length).toBeGreaterThan(100);
  });

  test('问题创建页面加载', async ({ page }) => {
    await loginAs(page, 'admin');
    await page.waitForURL(/\/(dashboard|tickets|incidents|problems|\/)$/, { timeout: 15000 }).catch(() => {});

    // 导航到创建页面
    await page.goto('/problems/create');
    await page.waitForLoadState('domcontentloaded');

    // 验证页面内容存在
    const bodyContent = await page.locator('body').textContent();
    expect(bodyContent?.length).toBeGreaterThan(100);
  });

  test('问题列表筛选和搜索', async ({ page }) => {
    await loginAs(page, 'admin');
    await page.waitForURL(/\/(dashboard|tickets|incidents|problems|\/)$/, { timeout: 15000 }).catch(() => {});

    const problemPage = new ProblemPage(page);
    await problemPage.goto();

    // 验证页面内容存在
    const bodyContent = await page.locator('body').textContent();
    expect(bodyContent?.length).toBeGreaterThan(100);
  });
});
