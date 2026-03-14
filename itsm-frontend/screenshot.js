const { chromium } = require('playwright');
const path = require('path');

const outputDir = '/Users/heidsoft/Downloads/research/itsm/docs/images';

async function main() {
  console.log('🚀 开始截图...\n');

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 1400, height: 900 }
  });
  const page = await context.newPage();

  // 启用网络拦截来查看请求
  page.on('console', msg => {
    if (msg.type() === 'error') {
      console.log('Console Error:', msg.text());
    }
  });

  // 1. 先登录
  console.log('🔐 登录中...');
  await page.goto('http://localhost:3000/login', { waitUntil: 'networkidle' });
  await page.waitForSelector('.ant-input', { timeout: 15000 });

  // 填写登录表单
  const inputs = page.locator('input.ant-input');
  await inputs.nth(0).fill('admin');
  await inputs.nth(1).fill('admin123');

  // 点击登录按钮
  await page.click('button[type="submit"]');

  // 等待网络请求完成
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(3000);

  // 检查是否登录成功（URL 变化或者看是否还有登录表单）
  const currentUrl = page.url();
  console.log('📍 当前 URL:', currentUrl);

  // 如果还在登录页，打印页面内容调试
  if (currentUrl.includes('login')) {
    const errorMsg = await page.$('.ant-message-error, .ant-alert-error, [class*="error"]');
    if (errorMsg) {
      console.log('❌ 登录错误:', await errorMsg.textContent());
    } else {
      // 打印网络请求
      console.log('⚠️ 仍在登录页，可能是网络问题');
    }
  } else {
    console.log('✅ 登录成功！');
  }

  // 2. 访问各个页面截图
  const pages = [
    { url: 'http://localhost:3000/dashboard', name: 'dashboard' },
    { url: 'http://localhost:3000/tickets', name: 'tickets' },
    { url: 'http://localhost:3000/incidents', name: 'incidents' },
    { url: 'http://localhost:3000/problems', name: 'problems' },
    { url: 'http://localhost:3000/changes', name: 'changes' },
    { url: 'http://localhost:3000/knowledge', name: 'knowledge' },
    { url: 'http://localhost:3000/workflow/designer', name: 'workflow-designer' },
    { url: 'http://localhost:3000/workflow/dashboard', name: 'workflow-dashboard' },
    { url: 'http://localhost:3000/sla-monitor', name: 'sla-monitor' },
    { url: 'http://localhost:3000/cmdb', name: 'cmdb' },
    { url: 'http://localhost:3000/assets', name: 'assets' },
    { url: 'http://localhost:3000/msp/management', name: 'msp-management' },
  ];

  for (const p of pages) {
    try {
      console.log(`📸 访问: ${p.name}`);
      await page.goto(p.url, { waitUntil: 'networkidle', timeout: 30000 });
      await page.waitForTimeout(2000);

      await page.screenshot({
        fullPage: true,
        path: `${outputDir}/${p.name}.png`
      });
      console.log(`✅ 已保存: ${p.name}.png`);
    } catch (e) {
      console.error(`❌ 失败: ${p.name} - ${e.message}`);
    }
  }

  console.log('\n🎉 截图完成!');
  await browser.close();
}

main().catch(console.error);
