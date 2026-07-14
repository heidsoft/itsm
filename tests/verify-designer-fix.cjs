// 工作流设计器修复验证脚本
const { chromium } = require('playwright');

const BASE_URL = 'http://localhost:3000';
const API_URL = 'http://localhost:8090';
const SCREENSHOT_DIR = '/tmp/itsm-designer-fix';

(async () => {
  const fs = require('fs');
  if (!fs.existsSync(SCREENSHOT_DIR)) {
    fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
  }

  const browser = await chromium.launch({ headless: true });
  const ctx = await browser.newContext({ viewport: { width: 1600, height: 1000 } });
  const page = await ctx.newPage();

  const errors = [];
  const warnings = [];

  page.on('pageerror', (err) => {
    errors.push(`[pageerror] ${err.message}`);
  });
  page.on('console', (msg) => {
    const type = msg.type();
    const text = msg.text();
    if (type === 'error') errors.push(`[console.error] ${text.slice(0, 250)}`);
    if (type === 'warning') {
      if (
        text.includes('destroyOnClose') ||
        text.includes('deprecated') ||
        text.includes('Warning:')
      ) {
        warnings.push(`[console.warn] ${text.slice(0, 250)}`);
      }
    }
  });

  console.log('===== 1. 登录 =====');
  const loginRes = await page.request.post(`${API_URL}/api/v1/auth/login`, {
    headers: { 'Content-Type': 'application/json' },
    data: { username: 'admin', password: 'admin123' },
  });
  const loginData = await loginRes.json();
  if (!loginData.data || !loginData.data.access_token) {
    console.error('登录失败', loginData);
    process.exit(1);
  }
  const token = loginData.data.access_token;
  console.log('Token 已获取');

  // 通过 localStorage 注入登录态
  await page.goto(BASE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.evaluate(({ token }) => {
    localStorage.setItem('access_token', token);
    localStorage.setItem('user', JSON.stringify({ id: 1, username: 'admin', name: '系统管理员', role: 'super_admin' }));
  }, { token });

  console.log('===== 2. 打开工作流设计器 =====');
  // 触发新工作流模态
  await page.goto(`${BASE_URL}/workflow/designer`, { waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.waitForTimeout(3000);
  await page.screenshot({ path: `${SCREENSHOT_DIR}/01-designer-with-modal.png`, fullPage: true });
  console.log('截图: 01-designer-with-modal.png');

  // 检查新工作流模态
  const modalVisible = await page.locator('text=选择工作流模板').isVisible().catch(() => false);
  console.log(`新工作流模态可见: ${modalVisible}`);

  if (modalVisible) {
    // 点击 "创建空白流程"
    const blankButton = page.locator('button:has-text("创建空白流程")');
    if (await blankButton.count() > 0) {
      await blankButton.first().click();
      console.log('点击创建空白流程');
    }
  }

  await page.waitForTimeout(5000);
  await page.screenshot({ path: `${SCREENSHOT_DIR}/02-blank-canvas.png`, fullPage: true });
  console.log('截图: 02-blank-canvas.png');

  // 验证 BPMN 画布是否渲染（关键 root-0 bug 修复）
  const canvasExists = await page.locator('.djs-container, .bjs-container, .diagram-container').count();
  console.log(`BPMN 画布元素数量: ${canvasExists}`);

  const svgInCanvas = await page.locator('.djs-container svg, .bjs-container svg').count();
  console.log(`画布内 SVG 元素数量: ${svgInCanvas}`);

  // 检查画布是否包含根元素 (修复前会缺少 root-0)
  const rootElement = await page.locator('[data-element-id="root-0"], .djs-root, .bjs-root').count();
  console.log(`BPMN 画布根元素数量: ${rootElement}`);

  console.log('===== 3. 切换到流程配置 Tab =====');
  // 切换到 "流程配置" Tab
  const configTab = page.locator('div[role="tab"]:has-text("流程配置")');
  if (await configTab.count() > 0) {
    await configTab.first().click();
    await page.waitForTimeout(2000);
  } else {
    console.log('未找到"流程配置"Tab');
  }
  await page.screenshot({ path: `${SCREENSHOT_DIR}/03-config-tab.png`, fullPage: true });
  console.log('截图: 03-config-tab.png');

  // 验证 "审批组" 字段是否出现
  const approverGroupsLabel = await page.locator('text=审批组').count();
  console.log(`"审批组" 标签出现次数: ${approverGroupsLabel}`);

  // 验证审批组选择器
  const approverGroupsSelect = await page.locator('div.ant-select:has(input[placeholder*="审批组"])').count();
  console.log(`审批组 Select 组件数量: ${approverGroupsSelect}`);

  // 检查 Group API 调用
  const groupApiCalls = [];
  page.on('response', async (res) => {
    if (res.url().includes('/api/v1/groups')) {
      groupApiCalls.push({ url: res.url(), status: res.status() });
    }
  });

  await page.waitForTimeout(2000);

  console.log('===== 4. 验证 Group 加载 =====');
  // 通过 API 直接验证 group 接口
  const groupRes = await page.request.get(`${API_URL}/api/v1/groups?page=1&page_size=10`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const groupData = await groupRes.json();
  console.log(`Group API 响应: code=${groupData.code}, groups数量=${groupData.data?.groups?.length || 0}`);

  // 创建测试组
  if (!groupData.data?.groups?.length) {
    console.log('当前无 Group，正在创建测试组用于演示...');
    for (const name of ['运维组', '研发组']) {
      const createRes = await page.request.post(`${API_URL}/api/v1/groups`, {
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        data: { name, description: `用于工作流审批测试的 ${name}` },
      });
      const createData = await createRes.json();
      console.log(`创建组 "${name}": code=${createData.code}`);
    }
  }

  // 刷新页面触发重新加载
  await page.reload({ waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.waitForTimeout(3000);
  // 关闭新工作流模态
  const cancelModal = page.locator('button[aria-label="Cancel"], button:has-text("取消")');
  if (await cancelModal.count() > 0) {
    await cancelModal.first().click().catch(() => {});
  }
  await page.waitForTimeout(1000);

  // 切到流程配置
  const configTab2 = page.locator('div[role="tab"]:has-text("流程配置")');
  if (await configTab2.count() > 0) {
    await configTab2.first().click();
    await page.waitForTimeout(2000);
  }

  await page.screenshot({ path: `${SCREENSHOT_DIR}/04-with-groups.png`, fullPage: true });
  console.log('截图: 04-with-groups.png');

  // 点击审批组选择器
  const approverGroupSelect = page.locator('.ant-select-selector').filter({ hasText: '审批组' });
  if (await approverGroupSelect.count() > 0) {
    await approverGroupSelect.first().click();
    await page.waitForTimeout(1500);
    await page.screenshot({ path: `${SCREENSHOT_DIR}/05-group-dropdown.png`, fullPage: true });
    console.log('截图: 05-group-dropdown.png');
  }

  // ==== 错误汇总 ====
  console.log('\n===== 错误与警告汇总 =====');
  const criticalErrors = errors.filter(e =>
    e.includes('root-0') ||
    e.includes('Cannot read properties of undefined') ||
    e.includes('getLayer') ||
    e.includes('createDiagram')
  );
  console.log(`关键错误 (root-0, getLayer, createDiagram): ${criticalErrors.length}`);
  criticalErrors.forEach(e => console.log(`  ❌ ${e}`));

  const modalWarnings = warnings.filter(w =>
    w.toLowerCase().includes('destroyonclose') ||
    w.toLowerCase().includes('deprecated')
  );
  console.log(`Modal 弃用警告: ${modalWarnings.length}`);
  modalWarnings.forEach(w => console.log(`  ⚠️ ${w}`));

  console.log(`\n所有 page error: ${errors.length}`);
  if (errors.length > 0 && errors.length <= 8) {
    errors.forEach(e => console.log(`  - ${e}`));
  }

  // ==== 结果汇总 ====
  const result = {
    canvasRendered: canvasExists > 0 && svgInCanvas > 0,
    rootElementFound: rootElement > 0,
    approverGroupsFieldFound: approverGroupsLabel > 0,
    criticalRoot0Errors: criticalErrors.length,
    modalDeprecatedWarnings: modalWarnings.length,
    totalErrors: errors.length,
    totalWarnings: warnings.length,
    passedChecks: [
      { name: 'BPMN 画布渲染', passed: canvasExists > 0 && svgInCanvas > 0 },
      { name: '画布根元素 (root-0)', passed: rootElement > 0 },
      { name: '审批组字段出现', passed: approverGroupsLabel > 0 },
      { name: '无 root-0 关键错误', passed: criticalErrors.length === 0 },
      { name: '无 Modal destroyOnClose 警告', passed: modalWarnings.length === 0 },
    ],
  };

  const fs2 = require('fs');
  fs2.writeFileSync(`${SCREENSHOT_DIR}/results.json`, JSON.stringify(result, null, 2));
  console.log(`\n===== 结果汇总 =====`);
  result.passedChecks.forEach(c => console.log(`${c.passed ? '✅' : '❌'} ${c.name}`));

  await browser.close();

  process.exit(result.passedChecks.every(c => c.passed) ? 0 : 1);
})().catch(err => {
  console.error('脚本执行失败:', err);
  process.exit(1);
});
