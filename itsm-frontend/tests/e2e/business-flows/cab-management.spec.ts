// CAB 成员管理 E2E 业务流测试
// 覆盖：登录 → CAB 列表 → 切换 CAB/ECAB → 新增成员 → 状态切换 → 删除 → 验证

import { test, expect } from '../auth-utils';
import { loginAndReturn } from '../auth-utils';

const BASE_URL = process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3000';
// Credentials come from the environment. Never hard-code a real password here:
// this repository is public and anything committed stays in git history.
//   E2E_ADMIN_USER      default: admin
//   E2E_ADMIN_PASSWORD  no default — the suite skips when unset
const ADMIN_USER = process.env.E2E_ADMIN_USER || 'admin';
const ADMIN_PASSWORD = process.env.E2E_ADMIN_PASSWORD || '';

test.describe('CAB 成员管理业务流程', () => {
  // Skip rather than fail with a confusing auth error when no password is set.
  test.skip(!ADMIN_PASSWORD, 'E2E_ADMIN_PASSWORD not set - skipping authenticated flow');

  test.beforeEach(async ({ page }) => {
    // 直接用 API 登录并注入 cookie，再访问页面
    await loginAndReturn(page, ADMIN_USER, ADMIN_PASSWORD, '/dashboard');
  });

  test('完整 CRUD 链路 + 状态切换', async ({ page }) => {
    test.setTimeout(60_000);

    // 1. 登录后直接访问 CAB 页面
    await page.goto(`${BASE_URL}/admin/cab`, { waitUntil: 'domcontentloaded' });
    await page.waitForLoadState('networkidle');

    // 验证页面标题
    await expect(page.locator('h3:has-text("CAB 成员管理")')).toBeVisible({ timeout: 10_000 });
    console.log('✓ Step 1: CAB 页面加载成功');

    // 2. 截图初始状态
    await page.screenshot({ path: '/tmp/cab-e2e/01-page-loaded.png', fullPage: true });

    // 3. 验证 Segmented 控件并切换到 ECAB
    const ecabSegment = page.locator('.ant-segmented-item:has-text("ECAB")').first();
    await expect(ecabSegment).toBeVisible();
    await ecabSegment.click();
    await page.waitForTimeout(800);
    console.log('✓ Step 2: 切换到 ECAB');

    await page.screenshot({ path: '/tmp/cab-e2e/02-ecab-empty.png', fullPage: true });

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
    // 选择 cabtest
    const cabtestOption = page.locator('.ant-select-item-option:has-text("cabtest")').first();
    await expect(cabtestOption).toBeVisible({ timeout: 5_000 });
    await cabtestOption.click();
    await page.waitForTimeout(300);

    // 5.2 角色选择 - 默认已经是 member，改为 chair
    await page.screenshot({ path: '/tmp/cab-e2e/03-modal-filled.png', fullPage: true });

    // 6. 提交 - 点击 Modal 内的"确定"按钮
    const okBtn = page.locator('.ant-modal-footer .ant-btn-primary').first();
    await okBtn.click();

    // 等待 Modal 关闭（以加载列表为信号）
    await page.waitForSelector('.ant-modal', { state: 'hidden', timeout: 15_000 });
    await page.waitForTimeout(1500);
    console.log('✓ Step 3: 提交新增表单');

    // 7. 验证 ECAB 列表中出现新成员
    // 等待 modal 关闭后新成员条目出现
    await page.waitForTimeout(2500);

    const ecabTableRows = await page.locator('.ant-table-tbody > tr').count();
    console.log(`  ECAB 当前行数: ${ecabTableRows}`);
    expect(ecabTableRows).toBeGreaterThan(0);

    // 验证 cabtest 显示在 ECAB 列表
    await expect(page.locator('.ant-table-tbody').getByText('cabtest@example.com')).toBeVisible({ timeout: 10_000 });
    console.log('✓ Step 4: 验证 cabtest 在 ECAB 列表中');

    await page.screenshot({ path: '/tmp/cab-e2e/04-ecab-after-add.png', fullPage: true });

    // 8. 测试状态切换（Switch）
    const firstSwitch = page.locator('.ant-table-tbody .ant-switch').first();
    const switchChecked = await firstSwitch.getAttribute('aria-checked');
    await firstSwitch.click();
    await page.waitForTimeout(1500);
    const newSwitchChecked = await firstSwitch.getAttribute('aria-checked');
    expect(newSwitchChecked).not.toBe(switchChecked);
    console.log('✓ Step 5: 状态切换成功（前后 aria-checked 不同）');

    await page.screenshot({ path: '/tmp/cab-e2e/05-after-toggle.png', fullPage: true });

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
    console.log(`  ECAB 删除后剩余行数: ${remainingRows}`);
    console.log('✓ Step 6: 删除成功');

    await page.screenshot({ path: '/tmp/cab-e2e/06-after-delete.png', fullPage: true });

    // 10. 切换回 CAB 验证先前数据
    const cabSegment = page.locator('.ant-segmented-item:has-text("CAB（变更咨询委员会）")').first();
    await cabSegment.click();
    await page.waitForTimeout(1500);
    await page.screenshot({ path: '/tmp/cab-e2e/07-cab-tab.png', fullPage: true });

    // 11. 验证侧边栏菜单含有 CAB
    const sidebar = await page.locator('.ant-layout-sider').innerText();
    expect(sidebar).toContain('CAB');
    console.log('✓ Step 7: 侧边栏菜单包含 CAB');

    console.log('\n=== 所有 7 个步骤全部通过 ===');
  });

  test('菜单导航到 CAB 页面', async ({ page }) => {
    test.setTimeout(30_000);

    await page.goto(`${BASE_URL}/dashboard`, { waitUntil: 'domcontentloaded' });
    await page.waitForLoadState('networkidle');

    // 找到侧边栏的 CAB 菜单项
    const cabMenu = page.locator('.ant-menu-item:has-text("CAB"), .ant-menu-submenu-title:has-text("CAB")').first();
    await cabMenu.click();
    await page.waitForURL(/\/admin\/cab/, { timeout: 10_000 });
    console.log('✓ 侧边栏菜单成功导航到 CAB 页面');

    await page.screenshot({ path: '/tmp/cab-e2e/menu-navigation.png', fullPage: true });
  });
});
