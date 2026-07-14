// 详细验证 BPMN 画布 root 元素
const { chromium } = require('playwright');

const API_URL = 'http://localhost:8090';
const BASE_URL = 'http://localhost:3000';

(async () => {
  const browser = await chromium.launch({ headless: true });
  const ctx = await browser.newContext({ viewport: { width: 1600, height: 1000 } });
  const page = await ctx.newPage();

  const errors = [];
  page.on('pageerror', (err) => {
    errors.push(`[pageerror] ${err.message}`);
  });
  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      errors.push(`[console.error] ${msg.text().slice(0, 300)}`);
    }
  });

  // 登录
  const loginRes = await page.request.post(`${API_URL}/api/v1/auth/login`, {
    headers: { 'Content-Type': 'application/json' },
    data: { username: 'admin', password: 'admin123' },
  });
  const loginData = await loginRes.json();
  const token = loginData.data.access_token;

  await page.goto(BASE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.evaluate(({ token }) => {
    localStorage.setItem('access_token', token);
    localStorage.setItem('user', JSON.stringify({ id: 1, username: 'admin', name: '系统管理员', role: 'super_admin' }));
  }, { token });

  await page.goto(`${BASE_URL}/workflow/designer`, { waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.waitForTimeout(2000);

  // 关闭新工作流模态
  await page.keyboard.press('Escape');
  await page.waitForTimeout(500);

  // 输入名称并创建
  const nameInput = page.locator('input[placeholder="自定义流程名称"]');
  if (await nameInput.count() > 0) {
    // 重新打开模态
    // 这里直接调用 API 创建简单 BPMN XML 来测试画布
  }

  // 走捷径：直接通过 setState 不行，我们直接验证已有画布渲染
  // 等待更长时间
  await page.waitForTimeout(3000);

  // 获取画布内部所有 DOM 元素的详细信息
  const canvasInfo = await page.evaluate(() => {
    const containers = document.querySelectorAll('.djs-container, .bjs-container, .diagram-container');
    const svgs = document.querySelectorAll('.djs-container svg, .bjs-container svg, .diagram-container svg');
    const djss = document.querySelectorAll('[class*="djs-"], [class*="bjs-"], [class^="djs"], [class^="bjs"]');

    const elementDetails = [];
    djss.forEach((el, i) => {
      if (i < 30) {
        elementDetails.push({
          tag: el.tagName,
          cls: el.getAttribute('class')?.slice(0, 100),
          dataElementId: el.getAttribute('data-element-id'),
        });
      }
    });

    return {
      containerCount: containers.length,
      svgCount: svgs.length,
      djsElementCount: djss.length,
      elements: elementDetails,
      firstContainerHTML: containers[0]?.outerHTML?.slice(0, 800),
    };
  });

  console.log('===== 画布 DOM 详情 =====');
  console.log(`容器数量: ${canvasInfo.containerCount}`);
  console.log(`SVG 数量: ${canvasInfo.svgCount}`);
  console.log(`djs 元素数量: ${canvasInfo.djsElementCount}`);
  console.log('\n===== djs 元素详情 =====');
  canvasInfo.elements.forEach((el, i) => {
    console.log(`  [${i}] <${el.tag}> class="${el.cls}" data-element-id="${el.dataElementId}"`);
  });

  console.log('\n===== 第一个容器 HTML =====');
  console.log(canvasInfo.firstContainerHTML);

  console.log('\n===== 错误 =====');
  console.log(`错误数: ${errors.length}`);
  errors.slice(0, 10).forEach(e => console.log(`  - ${e}`));

  await page.screenshot({ path: '/tmp/itsm-designer-fix-v2/06-canvas-detail.png', fullPage: true });

  await browser.close();
})().catch(err => {
  console.error('脚本失败:', err);
  process.exit(1);
});
