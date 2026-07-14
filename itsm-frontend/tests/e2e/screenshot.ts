/**
 * ITSM 自动化截图脚本
 *
 * 使用方法:
 * 1. 确保前后端服务已启动
 * 2. cd itsm-frontend
 * 3. npx playwright install chromium  (如果未安装)
 * 4. npx ts-node ../scripts/screenshot.ts
 *
 * 或者使用 npm 脚本:
 * npm run screenshot
 */

import type { Browser, Page } from '@playwright/test';
import { chromium } from '@playwright/test';
import path from 'path';
import fs from 'fs';

// 配置
const CONFIG = {
  baseURL: process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3000',
  username: 'admin',
  password: 'admin123',
  outputDir: path.join(__dirname, '..', '..', 'docs', 'images'),
};

// 需要截取的页面
const PAGES = [
  { path: '/login', name: 'login', fullPage: true, skipLogin: true },
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

interface PageConfig {
  path: string;
  name: string;
  fullPage: boolean;
  skipLogin?: boolean;
}

async function login(page: Page): Promise<boolean> {
  try {
    console.log('🔐 正在登录...');
    await page.goto(`${CONFIG.baseURL}/login`, { waitUntil: 'networkidle', timeout: 15000 });

    // 等待登录表单
    await page.waitForSelector('input[type="text"], input[name="username"], input[id="username"]', { timeout: 5000 }).catch(() => null);

    const usernameInput = await page.$('input[type="text"], input[name="username"], input[id="username"]');
    const passwordInput = await page.$('input[type="password"]');

    if (usernameInput && passwordInput) {
      await usernameInput.fill(CONFIG.username);
      await passwordInput.fill(CONFIG.password);

      const loginButton = await page.$('button[type="submit"], button:has-text("登录"), button:has-text("Sign in")');
      if (loginButton) {
        await loginButton.click();
        await page.waitForURL('**/dashboard', { timeout: 10000 }).catch(() => null);
        await page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => null);
        console.log('✅ 登录成功');
        return true;
      }
    }

    console.log('ℹ️ 可能已登录或无需登录');
    return true;
  } catch (error) {
    console.error('❌ 登录失败:', error);
    return false;
  }
}

async function capturePage(browser: Browser, pageConfig: PageConfig): Promise<void> {
  const page = await browser.newPage({
    viewport: { width: 1400, height: 900 },
  });

  try {
    const url = `${CONFIG.baseURL}${pageConfig.path}`;
    console.log(`📸 访问: ${url}`);

    if (pageConfig.skipLogin) {
      // 登录页面不需要登录
      await page.goto(url, { waitUntil: 'networkidle', timeout: 30000 });
    } else {
      // 先访问登录页面登录
      const loginSuccess = await login(page);
      if (!loginSuccess) {
        console.log(`⚠️ 跳过页面: ${pageConfig.path} (登录失败)`);
        return;
      }
      // 登录后访问目标页面
      await page.goto(url, { waitUntil: 'networkidle', timeout: 30000 });
    }

    // 等待页面渲染
    await page.waitForTimeout(2000);

    // 截取截图
    const filePath = path.join(CONFIG.outputDir, `${pageConfig.name}.png`);
    await page.screenshot({
      fullPage: pageConfig.fullPage,
      path: filePath,
      type: 'png',
      animations: 'disabled',
    });

    console.log(`✅ 已保存: ${filePath}`);
  } catch (error: any) {
    console.error(`❌ 截取失败 [${pageConfig.path}]: ${error.message}`);
  } finally {
    await page.close();
  }
}

async function main() {
  console.log('='.repeat(50));
  console.log('🚀 ITSM 自动化截图工具');
  console.log('='.repeat(50));
  console.log(`📡 目标: ${CONFIG.baseURL}`);
  console.log(`📁 输出: ${CONFIG.outputDir}`);
  console.log('');

  // 创建输出目录
  if (!fs.existsSync(CONFIG.outputDir)) {
    fs.mkdirSync(CONFIG.outputDir, { recursive: true });
    console.log(`✅ 创建输出目录: ${CONFIG.outputDir}`);
  }

  // 启动浏览器
  const browser = await chromium.launch({
    headless: true,
    args: ['--disable-dev-shm-usage', '--no-sandbox', '--disable-setuid-sandbox'],
  });

  try {
    let loginPerformed = false;

    for (const pageConfig of PAGES) {
      // 对于非登录页面，只在第一次需要登录
      if (!pageConfig.skipLogin && !loginPerformed) {
        // 使用新上下文确保登录状态
        const context = await browser.newContext();
        const loginPage = await context.newPage();

        const success = await login(loginPage);
        if (success) {
          loginPerformed = true;
          await context.close();
        }
      }

      await capturePage(browser, pageConfig);
    }

    console.log('');
    console.log('='.repeat(50));
    console.log('🎉 截图完成!');
    console.log(`📁 查看截图: ${path.resolve(CONFIG.outputDir)}`);
    console.log('='.repeat(50));

    // 列出生成的文件
    const files = fs.readdirSync(CONFIG.outputDir).filter(f => f.endsWith('.png'));
    console.log('\n📋 生成的文件:');
    files.forEach(f => console.log(`  - ${f}`));

  } catch (error: any) {
    console.error('❌ 错误:', error.message);
  } finally {
    await browser.close();
  }
}

main();
