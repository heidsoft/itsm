import { test, expect } from '@playwright/test';

test.describe('ITSM 工单创建流程测试', () => {
  test.beforeEach(async ({ page }) => {
    // 访问登录页面
    await page.goto('http://localhost:3000');
    await page.waitForLoadState('networkidle');
  });

  test('登录系统', async ({ page }) => {
    // 填写登录表单
    await page.fill('input[name="username"], input[id="username"], input[placeholder*="用户名"]', 'admin');
    await page.fill('input[name="password"], input[id="password"], input[placeholder*="密码"]', 'admin123');

    // 点击登录按钮
    await page.click('button[type="submit"], button:has-text("登录")');

    // 等待页面跳转
    await page.waitForURL('**/dashboard**', { timeout: 10000 }).catch(() => {
      // 如果没有跳转到 dashboard，检查是否登录成功
    });

    // 验证登录成功（检查页面是否有用户信息或退出按钮）
    const isLoggedIn = await page.locator('text=退出, text=Logout, [aria-label="user"]').count() > 0;
    expect(isLoggedIn).toBeTruthy();
  });

  test('创建 incident 类型工单', async ({ page }) => {
    // 先登录
    await page.fill('input[name="username"], input[id="username"], input[placeholder*="用户名"]', 'admin');
    await page.fill('input[name="password"], input[id="password"], input[placeholder*="密码"]', 'admin123');
    await page.click('button[type="submit"], button:has-text("登录")');
    await page.waitForTimeout(2000);

    // 导航到工单页面
    await page.goto('http://localhost:3000/tickets');
    await page.waitForLoadState('networkidle');

    // 点击创建工单按钮
    await page.click('button:has-text("创建"), button:has-text("新建"), button:has-text("Create")');
    await page.waitForTimeout(1000);

    // 填写工单信息
    await page.fill('input[name="title"], input[id="title"]', 'E2E测试-服务器宕机');
    await page.fill('textarea[name="description"], textarea[id="description"]', 'E2E自动化测试创建的事件工单');

    // 选择工单类型
    const typeSelect = page.locator('select[name="type"], #type, [data-testid="ticket-type"]');
    if (await typeSelect.count() > 0) {
      await typeSelect.selectOption('incident');
    }

    // 选择优先级
    const prioritySelect = page.locator('select[name="priority"], #priority, [data-testid="ticket-priority"]');
    if (await prioritySelect.count() > 0) {
      await prioritySelect.selectOption('urgent');
    }

    // 提交工单
    await page.click('button[type="submit"], button:has-text("提交"), button:has-text("创建")');

    // 验证创建成功
    await page.waitForTimeout(2000);
    const successMessage = await page.locator('text=成功, text=Success, text=创建成功').count() > 0;
    expect(successMessage).toBeTruthy();
  });
});
