import { test, expect, Page } from '@playwright/test';

const BASE = 'http://localhost:3000';
const ADMIN_USER = 'admin';
const ADMIN_PASS = 'AdminProd2026!';

test.describe('ITSM 生产环境诊断测试', () => {
  test('登录流程详细诊断', async ({ page }) => {
    // 1. 访问首页
    await page.goto(BASE, { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    
    const url1 = page.url();
    console.log(`[1] 首页URL: ${url1}`);
    await page.screenshot({ path: '/tmp/itsm-test/diag-01-home.png', fullPage: true });
    
    // 2. 获取页面HTML结构
    const bodyHTML = await page.evaluate(() => document.body.innerHTML.substring(0, 2000));
    console.log(`[2] 页面HTML前2000字符: ${bodyHTML.substring(0, 500)}`);
    
    // 3. 检查是否有登录表单
    const loginForm = await page.locator('form, input[type="password"], input[name="password"]').count();
    console.log(`[3] 登录表单元素数量: ${loginForm}`);
    
    // 4. 查找所有input元素
    const inputs = await page.locator('input').all();
    for (const input of inputs) {
      const type = await input.getAttribute('type');
      const name = await input.getAttribute('name');
      const placeholder = await input.getAttribute('placeholder');
      console.log(`  Input: type=${type}, name=${name}, placeholder=${placeholder}`);
    }
    
    // 5. 查找所有button元素
    const buttons = await page.locator('button').all();
    for (const btn of buttons) {
      const text = await btn.textContent();
      const type = await btn.getAttribute('type');
      console.log(`  Button: text="${text?.trim()}", type=${type}`);
    }
    
    // 6. 尝试登录
    if (url1.includes('/login') || loginForm > 0) {
      console.log('[6] 检测到登录页面，尝试登录...');
      
      // 填写用户名
      const usernameInput = page.locator('input[name="username"], input[type="text"]').first();
      await usernameInput.fill(ADMIN_USER);
      console.log('  已填写用户名');
      
      // 填写密码
      const passwordInput = page.locator('input[type="password"]').first();
      await passwordInput.fill(ADMIN_PASS);
      console.log('  已填写密码');
      
      await page.screenshot({ path: '/tmp/itsm-test/diag-02-login-filled.png', fullPage: true });
      
      // 点击登录
      const submitBtn = page.locator('button[type="submit"], button:has-text("登录"), button:has-text("Login")').first();
      await submitBtn.click();
      console.log('  已点击登录按钮');
      
      // 等待导航
      await page.waitForTimeout(5000);
      const url2 = page.url();
      console.log(`[7] 登录后URL: ${url2}`);
      await page.screenshot({ path: '/tmp/itsm-test/diag-03-after-login.png', fullPage: true });
      
      // 检查cookies
      const cookies = await page.context().cookies();
      console.log(`[8] Cookies数量: ${cookies.length}`);
      for (const c of cookies) {
        console.log(`  Cookie: ${c.name}=${c.value.substring(0, 30)}...`);
      }
      
      // 检查localStorage
      const token = await page.evaluate(() => localStorage.getItem('accessToken') || localStorage.getItem('token') || 'NOT_FOUND');
      console.log(`[9] localStorage token: ${token.substring(0, 50)}`);
      
      // 检查页面内容
      const pageText = await page.evaluate(() => document.body.innerText.substring(0, 1000));
      console.log(`[10] 页面文本前1000字符: ${pageText.substring(0, 300)}`);
      
      // 检查网络请求
      const consoleMessages: string[] = [];
      page.on('console', msg => consoleMessages.push(msg.text()));
      
    } else {
      console.log('[6] 未检测到登录页面，可能已登录');
      const pageText = await page.evaluate(() => document.body.innerText.substring(0, 500));
      console.log(`  页面文本: ${pageText.substring(0, 300)}`);
    }
  });

  test('各页面详细诊断', async ({ page }) => {
    // 先登录
    await page.goto(BASE, { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    
    const url = page.url();
    if (url.includes('/login') || await page.locator('input[type="password"]').count() > 0) {
      await page.locator('input[name="username"], input[type="text"]').first().fill(ADMIN_USER);
      await page.locator('input[type="password"]').first().fill(ADMIN_PASS);
      await page.locator('button[type="submit"], button:has-text("登录")').first().click();
      await page.waitForTimeout(5000);
    }
    
    const afterLoginUrl = page.url();
    console.log(`登录后URL: ${afterLoginUrl}`);
    await page.screenshot({ path: '/tmp/itsm-test/diag-04-logged-in.png', fullPage: true });
    
    // 测试各页面
    const pages = [
      { name: '工单列表', url: '/tickets' },
      { name: '事件管理', url: '/incidents' },
      { name: '问题管理', url: '/problems' },
      { name: '变更管理', url: '/changes' },
      { name: 'CMDB', url: '/cmdb' },
      { name: '服务目录', url: '/service-catalog' },
      { name: '知识库', url: '/knowledge' },
      { name: 'SLA监控', url: '/sla' },
      { name: '工作流', url: '/workflow' },
      { name: '系统用户', url: '/system/users' },
      { name: '发布管理', url: '/releases' },
      { name: '服务请求', url: '/service-requests' },
    ];
    
    for (const p of pages) {
      await page.goto(`${BASE}${p.url}`, { waitUntil: 'networkidle' });
      await page.waitForTimeout(3000);
      
      const currentUrl = page.url();
      const pageTitle = await page.title();
      const buttonCount = await page.locator('button').count();
      const linkCount = await page.locator('a').count();
      const tableRows = await page.locator('table tbody tr').count();
      const inputCount = await page.locator('input').count();
      
      // 检查是否有loading/空状态
      const hasLoading = await page.locator('.ant-spin, [class*="loading"], [class*="spinner"]').count();
      const hasEmpty = await page.locator('.ant-empty, [class*="empty"]').count();
      const hasError = await page.locator('.ant-result-error, [class*="error"]').count();
      
      console.log(`\n=== ${p.name} (${p.url}) ===`);
      console.log(`  实际URL: ${currentUrl}`);
      console.log(`  页面标题: ${pageTitle}`);
      console.log(`  按钮: ${buttonCount}, 链接: ${linkCount}, 表格行: ${tableRows}, 输入框: ${inputCount}`);
      console.log(`  Loading: ${hasLoading}, 空状态: ${hasEmpty}, 错误: ${hasError}`);
      
      // 获取页面主要文本
      const mainText = await page.evaluate(() => {
        const main = document.querySelector('main') || document.querySelector('[role="main"]') || document.body;
        return main.innerText.substring(0, 300);
      });
      console.log(`  页面内容: ${mainText.substring(0, 200)}`);
      
      await page.screenshot({ path: `/tmp/itsm-test/diag-page-${p.url.replace(/\//g, '-')}.png`, fullPage: true });
    }
  });
});
