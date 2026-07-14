/**
 * ITSM Deep Business Testing - Playwright
 * 深入业务测试：登录、访问关键页面、创建工单、检查错误
 */

import type { Page, Browser, ConsoleMessage, Response } from 'playwright';
import { chromium } from 'playwright';
import * as fs from 'fs';
import * as path from 'path';

const SCREENSHOT_DIR = '/tmp/itsm-screenshots';
const REPORT_FILE = '/tmp/itsm-test-report.md';

// 确保截图目录存在
if (!fs.existsSync(SCREENSHOT_DIR)) {
  fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
}

// 存储测试结果
interface TestResult {
  page: string;
  url: string;
  status: 'pass' | 'fail' | 'warning';
  screenshot: string;
  errors: string[];
  warnings: string[];
  apiFailures: string[];
  consoleErrors: string[];
  content: string;
  timestamp: string;
}

const testResults: TestResult[] = [];

/**
 * 执行登录
 */
async function loginAsAdmin(page: Page): Promise<boolean> {
  try {
    console.log('[LOGIN] Starting login process...');
    await page.context().clearCookies();
    await page.evaluate(() => {
      try {
        localStorage.clear();
        document.cookie.split(';').forEach(c => {
          const name = c.replace(/^ +/, '').replace(/=.*/, '');
          if (name) {
            document.cookie = name + '=;expires=' + new Date().toUTCString() + ';path=/';
          }
        });
      } catch (e) {}
    });

    await page.goto('/login');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForSelector('input.ant-input', { timeout: 15000 });

    const inputs = page.locator('input.ant-input');
    await inputs.nth(0).fill('admin');
    await inputs.nth(1).fill('admin123');
    await page.click('button[type="submit"]');

    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    // 检查是否登录成功
    const url = page.url();
    if (url.includes('/login')) {
      console.log('[LOGIN] Failed - still on login page');
      return false;
    }

    console.log('[LOGIN] Success - redirected to:', url);
    return true;
  } catch (error: any) {
    console.log('[LOGIN] Error:', error.message);
    return false;
  }
}

/**
 * 截取页面截图
 */
async function takeScreenshot(page: Page, name: string): Promise<string> {
  const filename = `${name}-${Date.now()}.png`;
  const filepath = path.join(SCREENSHOT_DIR, filename);
  try {
    await page.screenshot({ path: filepath, fullPage: true });
    console.log(`[SCREENSHOT] Saved: ${filename}`);
    return filepath;
  } catch (error: any) {
    console.log(`[SCREENSHOT] Failed: ${error.message}`);
    return '';
  }
}

/**
 * 测试单个页面
 */
async function testPage(
  browser: Browser,
  page: Page,
  pageName: string,
  url: string
): Promise<TestResult> {
  const result: TestResult = {
    page: pageName,
    url: url,
    status: 'pass',
    screenshot: '',
    errors: [],
    warnings: [],
    apiFailures: [],
    consoleErrors: [],
    content: '',
    timestamp: new Date().toISOString(),
  };

  const consoleErrors: string[] = [];
  const apiFailures: string[] = [];

  // 设置监听器
  const errorHandler = (msg: ConsoleMessage) => {
    if (msg.type() === 'error') {
      consoleErrors.push(msg.text());
    }
  };
  const responseHandler = (response: Response) => {
    if (response.status() >= 400) {
      apiFailures.push(`${response.status()} - ${response.url()}`);
    }
  };

  page.on('console', errorHandler);
  page.on('response', responseHandler);

  try {
    console.log(`\n[TEST] Testing: ${pageName} (${url})`);
    await page.goto(url, { timeout: 30000 });
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000); // 等待动态内容加载

    // 截取截图
    result.screenshot = await takeScreenshot(page, pageName.toLowerCase().replace(/\s+/g, '-'));

    // 获取页面标题
    result.content = await page.title();

    // 检查是否有错误内容
    const bodyText = (await page.locator('body').textContent()) || '';
    if (
      bodyText.includes('Error') ||
      bodyText.includes('error') ||
      bodyText.includes('404') ||
      bodyText.includes('500')
    ) {
      result.warnings.push('Page content contains error indicators');
    }

    result.consoleErrors = consoleErrors;
    result.apiFailures = apiFailures;

    if (consoleErrors.length > 0) {
      result.warnings.push(`Console errors: ${consoleErrors.length}`);
      console.log(`[WARNING] Console errors: ${consoleErrors.length}`);
    }

    if (apiFailures.length > 0) {
      result.warnings.push(`API failures: ${apiFailures.length}`);
      console.log(`[WARNING] API failures: ${apiFailures.length}`);
      apiFailures.forEach(f => console.log(`  - ${f}`));
    }

    result.status = result.warnings.length > 0 ? 'warning' : 'pass';
    console.log(`[RESULT] ${pageName}: ${result.status}`);
  } catch (error: any) {
    result.status = 'fail';
    result.errors.push(error.message);
    console.log(`[ERROR] ${pageName}: ${error.message}`);
    await takeScreenshot(page, `error-${pageName.toLowerCase().replace(/\s+/g, '-')}`);
  }

  // 移除监听器
  page.off('console', errorHandler);
  page.off('response', responseHandler);

  return result;
}

