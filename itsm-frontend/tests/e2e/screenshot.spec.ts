/**
 * ITSM 自动化截图脚本
 *
 * 使用方法:
 * cd itsm-frontend
 * npx playwright test --config=playwright.screenshot.config.ts
 */

import { test, devices, chromium } from '@playwright/test';
import path from 'path';
import fs from 'fs';

const CONFIG = {
  baseURL: 'http://localhost:3000',
  username: 'admin',
  password: 'admin123',
  outputDir: path.join(__dirname, '..', '..', 'docs', 'images'),
};

const PAGES = [
  { path: '/login', name: 'login', skipLogin: true },
  { path: '/dashboard', name: 'dashboard' },
  { path: '/tickets', name: 'tickets' },
  { path: '/incidents', name: 'incidents' },
  { path: '/problems', name: 'problems' },
  { path: '/changes', name: 'changes' },
  { path: '/knowledge', name: 'knowledge' },
  { path: '/workflow/designer', name: 'workflow-designer' },
  { path: '/workflow/dashboard', name: 'workflow-dashboard' },
  { path: '/sla-monitor', name: 'sla-monitor' },
  { path: '/cmdb', name: 'cmdb' },
  { path: '/assets', name: 'assets' },
  { path: '/msp/management', name: 'msp-management' },
];

// 创建输出目录
if (!fs.existsSync(CONFIG.outputDir)) {
  fs.mkdirSync(CONFIG.outputDir, { recursive: true });
}

test.describe('ITSM Screenshots', () => {
  test('capture login page', async ({ page }) => {
    await page.goto(`${CONFIG.baseURL}/login`);
    await page.waitForLoadState('networkidle');
    await page.screenshot({
      path: path.join(CONFIG.outputDir, 'login.png'),
      fullPage: true
    });
  });

  test('capture dashboard', async ({ page }) => {
    // Login first
    await page.goto(`${CONFIG.baseURL}/login`);
    await page.waitForLoadState('networkidle');
    await page.fill('input[type="text"], input[name="username"]', CONFIG.username);
    await page.fill('input[type="password"]', CONFIG.password);
    await page.click('button[type="submit"]');
    await page.waitForURL('**/dashboard', { timeout: 10000 });

    // Capture dashboard
    await page.goto(`${CONFIG.baseURL}/dashboard`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    await page.screenshot({
      path: path.join(CONFIG.outputDir, 'dashboard.png'),
      fullPage: true
    });
  });
});
