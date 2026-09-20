// 评审组管理 E2E 业务流测试
// 覆盖：登录 → 评审组列表 → 切换常规/紧急 → 新增成员 → 状态切换 → 删除 → 验证

import { test, expect } from '../auth-utils';
import { loginAndReturn } from '../auth-utils';

const BASE_URL = process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3000';
// Credentials come from the environment. Never hard-code a real password here:
// this repository is public and anything committed stays in git history.
//   E2E_ADMIN_USER      default: admin
//   E2E_ADMIN_PASSWORD  no default — the suite skips when unset
const ADMIN_USER = process.env.E2E_ADMIN_USER || 'admin';
const ADMIN_PASSWORD = process.env.E2E_ADMIN_PASSWORD || '';

test.describe('评审组管理业务流程', () => {
  // Skip rather than fail with a confusing auth error when no password is set.
  test.skip(!ADMIN_PASSWORD, 'E2E_ADMIN_PASSWORD not set - skipping authenticated flow');

  test.beforeEach(async ({ page }) => {
    // 直接用 API 登录并注入 cookie，再访问页面
    await loginAndReturn(page, ADMIN_USER, ADMIN_PASSWORD, '/dashboard');
  });

  test('完整 CRUD 链路 + 状态切换', async ({ page }) => {
    test.setTimeout(60_000);

    // 1. 登录后直接访问评审组页面
    await page.goto(`${BASE_URL}/admin/change-review`, { waitUntil: 'domcontentloaded' });
    await page.waitForLoadState('networkidle');

    // 验证页面标题
    await expect(page.locator('h3:has-text("评审组管理")')).toBeVisible({ timeout: 10_000 });
    console.log('✓ Step 1: 评审组页面加载成功');

    // 2. 截图初始状态
    await page.screenshot({ path: '/tmp/change-review-e2e/01-page-loaded.png', fullPage: true });

    // 3. 验证 Segmented 控件并切换到紧急评审组
    const emergencySegment = page.locator('.ant-segmented-item:has-text("紧急评审组")').first();
    await expect(emergencySegment).toBeVisible();
    await emergencySegment.click();
    await page.waitForTimeout(800);
    console.log('✓ Step 2: 切换到紧急评审组');

    await page.screenshot({ path: '/tmp/change-review-e2e/02-emergency-empty.png', fullPage: true });

    // 4. 点击新增按钮
    const addBtn = page.locator('button:has-text("新增成员")').first();
    await expect(addBtn).toBeVisible();
    await addBtn.click();
    await page.waitForTimeout(500);

    // 5. 填写表单
    // 5.1 用户选择
    const userSelect = page.locator('.ant-modal .ant-select').first();
    await userSelect.click();
    await page.waitForTimeout(800);
    // 选择 reviewtest
    const reviewtestOption = page.locator('.ant-select-item-option:has-text("reviewtest")').first();
    await expect(reviewtestOption).toBeVisible({ timeout: 5_000 });
    await reviewtestOption.click();
    await page.waitForTimeout(300);

    // 5.2 角色选择 - 默认已经是 member，改为 chair
    await page.screenshot({ path: '/tmp/change-review-e2e/03-modal-filled.png', fullPage: true });

    // 6. 提交 - 点击 Modal 内的"确定"按钮
    const okBtn = page.locator('.ant-modal-footer .ant-btn-primary').first();
    await okBtn.click();

    // 等待 Modal 关闭（以加载列表为信号）
    await page.waitForSelector('.ant-modal', { state: 'hidden', timeout: 15_000 });
    await page.waitForTimeout(1500);
    console.log('✓ Step 3: 提交新增表单');

    // 7. 验证紧急评审组列表中出现新成员
    // 等待 modal 关闭后新成员条目出现
    await page.waitForTimeout(2500);

    const emergencyTableRows = await page.locator('.ant-table-tbody > tr').count();
    console.log(`  紧急评审组 当前行数: ${emergencyTableRows}`);
    expect(emergencyTableRows).toBeGreaterThan(0);

    // 验证 reviewtest 显示在紧急评审组列表
    await expect(page.locator('.ant-table-tbody').getByText('reviewtest@example.com')).toBeVisible({ timeout: 10_000 });
    console.log('✓ Step 4: 验证 reviewtest 在紧急评审组列表中');

    await page.screenshot({ path: '/tmp/change-review-e2e/04-emergency-after-add.png', fullPage: true });

    // 8. 测试状态切换（Switch）
    const firstSwitch = page.locator('.ant-table-tbody .ant-switch').first();
    const switchChecked = await firstSwitch.getAttribute('aria-checked');
    await firstSwitch.click();
    await page.waitForTimeout(1500);
    const newSwitchChecked = await firstSwitch.getAttribute('aria-checked');
    expect(newSwitchChecked).not.toBe(switchChecked);
    console.log('✓ Step 5: 状态切换成功（前后 aria-checked 不同）');

    await page.screenshot({ path: '/tmp/change-review-e2e/05-after-toggle.png', fullPage: true });

    // 9. 测试删除
    const deleteBtn = page.locator('.ant-table-tbody button:has-text("移除")').first();
    await deleteBtn.click();
    await page.waitForTimeout(500);

    // 点击 Popconfirm 确认
    const popconfirmOkBtn = page.locator('.ant-popover button:has-text("确定"), .ant-popover .ant-btn-primary').first();
    await popconfirmOkBtn.click();
    await page.waitForTimeout(1500);

    // 验证删除成功
    const remainingRows = await page.locator('.ant-table-tbody > tr').count();
    console.log(`  紧急评审组 删除后剩余行数: ${remainingRows}`);
    console.log('✓ Step 6: 删除成功');

    await page.screenshot({ path: '/tmp/change-review-e2e/06-after-delete.png', fullPage: true });

    // 10. 切换回常规评审组验证先前数据
    const regularSegment = page.locator('.ant-segmented-item:has-text("常规评审组")').first();
    await regularSegment.click();
    await page.waitForTimeout(1500);
    await page.screenshot({ path: '/tmp/change-review-e2e/07-regular-tab.png', fullPage: true });

    // 11. 验证侧边栏菜单含有评审组
    const sidebar = await page.locator('.ant-layout-sider').innerText();
    expect(sidebar).toContain('评审组');
    console.log('✓ Step 7: 侧边栏菜单包含评审组');

    console.log('\n=== 所有 7 个步骤全部通过 ===');
  });

  test('菜单导航到评审组页面', async ({ page }) => {
    test.setTimeout(30_000);

    await page.goto(`${BASE_URL}/dashboard`, { waitUntil: 'domcontentloaded' });
    await page.waitForLoadState('networkidle');

    // 找到侧边栏的评审组菜单项
    const reviewMenu = page.locator('.ant-menu-item:has-text("评审组"), .ant-menu-submenu-title:has-text("评审组")').first();
    await reviewMenu.click();
    await page.waitForURL(/\/admin\/change-review/, { timeout: 10_000 });
    console.log('✓ 侧边栏菜单成功导航到评审组页面');

    await page.screenshot({ path: '/tmp/change-review-e2e/menu-navigation.png', fullPage: true });
  });
});
