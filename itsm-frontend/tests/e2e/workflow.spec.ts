/**
 * Workflow Management E2E Tests
 * 工作流管理模块测试
 */

import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

test.describe('Workflow Management - 工作流管理', () => {
  test.describe.configure({ timeout: 90_000 });

  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
  });

  async function openWorkflowPage(page: import('@playwright/test').Page, path: string) {
    await page.goto(path, { waitUntil: 'domcontentloaded', timeout: 60_000 });
    await expect(page).toHaveURL(new RegExp(path.replace('/', '\\/')));
    await expect(page.locator('.main-content')).toBeVisible({ timeout: 60_000 });
  }

  test.describe('Workflow List - 工作流列表', () => {
    test('should navigate to workflow page', async ({ page }) => {
      await openWorkflowPage(page, '/workflow');
      await expect(page.getByRole('heading', { name: '工作流管理' })).toBeVisible({
        timeout: 30_000,
      });
      await expect(page.getByRole('button', { name: /新建工作流|创建工作流/ })).toBeVisible({
        timeout: 30_000,
      });
    });

    test('should display workflow list', async ({ page }) => {
      await openWorkflowPage(page, '/workflow');
      await expect(page.getByText('统一管理工作流定义')).toBeVisible({ timeout: 30_000 });
      await expect(page.locator('.ant-table')).toBeVisible({ timeout: 30_000 });
      await expect(page.getByRole('row', { name: /工单|流程|flow/i }).first()).toBeVisible({
        timeout: 30_000,
      });
    });
  });

  test.describe('Workflow Designer - 流程设计器', () => {
    test('should navigate to workflow designer', async ({ page }) => {
      await openWorkflowPage(page, '/workflow/designer');
      await expect(page.getByRole('tab', { name: '流程设计' })).toBeVisible({ timeout: 30_000 });
    });
  });

  test.describe('Workflow Instances - 流程实例', () => {
    test('should navigate to workflow instances', async ({ page }) => {
      await openWorkflowPage(page, '/workflow/instances');
      await expect(page.getByRole('heading', { name: '工作流实例' })).toBeVisible({
        timeout: 30_000,
      });
      await expect(page.getByPlaceholder('按流程 Key 搜索')).toBeVisible({ timeout: 30_000 });
    });
  });

  test.describe('Workflow Dashboard - 工作流仪表盘', () => {
    test('should navigate to workflow dashboard', async ({ page }) => {
      await openWorkflowPage(page, '/workflow/dashboard');
      await expect(page.getByRole('heading', { name: /BPMN流程监控仪表盘|BPMN/ })).toBeVisible({
        timeout: 30_000,
      });
    });
  });

  test.describe('Workflow Audit - 审计日志', () => {
    test('should navigate to workflow audit', async ({ page }) => {
      await openWorkflowPage(page, '/workflow/audit');
      await expect(page.getByRole('heading', { name: /BPMN审计日志|审计日志/ })).toBeVisible({
        timeout: 30_000,
      });
      await expect(page.getByPlaceholder(/流程定义Key|流程定义/)).toBeVisible({ timeout: 30_000 });
    });
  });
});