/**
 * 测试创建工单流程
 */
async function testCreateTicket(page: Page): Promise<TestResult> {
  const result: TestResult = {
    page: 'Create Ticket',
    url: '/tickets/create',
    status: 'pass',
    screenshot: '',
    errors: [],
    warnings: [],
    apiFailures: [],
    consoleErrors: [],
    content: '',
    timestamp: new Date().toISOString(),
  };

  const consoleErrors: string[] = [];
  const apiFailures: string[] = [];

  page.on('console', (msg: ConsoleMessage) => {
    if (msg.type() === 'error') {
      consoleErrors.push(msg.text());
    }
  });
  page.on('response', (response: Response) => {
    if (response.status() >= 400) {
      apiFailures.push(`${response.status()} - ${response.url()}`);
    }
  });

  try {
    console.log('\n[TEST] Testing: Create Ticket');
    await page.goto('/tickets/create', { timeout: 30000 });
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    result.screenshot = await takeScreenshot(page, 'ticket-create');

    // 检查表单元素是否存在
    const formExists = await page
      .locator('.ant-form')
      .isVisible()
      .catch(() => false);
    if (!formExists) {
      result.warnings.push('Form not visible');
    }

    // 尝试填写表单
    const titleInput = page.locator('input.ant-input').first();
    if (await titleInput.isVisible()) {
      await titleInput.fill('Test Ticket from E2E Testing');
      result.warnings.push('Form is fillable');
    }

    result.consoleErrors = consoleErrors;
    result.apiFailures = apiFailures;

    console.log(`[RESULT] Create Ticket: ${result.status}`);
  } catch (error: any) {
    result.status = 'fail';
    result.errors.push(error.message);
    console.log(`[ERROR] Create Ticket: ${error.message}`);
    await takeScreenshot(page, 'error-ticket-create');
  }

  return result;
}

/**
 * 生成测试报告
 */
