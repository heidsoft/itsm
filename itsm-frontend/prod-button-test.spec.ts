import { test, expect, Page } from '@playwright/test';

const BASE = 'http://localhost:3000';
const ADMIN_USER = 'admin';
const ADMIN_PASS = 'AdminProd2026!';

async function login(page: Page) {
  await page.goto(BASE);
  await page.waitForTimeout(2000);
  // 如果已经在登录页
  const url = page.url();
  if (url.includes('/login') || url.includes('/auth')) {
    await page.fill('input[name="username"], input[placeholder*="用户"], input[type="text"]', ADMIN_USER);
    await page.fill('input[name="password"], input[placeholder*="密码"], input[type="password"]', ADMIN_PASS);
    await page.click('button[type="submit"], button:has-text("登录"), button:has-text("Login")');
    await page.waitForTimeout(3000);
  }
}

test.describe('ITSM 生产环境按钮级功能测试', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('01 - 登录页面', async ({ page }) => {
    await page.goto(BASE);
    await page.waitForTimeout(2000);
    const screenshot = '/tmp/itsm-test/01-login-page.png';
    await page.screenshot({ path: screenshot, fullPage: true });
    console.log('Login page screenshot saved');
  });

  test('02 - 仪表盘按钮测试', async ({ page }) => {
    await page.goto(BASE);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/02-dashboard.png', fullPage: true });
    
    // 测试侧边栏导航按钮
    const sidebarButtons = await page.locator('nav a, aside a, [role="menuitem"]').all();
    console.log(`Dashboard sidebar buttons found: ${sidebarButtons.length}`);
    
    // 测试顶部工具栏按钮
    const topButtons = await page.locator('header button, [data-testid]').all();
    console.log(`Top toolbar buttons found: ${topButtons.length}`);
  });

  test('03 - 工单列表按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/tickets`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/03-ticket-list.png', fullPage: true });
    
    // 测试"新建"按钮
    const createBtn = page.locator('button:has-text("新建"), button:has-text("创建"), button:has-text("New"), a:has-text("新建")').first();
    if (await createBtn.isVisible()) {
      console.log('✅ 新建工单按钮可见');
      await createBtn.click();
      await page.waitForTimeout(2000);
      await page.screenshot({ path: '/tmp/itsm-test/03-ticket-create-modal.png', fullPage: true });
      console.log('✅ 新建工单按钮可点击');
      // 关闭弹窗
      await page.keyboard.press('Escape');
      await page.waitForTimeout(500);
    } else {
      console.log('❌ 新建工单按钮不可见');
    }

    // 测试搜索框
    const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();
    if (await searchInput.isVisible()) {
      console.log('✅ 搜索框可见');
      await searchInput.fill('test');
      await page.waitForTimeout(1000);
      console.log('✅ 搜索框可输入');
    }

    // 测试筛选按钮
    const filterBtn = page.locator('button:has-text("筛选"), button:has-text("过滤"), button:has-text("Filter")').first();
    if (await filterBtn.isVisible()) {
      console.log('✅ 筛选按钮可见');
    }

    // 测试分页按钮
    const paginationBtns = await page.locator('.ant-pagination button, nav[aria-label="pagination"] button').all();
    console.log(`分页按钮数量: ${paginationBtns.length}`);
  });

  test('04 - 工单详情按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/tickets`);
    await page.waitForTimeout(3000);
    
    // 点击第一个工单
    const firstTicket = page.locator('table tbody tr').first();
    if (await firstTicket.isVisible()) {
      await firstTicket.click();
      await page.waitForTimeout(3000);
      await page.screenshot({ path: '/tmp/itsm-test/04-ticket-detail.png', fullPage: true });
      console.log('✅ 工单详情页可访问');
      
      // 测试操作按钮
      const actionBtns = await page.locator('button').all();
      for (const btn of actionBtns.slice(0, 10)) {
        const text = await btn.textContent();
        if (text?.trim()) {
          console.log(`  按钮: ${text.trim()}`);
        }
      }
    }
  });

  test('05 - 事件管理按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/incidents`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/05-incidents.png', fullPage: true });
    
    const createBtn = page.locator('button:has-text("新建"), button:has-text("创建"), button:has-text("New")').first();
    if (await createBtn.isVisible()) {
      console.log('✅ 事件新建按钮可见');
    } else {
      console.log('❌ 事件新建按钮不可见');
    }
  });

  test('06 - 问题管理按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/problems`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/06-problems.png', fullPage: true });
    
    const createBtn = page.locator('button:has-text("新建"), button:has-text("创建"), button:has-text("New")').first();
    if (await createBtn.isVisible()) {
      console.log('✅ 问题新建按钮可见');
    } else {
      console.log('❌ 问题新建按钮不可见');
    }
  });

  test('07 - 变更管理按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/changes`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/07-changes.png', fullPage: true });
    
    const createBtn = page.locator('button:has-text("新建"), button:has-text("创建"), button:has-text("New")').first();
    if (await createBtn.isVisible()) {
      console.log('✅ 变更新建按钮可见');
    } else {
      console.log('❌ 变更新建按钮不可见');
    }
  });

  test('08 - CMDB 按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/cmdb`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/08-cmdb.png', fullPage: true });
    
    const buttons = await page.locator('button').all();
    console.log(`CMDB 页面按钮数量: ${buttons.length}`);
  });

  test('09 - 服务目录按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/service-catalog`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/09-service-catalog.png', fullPage: true });
    
    const buttons = await page.locator('button').all();
    console.log(`服务目录按钮数量: ${buttons.length}`);
  });

  test('10 - 知识库按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/knowledge`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/10-knowledge.png', fullPage: true });
    
    const createBtn = page.locator('button:has-text("新建"), button:has-text("创建"), button:has-text("New")').first();
    if (await createBtn.isVisible()) {
      console.log('✅ 知识库新建按钮可见');
    } else {
      console.log('❌ 知识库新建按钮不可见');
    }
  });

  test('11 - SLA 监控按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/sla`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/11-sla.png', fullPage: true });
    
    const buttons = await page.locator('button').all();
    console.log(`SLA 页面按钮数量: ${buttons.length}`);
  });

  test('12 - 工作流按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/workflow`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/12-workflow.png', fullPage: true });
    
    const buttons = await page.locator('button').all();
    console.log(`工作流页面按钮数量: ${buttons.length}`);
  });

  test('13 - 系统管理按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/system/users`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/13-system-users.png', fullPage: true });
    
    const createBtn = page.locator('button:has-text("新建"), button:has-text("创建"), button:has-text("New")').first();
    if (await createBtn.isVisible()) {
      console.log('✅ 用户管理新建按钮可见');
    } else {
      console.log('❌ 用户管理新建按钮不可见');
    }
  });

  test('14 - 发布管理按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/releases`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/14-releases.png', fullPage: true });
    
    const buttons = await page.locator('button').all();
    console.log(`发布管理按钮数量: ${buttons.length}`);
  });

  test('15 - 标准变更按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/standard-changes`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/15-standard-changes.png', fullPage: true });
    
    const buttons = await page.locator('button').all();
    console.log(`标准变更按钮数量: ${buttons.length}`);
  });

  test('16 - 服务请求按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/service-requests`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/16-service-requests.png', fullPage: true });
    
    const buttons = await page.locator('button').all();
    console.log(`服务请求按钮数量: ${buttons.length}`);
  });

  test('17 - AI 助手按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/ai/chat`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/17-ai-chat.png', fullPage: true });
    
    const buttons = await page.locator('button').all();
    console.log(`AI 助手按钮数量: ${buttons.length}`);
  });

  test('18 - 仪表盘详情按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/dashboard`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/18-dashboard-detail.png', fullPage: true });
    
    const buttons = await page.locator('button').all();
    console.log(`仪表盘按钮数量: ${buttons.length}`);
  });
});
