import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

test.describe('Security - 浏览器侧安全验证', () => {
  test('RBAC should block non-admin from admin pages', async ({ page }) => {
    await loginAndReturn(page, 'employee', 'admin123');
    await page.goto('/admin/users');
    await page.waitForLoadState('networkidle');

    const forbidden = page.locator(
      'text=403, text=Forbidden, text=无权限, text=Access Denied, .ant-result-403, .ant-alert-error'
    );
    const blocked = !page.url().includes('/admin/users') || (await forbidden.count()) > 0;
    expect(blocked).toBeTruthy();
  });

  test('XSS payload should not execute when rendered in ticket detail', async ({ page }) => {
    await page.addInitScript(() => {
      (window as any).__xss_executed = 0;
      (window as any).__xss_mark = () => {
        (window as any).__xss_executed = 1;
      };
    });

    await loginAndReturn(page, 'end_user', 'admin123');
    await page.goto('/tickets/create');
    await page.waitForSelector('form, [data-testid="ticket-form"]', { timeout: 15000 });

    const title = `XSS Test ${Date.now()}`;
    const payload = `<img src=x onerror="window.__xss_mark && window.__xss_mark()">`;

    await page
      .locator('input[id*="title"], input[name*="title"], input[placeholder*="标题"]')
      .first()
      .fill(title);
    await page
      .locator('textarea[id*="description"], textarea[name*="description"], textarea[placeholder*="描述"]')
      .first()
      .fill(payload);

    const createResp = page
      .waitForResponse(
        resp => resp.url().includes('/api/v1/tickets') && resp.request().method() === 'POST',
        { timeout: 15000 }
      )
      .catch(() => null);

    await page
      .locator(
        'button[type="submit"], button:has-text("提交"), button:has-text("创建"), button:has-text("创建工单")'
      )
      .first()
      .click();
    await createResp;

    await page.waitForLoadState('networkidle');
    const xss = await page.evaluate(() => (window as any).__xss_executed);
    expect(xss).toBe(0);
  });

  test('Sensitive data should not be sent via URL query on login', async ({ page }) => {
    await page.goto('/login');
    await page.waitForSelector('input.ant-input', { timeout: 15000 });

    const inputs = page.locator('input.ant-input');
    await inputs.nth(0).fill('admin');
    await inputs.nth(1).fill('admin123');
    await page.click('button[type="submit"]');
    await page.waitForLoadState('domcontentloaded');

    expect(page.url()).not.toContain('admin123');
    expect(page.url()).not.toMatch(/password=/i);
  });
});
