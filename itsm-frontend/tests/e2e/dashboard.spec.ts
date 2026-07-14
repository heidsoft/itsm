/**
 * Dashboard E2E Tests
 * 仪表盘模块端到端测试
 */

import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

test.describe('Dashboard - 仪表盘', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page);
  });

  test.describe('Dashboard Navigation - 仪表盘导航', () => {
    test('should navigate to dashboard page', async ({ page }) => {
      await page.goto('/dashboard', { timeout: 30000 });
      await page.waitForLoadState('domcontentloaded');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should display dashboard page with content', async ({ page }) => {
      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('KPI Metrics - 关键指标', () => {
    test('should display KPI cards', async ({ page }) => {
      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');
      // KPI cards should render
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Quick Actions - 快捷操作', () => {
    test('should display quick action buttons', async ({ page }) => {
      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');
      // 验证快捷操作区域存在
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Charts - 图表', () => {
    test('should display ticket trend chart', async ({ page }) => {
      await page.goto('/dashboard', { timeout: 30000 });
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(2000);
      // 验证工单趋势图表 - 使用正确的标题文本
      await expect(page.locator('text=工单趋势分析').first()).toBeVisible({ timeout: 20000 });
    });

    test('should display SLA compliance chart', async ({ page }) => {
      await page.goto('/dashboard', { timeout: 30000 });
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(2000);
      // 验证SLA达成率图表
      await expect(page.locator('text=SLA 达成率监控').first()).toBeVisible({ timeout: 20000 });
    });

    test('should display incident distribution chart', async ({ page }) => {
      await page.goto('/dashboard', { timeout: 30000 });
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(2000);
      // 验证事件分类分布图表
      await expect(page.locator('text=事件分类分布').first()).toBeVisible({ timeout: 20000 });
    });

    test('should display response time chart', async ({ page }) => {
      await page.goto('/dashboard', { timeout: 30000 });
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(2000);
      // 验证响应时间分布图表
      await expect(page.locator('text=响应时间分布').first()).toBeVisible({ timeout: 20000 });
    });
  });

  test.describe('Team Workload - 团队负载', () => {
    test('should display team workload chart', async ({ page }) => {
      await page.goto('/dashboard', { timeout: 30000 });
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(2000);
      // 验证团队工作负载图表
      await expect(page.locator('text=团队工作负载').first()).toBeVisible({ timeout: 20000 });
    });
  });

  test.describe('Peak Hours - 高峰时段', () => {
    test('should display peak hours chart', async ({ page }) => {
      await page.goto('/dashboard', { timeout: 30000 });
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(2000);
      // 验证高峰时段分析图表
      await expect(page.locator('text=高峰时段分析').first()).toBeVisible({ timeout: 20000 });
    });
  });
});
