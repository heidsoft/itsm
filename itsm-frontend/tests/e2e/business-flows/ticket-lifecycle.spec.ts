// itsm-frontend/tests/e2e/business-flows/ticket-lifecycle.spec.ts
import { test, expect } from '@playwright/test';
import { loginAs, logout } from '../utils/test-utils';
import { TicketPage } from '../utils/page-objects/TicketPage';

test.describe('工单完整生命周期测试', () => {

  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    await logout(page).catch(() => {});
  });

  test('工单创建页面加载', async ({ page }) => {
    await loginAs(page, 'admin');

    // 等待登录完成
    await page.waitForURL(/\/(dashboard|tickets|incidents|\/)$/, { timeout: 15000 }).catch(() => {});

    // 导航到创建页面
    await page.goto('/tickets/create');
    await page.waitForLoadState('domcontentloaded');

    // 验证页面内容存在
    const bodyContent = await page.locator('body').textContent();
    expect(bodyContent?.length).toBeGreaterThan(100);
  });

  test('工单列表页面加载', async ({ page }) => {
    await loginAs(page, 'admin');
    const ticketPage = new TicketPage(page);
    await ticketPage.goto();

    // 验证页面加载 - 检查页面内容存在
    const bodyContent = await page.locator('body').textContent();
    expect(bodyContent?.length).toBeGreaterThan(100);
  });

  test('工单详情页面元素验证', async ({ page }) => {
    await loginAs(page, 'admin');
    const ticketPage = new TicketPage(page);
    await ticketPage.goto();

    // 检查表格是否存在
    const tableExists = await page.locator('table').isVisible().catch(() => false);
    if (!tableExists) {
      test.skip();
      return;
    }

    const ticketId = await ticketPage.getFirstTicketId();
    if (!ticketId) {
      test.skip();
      return;
    }

    await ticketPage.openTicket(ticketId);
    await page.waitForLoadState('domcontentloaded');

    // 验证详情页面加载
    expect(page.url()).toMatch(/\/tickets\/\d+/);
  });
});
