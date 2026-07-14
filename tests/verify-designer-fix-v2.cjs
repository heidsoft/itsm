// 工作流设计器修复验证脚本 v2
const { chromium } = require('playwright');

const BASE_URL = 'http://localhost:3000';
const API_URL = 'http://localhost:8090';
const SCREENSHOT_DIR = '/tmp/itsm-designer-fix-v2';

async function loginViaApi(page) {
  const res = await page.request.post(`${API_URL}/api/v1/auth/login`, {
    headers: { 'Content-Type': 'application/json' },
    data: { username: 'admin', password: 'admin123' },
  });
  const data = await res.json();
  if (!data.data || !data.data.access_token) throw new Error('登录失败: ' + JSON.stringify(data));
  return data.data.access_token;
}

async function closeAnyOpenModal(page) {
  // 关闭可能打开的模态：按 ESC 键或点击取消按钮
  await page.keyboard.press('Escape').catch(() => {});
  await page.waitForTimeout(500);
  const cancelBtn = page.locator('.ant-modal-wrap:not(.ant-modal-hidden) button:has-text("取消")');
  if (await cancelBtn.count() > 0) {
    await cancelBtn.first().click().catch(() => {});
    await page.waitForTimeout(500);
  }
  // 再按 ESC
  await page.keyboard.press('Escape').catch(() => {});
  await page.waitForTimeout(500);
}

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
  const token = await loginViaApi(page);
  console.log('Token 已获取');

  // 准备测试组数据
  console.log('\n===== 2. 准备测试数据：创建测试审批组 =====');
  const groupRes = await page.request.get(`${API_URL}/api/v1/groups?page=1&page_size=10`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const groupData = await groupRes.json();
  let groupCount = groupData.data?.groups?.length || 0;
  console.log(`当前已有 ${groupCount} 个审批组`);
  if (groupCount === 0) {
    for (const name of ['运维组', '研发组', '财务审批组']) {
      const r = await page.request.post(`${API_URL}/api/v1/groups`, {
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        data: { name, description: `用于工作流审批测试的 ${name}` },
      });
      const rd = await r.json();
      console.log(`  ✓ 创建组 "${name}": code=${rd.code}`);
    }
  }

  // 通过 localStorage 注入登录态
  await page.goto(BASE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.evaluate(({ token }) => {
    localStorage.setItem('access_token', token);
    localStorage.setItem('user', JSON.stringify({ id: 1, username: 'admin', name: '系统管理员', role: 'super_admin' }));
  }, { token });

  console.log('\n===== 3. 打开工作流设计器 =====');
  await page.goto(`${BASE_URL}/workflow/designer`, { waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.waitForTimeout(3000);

  // 新工作流模态应该自动出现
  const modalVisible = await page.locator('text=选择工作流模板').isVisible().catch(() => false);
  console.log(`新工作流模态可见: ${modalVisible}`);

  if (modalVisible) {
    await page.screenshot({ path: `${SCREENSHOT_DIR}/01-new-workflow-modal.png`, fullPage: true });
    // 创建空白流程
    const blankButton = page.locator('button:has-text("创建空白流程")');
    if (await blankButton.count() > 0) {
      await blankButton.first().click();
      console.log('✓ 点击 "创建空白流程"');
    }
  }

  // 等待画布初始化（修复后应在几秒内完成）
  await page.waitForTimeout(6000);
  await page.screenshot({ path: `${SCREENSHOT_DIR}/02-blank-canvas.png`, fullPage: true });
  console.log('截图: 02-blank-canvas.png');

  // === 关键验证：BPMN 画布是否正确渲染 ===
  console.log('\n===== 4. 验证 BPMN 画布渲染（root-0 bug 修复） =====');
  const canvasExists = await page.locator('.djs-container, .bjs-container').count();
  console.log(`BPMN 画布容器数量: ${canvasExists}`);

  const svgInCanvas = await page.locator('.djs-container svg, .bjs-container svg').count();
  console.log(`画布内 SVG 元素数量: ${svgInCanvas}`);

  // 检查 root-0 元素（修复前会缺失）
  const rootElement = await page.locator('[data-element-id="root-0"], .djs-root, .bjs-root').count();
  console.log(`BPMN 画布 root 元素数量: ${rootElement}`);

  // 检查是否至少有一个可见图形（开始事件）
  const startEventCount = await page.locator('.djs-element, .bjs-element').count();
  console.log(`画布内可见元素总数: ${startEventCount}`);

  // 关闭可能存在的模态（确保能切到 Tab）
  await closeAnyOpenModal(page);

  // === 切到流程配置 Tab ===
  console.log('\n===== 5. 切到流程配置 Tab（验证审批组） =====');
  await page.waitForTimeout(1000);
  const configTab = page.locator('div[role="tab"]:has-text("流程配置")');
  if (await configTab.count() > 0) {
    await configTab.first().click({ force: true });
    console.log('✓ 切换到流程配置 Tab');
    await page.waitForTimeout(2000);
  }
  await page.screenshot({ path: `${SCREENSHOT_DIR}/03-config-tab.png`, fullPage: true });

  // 验证 "审批组" 字段
  const approverGroupsLabel = await page.locator('text=审批组').count();
  console.log(`"审批组" 标签出现次数: ${approverGroupsLabel}`);

  // 找带 Users 图标的审批组选择器
  const groupSelectTrigger = page.locator('.ant-select').filter({ has: page.locator('.anticon-users, svg.lucide-users') });
  const groupSelectCount = await groupSelectTrigger.count();
  console.log(`带 Users 图标的 Select 组件: ${groupSelectCount}`);

  // 找占位符为 "选择审批组" 的 Select
  const groupSelectByPlaceholder = page.locator('.ant-select').filter({ has: page.locator('input[placeholder*="审批组"]') });
  const placeholderCount = await groupSelectByPlaceholder.count();
  console.log(`占位符含 "审批组" 的 Select: ${placeholderCount}`);

  // 点击审批组 Select 打开下拉
  if (placeholderCount > 0) {
    await groupSelectByPlaceholder.first().click();
    await page.waitForTimeout(1500);
    await page.screenshot({ path: `${SCREENSHOT_DIR}/04-group-dropdown.png`, fullPage: true });

    const dropdownItems = await page.locator('.ant-select-item').count();
    console.log(`下拉项数量: ${dropdownItems}`);

    // 关闭下拉
    await page.keyboard.press('Escape');
    await page.waitForTimeout(500);
  }

  // === 错误汇总 ===
  console.log('\n===== 错误与警告汇总 =====');
  const criticalErrors = errors.filter(e =>
    e.includes('root-0') ||
    e.includes('Cannot read properties of undefined') ||
    e.includes('getLayer') ||
    e.includes('createDiagram') ||
    e.includes('diagram-js')
  );
  console.log(`\n关键 BPMN 错误: ${criticalErrors.length}`);
  criticalErrors.slice(0, 5).forEach(e => console.log(`  ❌ ${e}`));

  const modalWarnings = warnings.filter(w =>
    w.toLowerCase().includes('destroyonclose') ||
    w.toLowerCase().includes('deprecated') ||
    w.includes('Modal.destroyOnClose')
  );
  console.log(`\nModal destroyOnClose 警告: ${modalWarnings.length}`);
  modalWarnings.slice(0, 5).forEach(w => console.log(`  ⚠️ ${w}`));

  console.log(`\n所有 page error: ${errors.length}`);
  if (errors.length > 0 && errors.length <= 10) {
    errors.forEach(e => console.log(`  - ${e}`));
  }

  // === 结果 ===
  const result = {
    canvasRendered: canvasExists > 0 && svgInCanvas > 0,
    rootElementFound: rootElement > 0,
    canvasHasElements: startEventCount > 0,
    approverGroupsFieldFound: approverGroupsLabel > 0 || placeholderCount > 0,
    criticalBpmnErrors: criticalErrors.length,
    modalDeprecatedWarnings: modalWarnings.length,
    totalErrors: errors.length,
    passedChecks: [
      { name: 'BPMN 画布 SVG 渲染', passed: canvasExists > 0 && svgInCanvas > 0 },
      { name: 'BPMN 画布 root 元素', passed: rootElement > 0 },
      { name: '画布含 BPMN 元素（开始事件等）', passed: startEventCount > 0 },
      { name: '审批组字段出现', passed: approverGroupsLabel > 0 || placeholderCount > 0 },
      { name: '无 root-0 / getLayer 关键错误', passed: criticalErrors.length === 0 },
      { name: '无 Modal destroyOnClose 警告', passed: modalWarnings.length === 0 },
    ],
  };

  fs.writeFileSync(`${SCREENSHOT_DIR}/results.json`, JSON.stringify(result, null, 2));
  console.log(`\n===== 检查结果汇总 =====`);
  result.passedChecks.forEach(c => console.log(`${c.passed ? '✅' : '❌'} ${c.name}`));
  const allPassed = result.passedChecks.every(c => c.passed);
  console.log(`\n总体: ${allPassed ? '✅ 全部通过' : '❌ 有未通过项'}`);

  await browser.close();
  process.exit(allPassed ? 0 : 1);
})().catch(err => {
  console.error('脚本执行失败:', err);
  process.exit(1);
});
