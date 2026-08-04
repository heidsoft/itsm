import { test, expect, Page } from '@playwright/test';

const BASE = 'http://localhost:3000';
const ADMIN_USER = 'admin';
const ADMIN_PASS = 'AdminProd2026!';

async function login(page: Page) {
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(2000);
  
  // 填写登录表单
  await page.locator('input[placeholder*="用户名"], input[name="username"], input[type="text"]').first().fill(ADMIN_USER);
  await page.locator('input[type="password"]').first().fill(ADMIN_PASS);
  await page.locator('button[type="submit"], button:has-text("登录")').first().click();
  await page.waitForTimeout(5000);
  
  // 验证登录成功
  const url = page.url();
  console.log(`登录后URL: ${url}`);
  if (url.includes('/login')) {
    throw new Error('登录失败，仍在登录页面');
  }
}

test.describe('ITSM 生产环境按钮级功能测试', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('01 - 仪表盘按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/dashboard`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/01-dashboard.png', fullPage: true });
    
    const currentUrl = page.url();
    console.log(`仪表盘URL: ${currentUrl}`);
    
    // 测试侧边栏导航
    const sidebarLinks = await page.locator('nav a, aside a, [role="menuitem"], .ant-menu-item').all();
    console.log(`侧边栏导航项: ${sidebarLinks.length}`);
    for (const link of sidebarLinks.slice(0, 15)) {
      const text = await link.textContent();
      if (text?.trim()) console.log(`  - ${text.trim()}`);
    }
    
    // 测试顶部工具栏
    const headerBtns = await page.locator('header button, [data-testid]').all();
    console.log(`顶部工具栏按钮: ${headerBtns.length}`);
    
    // 测试页面内容按钮
    const pageBtns = await page.locator('main button, [role="main"] button').all();
    console.log(`页面内容按钮: ${pageBtns.length}`);
  });

  test('02 - 工单列表按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/tickets`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/02-ticket-list.png', fullPage: true });
    
    console.log(`工单列表URL: ${page.url()}`);
    
    // 测试"新建"按钮
    const createBtn = page.locator('button:has-text("新建"), button:has-text("创建"), button:has-text("New"), a:has-text("新建"), a[href*="create"]').first();
    if (await createBtn.isVisible({ timeout: 5000 }).catch(() => false)) {
      console.log('✅ 新建工单按钮可见');
      await createBtn.click();
      await page.waitForTimeout(2000);
      await page.screenshot({ path: '/tmp/itsm-test/02-ticket-create.png', fullPage: true });
      console.log('✅ 新建工单按钮可点击');
      await page.keyboard.press('Escape');
      await page.waitForTimeout(500);
    } else {
      console.log('❌ 新建工单按钮不可见');
      // 列出所有可见按钮
      const allBtns = await page.locator('button:visible').all();
      for (const btn of allBtns) {
        const text = await btn.textContent();
        if (text?.trim()) console.log(`  可见按钮: "${text.trim()}"`);
      }
    }
    
    // 测试搜索框
    const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();
    if (await searchInput.isVisible({ timeout: 3000 }).catch(() => false)) {
      console.log('✅ 搜索框可见');
      await searchInput.fill('test');
      await page.waitForTimeout(1000);
      console.log('✅ 搜索框可输入');
      await searchInput.clear();
    } else {
      console.log('❌ 搜索框不可见');
    }
    
    // 测试筛选/Tab按钮
    const tabBtns = await page.locator('.ant-tabs-tab, [role="tab"]').all();
    console.log(`Tab按钮数量: ${tabBtns.length}`);
    for (const tab of tabBtns) {
      const text = await tab.textContent();
      if (text?.trim()) console.log(`  Tab: "${text.trim()}"`);
    }
    
    // 测试表格行数
    const rows = await page.locator('table tbody tr, .ant-table-row').all();
    console.log(`表格行数: ${rows.length}`);
  });

  test('03 - 工单详情按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/tickets`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    
    const rows = await page.locator('table tbody tr, .ant-table-row').all();
    if (rows.length > 0) {
      await rows[0].click();
      await page.waitForTimeout(3000);
      await page.screenshot({ path: '/tmp/itsm-test/03-ticket-detail.png', fullPage: true });
      console.log(`工单详情URL: ${page.url()}`);
      
      // 列出所有操作按钮
      const actionBtns = await page.locator('button:visible').all();
      console.log(`详情页按钮数量: ${actionBtns.length}`);
      for (const btn of actionBtns) {
        const text = await btn.textContent();
        if (text?.trim()) console.log(`  按钮: "${text.trim()}"`);
      }
    } else {
      console.log('❌ 没有工单数据');
    }
  });

  test('04 - 事件管理按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/incidents`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/04-incidents.png', fullPage: true });
    console.log(`事件管理URL: ${page.url()}`);
    
    const btns = await page.locator('button:visible').all();
    console.log(`按钮数量: ${btns.length}`);
    for (const btn of btns) {
      const text = await btn.textContent();
      if (text?.trim()) console.log(`  按钮: "${text.trim()}"`);
    }
  });

  test('05 - 问题管理按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/problems`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/05-problems.png', fullPage: true });
    console.log(`问题管理URL: ${page.url()}`);
    
    const btns = await page.locator('button:visible').all();
    console.log(`按钮数量: ${btns.length}`);
    for (const btn of btns) {
      const text = await btn.textContent();
      if (text?.trim()) console.log(`  按钮: "${text.trim()}"`);
    }
  });

  test('06 - 变更管理按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/changes`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/06-changes.png', fullPage: true });
    console.log(`变更管理URL: ${page.url()}`);
    
    const btns = await page.locator('button:visible').all();
    console.log(`按钮数量: ${btns.length}`);
    for (const btn of btns) {
      const text = await btn.textContent();
      if (text?.trim()) console.log(`  按钮: "${text.trim()}"`);
    }
  });

  test('07 - CMDB 按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/cmdb`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/07-cmdb.png', fullPage: true });
    console.log(`CMDB URL: ${page.url()}`);
    
    const btns = await page.locator('button:visible').all();
    console.log(`按钮数量: ${btns.length}`);
    for (const btn of btns) {
      const text = await btn.textContent();
      if (text?.trim()) console.log(`  按钮: "${text.trim()}"`);
    }
  });

  test('08 - 服务目录按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/service-catalog`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/08-service-catalog.png', fullPage: true });
    console.log(`服务目录URL: ${page.url()}`);
    
    const btns = await page.locator('button:visible').all();
    console.log(`按钮数量: ${btns.length}`);
    for (const btn of btns) {
      const text = await btn.textContent();
      if (text?.trim()) console.log(`  按钮: "${text.trim()}"`);
    }
  });

  test('09 - 知识库按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/knowledge`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/09-knowledge.png', fullPage: true });
    console.log(`知识库URL: ${page.url()}`);
    
    const btns = await page.locator('button:visible').all();
    console.log(`按钮数量: ${btns.length}`);
    for (const btn of btns) {
      const text = await btn.textContent();
      if (text?.trim()) console.log(`  按钮: "${text.trim()}"`);
    }
  });

  test('10 - SLA 监控按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/sla`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/10-sla.png', fullPage: true });
    console.log(`SLA URL: ${page.url()}`);
    
    const btns = await page.locator('button:visible').all();
    console.log(`按钮数量: ${btns.length}`);
    for (const btn of btns) {
      const text = await btn.textContent();
      if (text?.trim()) console.log(`  按钮: "${text.trim()}"`);
    }
  });

  test('11 - 工作流按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/workflow`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/11-workflow.png', fullPage: true });
    console.log(`工作流URL: ${page.url()}`);
    
    const btns = await page.locator('button:visible').all();
    console.log(`按钮数量: ${btns.length}`);
    for (const btn of btns) {
      const text = await btn.textContent();
      if (text?.trim()) console.log(`  按钮: "${text.trim()}"`);
    }
  });

  test('12 - 系统管理按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/system/users`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/12-system-users.png', fullPage: true });
    console.log(`系统用户URL: ${page.url()}`);
    
    const btns = await page.locator('button:visible').all();
    console.log(`按钮数量: ${btns.length}`);
    for (const btn of btns) {
      const text = await btn.textContent();
      if (text?.trim()) console.log(`  按钮: "${text.trim()}"`);
    }
  });

  test('13 - 发布管理按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/releases`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/13-releases.png', fullPage: true });
    console.log(`发布管理URL: ${page.url()}`);
    
    const btns = await page.locator('button:visible').all();
    console.log(`按钮数量: ${btns.length}`);
    for (const btn of btns) {
      const text = await btn.textContent();
      if (text?.trim()) console.log(`  按钮: "${text.trim()}"`);
    }
  });

  test('14 - 标准变更按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/standard-changes`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/14-standard-changes.png', fullPage: true });
    console.log(`标准变更URL: ${page.url()}`);
    
    const btns = await page.locator('button:visible').all();
    console.log(`按钮数量: ${btns.length}`);
    for (const btn of btns) {
      const text = await btn.textContent();
      if (text?.trim()) console.log(`  按钮: "${text.trim()}"`);
    }
  });

  test('15 - 服务请求按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/service-requests`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/15-service-requests.png', fullPage: true });
    console.log(`服务请求URL: ${page.url()}`);
    
    const btns = await page.locator('button:visible').all();
    console.log(`按钮数量: ${btns.length}`);
    for (const btn of btns) {
      const text = await btn.textContent();
      if (text?.trim()) console.log(`  按钮: "${text.trim()}"`);
    }
  });

  test('16 - AI 助手按钮测试', async ({ page }) => {
    await page.goto(`${BASE}/ai/chat`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/itsm-test/16-ai-chat.png', fullPage: true });
    console.log(`AI 助手URL: ${page.url()}`);
    
    const btns = await page.locator('button:visible').all();
    console.log(`按钮数量: ${btns.length}`);
    for (const btn of btns) {
      const text = await btn.textContent();
      if (text?.trim()) console.log(`  按钮: "${text.trim()}"`);
    }
  });
});
