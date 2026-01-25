import { test, expect } from '@playwright/test';

test.describe('Integration Tests', () => {
  test('login flow and navigation', async ({ page }) => {
    // 1. Go to login page
    await page.goto('/login');
    
    // Check if we are on login page
    await expect(page.getByRole('heading', { name: /登录/ })).toBeVisible();

    // 2. Fill credentials
    await page.getByPlaceholder('用户名').fill('user1');
    await page.getByPlaceholder('密码').fill('user123');
    
    // 3. Click login
    await page.getByRole('button', { name: '登录' }).click();

    // 4. Wait for navigation to dashboard
    // Depending on the app, it might redirect to / or /dashboard
    await expect(page).toHaveURL(/\/dashboard|^\/$/);
    
    // 5. Verify dashboard content
    // Check for some dashboard elements like "概览" or "工单"
    await expect(page.getByText('工作台概览')).toBeVisible();

    // 6. Navigate to Tickets page
    await page.goto('/tickets');
    await expect(page).toHaveURL(/\/tickets/);
    await expect(page.getByText('工单列表')).toBeVisible();

    // 7. Navigate to Service Catalog
    await page.goto('/service-catalog');
    await expect(page).toHaveURL(/\/service-catalog/);
    // There might be "服务目录" or similar text
    // await expect(page.getByText('服务目录')).toBeVisible();

    // 8. Navigate to Knowledge Base
    await page.goto('/knowledge');
    await expect(page).toHaveURL(/\/knowledge/);
    await expect(page.getByText('文章列表')).toBeVisible();
    
    // 9. Navigate to Incidents
    await page.goto('/incidents');
    await expect(page).toHaveURL(/\/incidents/);
  });
});
