import { chromium } from 'playwright';
import path from 'path';
import fs from 'fs';

// 截图配置
const config = {
  baseURL: process.env.BASE_URL || 'http://localhost:3000',
  username: 'admin',
  password: 'admin123',
  outputDir: './screenshots',
};

// 需要截取的页面
const pages = [
  { path: '/login', name: 'login', fullPage: true },
  { path: '/dashboard', name: 'dashboard', fullPage: true },
  { path: '/tickets', name: 'tickets', fullPage: true },
  { path: '/incidents', name: 'incidents', fullPage: true },
  { path: '/problems', name: 'problems', fullPage: true },
  { path: '/changes', name: 'changes', fullPage: true },
  { path: '/knowledge', name: 'knowledge', fullPage: true },
  { path: '/workflow/designer', name: 'workflow-designer', fullPage: true },
  { path: '/workflow/dashboard', name: 'workflow-dashboard', fullPage: true },
  { path: '/sla-monitor', name: 'sla-monitor', fullPage: true },
  { path: '/cmdb', name: 'cmdb', fullPage: true },
  { path: '/assets', name: 'assets', fullPage: true },
  { path: '/msp/management', name: 'msp-management', fullPage: true },
];

async function login(page: any) {
  console.log('🔐 尝试登录...');
  await page.goto(`${config.baseURL}/login`);

  // 等待登录表单加载
  await page.waitForSelector('input[type="text"], input[name="username"]', { timeout: 10000 }).catch(() => null);

  // 尝试多种登录方式
  const usernameInput = await page.$('input[type="text"]') || await page.$('input[name="username"]');
  const passwordInput = await page.$('input[type="password"]');

  if (usernameInput && passwordInput) {
    await usernameInput.fill(config.username);
    await passwordInput.fill(config.password);

    // 查找登录按钮并点击
    const loginButton = await page.$('button[type="submit"]') || await page.$('button:has-text("登录")');
    if (loginButton) {
      await loginButton.click();
      await page.waitForURL('**/dashboard', { timeout: 10000 }).catch(() => null);
      console.log('✅ 登录成功');
    }
  } else {
    // 可能已经登录，直接跳转
    console.log('ℹ️ 可能已登录或无需登录');
  }

  // 等待页面加载完成
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => null);
}

async function takeScreenshot(browser: any, pagePath: string, name: string, fullPage: boolean) {
  const page = await browser.newPage();

  try {
    console.log(`📸 截取页面: ${pagePath}`);

    // 如果需要登录且不是登录页面，先登录
    if (pagePath !== '/login') {
      await page.goto(`${config.baseURL}/login`);
      await page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => null);

      const usernameInput = await page.$('input[type="text"]') || await page.$('input[name="username"]');
      const passwordInput = await page.$('input[type="password"]');

      if (usernameInput && passwordInput) {
        await usernameInput.fill(config.username);
        await passwordInput.fill(config.password);
        const loginButton = await page.$('button[type="submit"]') || await page.$('button:has-text("登录")');
        if (loginButton) {
          await loginButton.click();
          await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => null);
        }
      }
    }

    // 访问目标页面
    await page.goto(`${config.baseURL}${pagePath}`, { waitUntil: 'networkidle', timeout: 30000 });

    // 等待页面渲染
    await page.waitForTimeout(2000);

    // 截取截图
    const filePath = path.join(config.outputDir, `${name}.png`);
    await page.screenshot({ fullPage, path: filePath, type: 'png' });

    console.log(`✅ 已保存: ${filePath}`);
  } catch (error: any) {
    console.error(`❌ 页面截取失败 ${pagePath}: ${error.message}`);
  } finally {
    await page.close();
  }
}

async function main() {
  console.log('🚀 ITSM 自动化截图工具\n');

  // 创建输出目录
  if (!fs.existsSync(config.outputDir)) {
    fs.mkdirSync(config.outputDir, { recursive: true });
  }

  // 启动浏览器
  const browser = await chromium.launch({
    headless: true,
    args: ['--disable-dev-shm-usage', '--no-sandbox'],
  });

  try {
    // 先登录获取会话
    const context = await browser.newContext();
    const loginPage = await context.newPage();

    console.log('🔐 正在登录...');
    await loginPage.goto(`${config.baseURL}/login`);
    await loginPage.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => null);

    const usernameInput = await loginPage.$('input[type="text"]') || await loginPage.$('input[name="username"]');
    const passwordInput = await loginPage.$('input[type="password"]');

    if (usernameInput && passwordInput) {
      await usernameInput.fill(config.username);
      await passwordInput.fill(config.password);

      const loginButton = await loginPage.$('button[type="submit"]') || await loginPage.$('button:has-text("登录")');
      if (loginButton) {
        await Promise.all([
          loginButton.click(),
          loginPage.waitForURL('**/dashboard', { timeout: 15000 }).catch(() => null),
        ]);
      }
    }
    await loginPage.waitForTimeout(2000);
    console.log('✅ 登录完成\n');

    // 截取每个页面
    for (const page of pages) {
      await takeScreenshot(browser, page.path, page.name, page.fullPage);
    }

    console.log('\n🎉 截图完成！');
    console.log(`📁 截图保存在: ${path.resolve(config.outputDir)}`);

  } catch (error: any) {
    console.error('❌ 错误:', error.message);
  } finally {
    await browser.close();
  }
}

main();
