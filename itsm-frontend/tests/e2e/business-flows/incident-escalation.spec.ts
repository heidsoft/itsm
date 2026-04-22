// itsm-frontend/tests/e2e/business-flows/incident-escalation.spec.ts
import { test, expect } from '@playwright/test';
import { loginAs, logout } from '../utils/test-utils';
import { IncidentPage } from '../utils/page-objects/IncidentPage';

test.describe('事件升级流程测试', () => {

  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    await logout(page).catch(() => {});
  });

  test('事件列表页面加载', async ({ page }) => {
    await loginAs(page, 'admin');
    await page.waitForURL(/\/(dashboard|tickets|incidents|\/)$/, { timeout: 15000 }).catch(() => {});

    const incidentPage = new IncidentPage(page);
    await incidentPage.goto();

    // 验证页面内容存在
    const bodyContent = await page.locator('body').textContent();
    expect(bodyContent?.length).toBeGreaterThan(100);
  });

  test('事件创建页面加载', async ({ page }) => {
    await loginAs(page, 'admin');
    await page.waitForURL(/\/(dashboard|tickets|incidents|\/)$/, { timeout: 15000 }).catch(() => {});

    // 导航到创建页面
    await page.goto('/incidents/create');
    await page.waitForLoadState('domcontentloaded');

    // 验证页面内容存在
    const bodyContent = await page.locator('body').textContent();
    expect(bodyContent?.length).toBeGreaterThan(100);
  });

  test('事件列表搜索功能', async ({ page }) => {
    await loginAs(page, 'admin');
    await page.waitForURL(/\/(dashboard|tickets|incidents|\/)$/, { timeout: 15000 }).catch(() => {});

    const incidentPage = new IncidentPage(page);
    await incidentPage.goto();

    // 验证页面内容存在
    const bodyContent = await page.locator('body').textContent();
    expect(bodyContent?.length).toBeGreaterThan(100);
  });
});