function generateReport(results: TestResult[]): string {
  let report = `# ITSM Deep Business Test Report\n\n`;
  report += `Generated: ${new Date().toISOString()}\n\n`;
  report += `## Summary\n\n`;
  report += `- Total Pages Tested: ${results.length}\n`;
  report += `- Passed: ${results.filter(r => r.status === 'pass').length}\n`;
  report += `- Warnings: ${results.filter(r => r.status === 'warning').length}\n`;
  report += `- Failed: ${results.filter(r => r.status === 'fail').length}\n\n`;

  report += `## Detailed Results\n\n`;

  results.forEach(result => {
    report += `### ${result.page}\n`;
    report += `- URL: ${result.url}\n`;
    report += `- Status: ${result.status}\n`;
    if (result.screenshot) {
      report += `- Screenshot: ${result.screenshot}\n`;
    }
    if (result.content) {
      report += `- Page Title: ${result.content}\n`;
    }
    if (result.consoleErrors.length > 0) {
      report += `- Console Errors (${result.consoleErrors.length}):\n`;
      result.consoleErrors.forEach(e => (report += `  - ${e}\n`));
    }
    if (result.apiFailures.length > 0) {
      report += `- API Failures (${result.apiFailures.length}):\n`;
      result.apiFailures.forEach(f => (report += `  - ${f}\n`));
    }
    if (result.warnings.length > 0) {
      report += `- Warnings:\n`;
      result.warnings.forEach(w => (report += `  - ${w}\n`));
    }
    if (result.errors.length > 0) {
      report += `- Errors:\n`;
      result.errors.forEach(e => (report += `  - ${e}\n`));
    }
    report += `\n`;
  });

  // 问题汇总
  report += `## Issues Summary\n\n`;

  const allIssues: { page: string; issue: string; type: string }[] = [];
  results.forEach(r => {
    r.consoleErrors.forEach(e => allIssues.push({ page: r.page, issue: e, type: 'Console Error' }));
    r.apiFailures.forEach(f => allIssues.push({ page: r.page, issue: f, type: 'API Failure' }));
    r.warnings.forEach(w => allIssues.push({ page: r.page, issue: w, type: 'Warning' }));
    r.errors.forEach(e => allIssues.push({ page: r.page, issue: e, type: 'Error' }));
  });

  if (allIssues.length === 0) {
    report += `No issues found.\n`;
  } else {
    report += `Total Issues: ${allIssues.length}\n\n`;
    allIssues.forEach((issue, idx) => {
      report += `${idx + 1}. [${issue.type}] ${issue.page}\n`;
      report += `   - ${issue.issue}\n\n`;
    });
  }

  return report;
}

/**
 * 主测试函数
 */
async function runDeepTests() {
  console.log('=== ITSM Deep Business Testing ===\n');

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });
  const page = await context.newPage();

  // 1. 登录
  const loginSuccess = await loginAsAdmin(page);
  if (!loginSuccess) {
    console.log('[FATAL] Login failed, cannot continue tests');
    await browser.close();
    process.exit(1);
  }

  // 2. 测试关键页面
  const pagesToTest = [
    { name: 'Service Catalog', url: '/service-catalog' },
    { name: 'Tickets', url: '/tickets' },
    { name: 'Incidents', url: '/incidents' },
    { name: 'Problems', url: '/problems' },
    { name: 'Changes', url: '/changes' },
    { name: 'Knowledge Base', url: '/knowledge' },
    { name: 'Workflow', url: '/workflow' },
    { name: 'Dashboard', url: '/dashboard' },
    { name: 'SLAs', url: '/sla' },
    { name: 'Service Requests', url: '/service-requests' },
  ];

  for (const pageInfo of pagesToTest) {
    const result = await testPage(browser, page, pageInfo.name, pageInfo.url);
    testResults.push(result);
  }

  // 3. 测试创建工单
  const ticketResult = await testCreateTicket(page);
  testResults.push(ticketResult);

  // 4. 生成报告
  const report = generateReport(testResults);
  fs.writeFileSync(REPORT_FILE, report);
  console.log(`\n[REPORT] Saved to: ${REPORT_FILE}`);

  // 5. 打印摘要
  console.log('\n=== Test Summary ===');
  console.log(`Total Pages: ${testResults.length}`);
  console.log(`Passed: ${testResults.filter(r => r.status === 'pass').length}`);
  console.log(`Warnings: ${testResults.filter(r => r.status === 'warning').length}`);
  console.log(`Failed: ${testResults.filter(r => r.status === 'fail').length}`);
  console.log(`\nScreenshots saved to: ${SCREENSHOT_DIR}`);

  await browser.close();
}

// 运行测试
runDeepTests().catch(console.error);
