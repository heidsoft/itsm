/**
 * 模块访问测试 - 无需认证
 * 测试各模块页面是否正确重定向到登录页
 */

import { test, expect } from '@playwright/test';

const protectedRoutes = [
  '/releases',
  '/assets',
  '/licenses',
  '/sla-dashboard',
  '/workflow',
  '/workflow/designer',
  '/workflow/instances',
  '/workflow/dashboard',
  '/workflow/audit',
  '/workflow/sla',
];

test.describe('Module Access - 模块访问控制', () => {
  protectedRoutes.forEach((route) => {
    test(`should redirect ${route} to login when not authenticated`, async ({ page }) => {
      await page.goto(route);
      await expect(page).toHaveURL(/\/login/, { timeout: 15000 });
    });
  });
});
