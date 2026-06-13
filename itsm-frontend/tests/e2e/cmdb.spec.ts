/**
 * CMDB E2E Tests
 * 配置管理数据库模块业务测试
 */

import { test, expect, Page } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

test.describe('CMDB - 配置管理', () => {
  test.describe.configure({ timeout: 120_000 });

  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
  });

  async function openPage(page: Page, path: string) {
    await page.goto(path, { waitUntil: 'domcontentloaded', timeout: 60_000 });
    await expect(page).toHaveURL(new RegExp(path.replace('/', '\\/')));
    await expect(page.locator('.main-content')).toBeVisible({ timeout: 60_000 });
  }

  async function selectFormOption(page: Page, label: string, option: string) {
    await page.getByRole('combobox', { name: new RegExp(`^(\\*\\s*)?${label}$`) }).click();
    const optionLocator = page.locator('.ant-select-item-option').filter({ hasText: option });
    await expect(optionLocator.last()).toBeVisible({ timeout: 10_000 });
    await optionLocator.last().click();
  }

  test('should show CMDB business overview and primary operations', async ({ page }) => {
    await openPage(page, '/cmdb');

    await expect(page.getByRole('heading', { name: '配置管理数据库 (CMDB)' })).toBeVisible();
    await expect(page.getByText('管理配置项、云资源同步、关系拓扑和核对结果。')).toBeVisible();
    await expect(page.getByRole('button', { name: '新增配置项' })).toBeVisible();
    await expect(page.getByRole('button', { name: '同步云资源' })).toBeVisible();
    await expect(page.getByText('配置项总数')).toBeVisible();
    await expect(page.getByText('云资源同步状态')).toBeVisible();
    await expect(page.getByRole('tab', { name: /配置项/ })).toBeVisible();
    await expect(page.getByRole('tab', { name: /类型分布/ })).toBeVisible();
    await expect(page.getByRole('tab', { name: /云资源/ })).toBeVisible();
    await expect(page.getByRole('tab', { name: /核对/ })).toBeVisible();
  });

  test('should create, search, and filter a configuration item like an operator', async ({ page }) => {
    const unique = Date.now();
    const ciName = `E2E-CMDB-核心数据库-${unique}`;
    const serialNumber = `SN-CMDB-${unique}`;

    await openPage(page, '/cmdb/cis/create');
    await expect(page.getByRole('heading', { name: '录入配置项' })).toBeVisible();

    await page.getByPlaceholder('请输入资产名称').fill(ciName);
    await selectFormOption(page, '资产类型', 'database');
    await selectFormOption(page, '状态', '使用中');
    await selectFormOption(page, '环境', '生产');
    await selectFormOption(page, '重要性', '关键');
    await page.getByPlaceholder('请输入序列号（可选）').fill(serialNumber);
    await page.getByPlaceholder('请输入型号（可选）').fill('PostgreSQL-HA');
    await page.getByPlaceholder('请输入厂商（可选）').fill('PostgreSQL');
    await page.getByPlaceholder('请输入位置（可选）').fill('上海-生产机房-A01');
    await page.getByPlaceholder('请输入资产标签（可选）').fill(`DB-${unique}`);
    await page.getByPlaceholder('请输入拥有者（可选）').fill('DBA团队');
    await selectFormOption(page, '数据来源', '手工录入');
    await page.getByPlaceholder('请输入扩展属性 JSON（可选）').fill('{"ha":true,"owner":"DBA"}');

    await page.getByRole('button', { name: '保存配置项' }).click();
    await expect(page).toHaveURL(/\/cmdb/, { timeout: 60_000 });
    await expect(page.getByRole('heading', { name: '配置管理数据库 (CMDB)' })).toBeVisible({
      timeout: 60_000,
    });

    await page.getByPlaceholder('搜索名称/序列号').fill(serialNumber);
    await page.getByRole('button', { name: /查\s*询/ }).click();
    await expect(page.getByRole('cell', { name: ciName })).toBeVisible({ timeout: 30_000 });
    await expect(page.getByText('PostgreSQL-HA / PostgreSQL')).toBeVisible();

    await selectFormOption(page, '资产类型', 'database');
    await page.getByRole('button', { name: /查\s*询/ }).click();
    await expect(page.getByRole('cell', { name: ciName })).toBeVisible({ timeout: 30_000 });
  });

  test('should expose CI type management page with business controls', async ({ page }) => {
    await openPage(page, '/admin/cmdb-types');

    await expect(page.getByRole('heading', { name: /CI类型|配置项类型|CMDB/ })).toBeVisible({
      timeout: 30_000,
    });
    await expect(page.getByRole('button', { name: /新建|新增|创建/ })).toBeVisible({
      timeout: 30_000,
    });
    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 30_000 });
  });

  test('should expose cloud resources and reconciliation operations', async ({ page }) => {
    await openPage(page, '/cmdb/cloud-resources');
    await expect(page.getByRole('heading', { name: /云资源/ })).toBeVisible({ timeout: 30_000 });
    await expect(page.getByText('查看已发现的云资源，并将资源新建或绑定为 CMDB 配置项。')).toBeVisible({
      timeout: 30_000,
    });

    await openPage(page, '/cmdb/reconciliation');
    await expect(page.getByRole('heading', { name: /云资源核对|核对|对账/ })).toBeVisible({
      timeout: 30_000,
    });
    await expect(page.getByText('孤儿配置项（引用资源不存在）')).toBeVisible({ timeout: 30_000 });
    await expect(page.getByText('未关联配置项（有云资源ID但未绑定）')).toBeVisible({
      timeout: 30_000,
    });
  });
});
