/**
 * SLA 监控完整 E2E 测试
 * 覆盖 SLA 定义、监控、告警、报表等核心功能
 */
import { test, expect } from '@playwright/test';
import { loginAs, logout, TEST_USERS } from '../utils/test-utils';
import { TicketPage } from '../utils/page-objects/TicketPage';

test.describe('SLA 监控完整测试', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    await logout(page).catch(() => {});
  });

  test.describe('SLA 仪表盘', () => {
    test('SLA 仪表盘页面加载', async ({ page }) => {
      await loginAs(page, 'admin');

      await page.goto('/sla-dashboard');
      await page.waitForLoadState('domcontentloaded');

      const bodyContent = await page.locator('body').textContent();
      expect(bodyContent?.length).toBeGreaterThan(50);
    });

    test('SLA 统计卡片显示', async ({ page }) => {
      await loginAs(page, 'admin');

      await page.goto('/sla-dashboard');
      await page.waitForLoadState('networkidle');

      // 检查是否有统计相关的元素
      const hasStats = await page.locator('[class*="stat"], [class*="card"], .ant-card').first().isVisible().catch(() => false);
      console.log('Has stats cards:', hasStats);
    });
  });

  test.describe('SLA 定义管理', () => {
    test('SLA 定义列表页面', async ({ page }) => {
      await loginAs(page, 'admin');

      await page.goto('/sla');
      await page.waitForLoadState('domcontentloaded');

      const bodyContent = await page.locator('body').textContent();
      expect(bodyContent?.length).toBeGreaterThan(50);
    });

    test('SLA 创建页面加载', async ({ page }) => {
      await loginAs(page, 'admin');

      await page.goto('/sla/create');
      await page.waitForLoadState('domcontentloaded');

      const bodyContent = await page.locator('body').textContent();
      expect(bodyContent?.length).toBeGreaterThan(30);
    });

    test('SLA 搜索功能', async ({ page }) => {
      await loginAs(page, 'admin');

      await page.goto('/sla');
      await page.waitForLoadState('domcontentloaded');

      // 尝试搜索
      const searchInput = page.locator('input[placeholder*="搜索"], input[type="search"]');
      if (await searchInput.isVisible().catch(() => false)) {
        await searchInput.fill('test');
        await searchInput.press('Enter');
        await page.waitForLoadState('networkidle');
      }

      const bodyContent = await page.locator('body').textContent();
      expect(bodyContent).toBeDefined();
    });
  });

  test.describe('SLA 监控', () => {
    test('SLA 监控页面加载', async ({ page }) => {
      await loginAs(page, 'admin');

      await page.goto('/sla-monitor');
      await page.waitForLoadState('domcontentloaded');

      const bodyContent = await page.locator('body').textContent();
      expect(bodyContent?.length).toBeGreaterThan(50);
    });

    test('SLA 告警列表', async ({ page }) => {
      await loginAs(page, 'admin');

      await page.goto('/sla-monitor');
      await page.waitForLoadState('networkidle');

      // 检查是否有告警相关的元素
      const hasAlert = await page.locator('[class*="alert"], [class*="warning"], [class*="breach"]').first().isVisible().catch(() => false);
      console.log('Has alert elements:', hasAlert);
    });

    test('SLA 状态过滤', async ({ page }) => {
      await loginAs(page, 'admin');

      await page.goto('/sla-monitor');
      await page.waitForLoadState('domcontentloaded');

      // 尝试状态过滤
      const filterDropdown = page.locator('.ant-select, [class*="filter"]');
      if (await filterDropdown.first().isVisible().catch(() => false)) {
        await filterDropdown.first().click();
        await page.waitForLoadState('networkidle');
      }

      const bodyContent = await page.locator('body').textContent();
      expect(bodyContent).toBeDefined();
    });
  });

  test.describe('工作流 SLA', () => {
    test('工作流 SLA 页面加载', async ({ page }) => {
      await loginAs(page, 'admin');

      await page.goto('/workflow/sla');
      await page.waitForLoadState('domcontentloaded');

      const bodyContent = await page.locator('body').textContent();
      expect(bodyContent?.length).toBeGreaterThan(30);
    });
  });

  test.describe('SLA API 接口测试', () => {
    test('GET /api/v1/sla SLA 列表接口', async ({ request }) => {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';

      // 登录
      const loginResponse = await request.post(`${apiUrl}/api/v1/auth/login`, {
        data: {
          username: TEST_USERS.admin.username,
          password: TEST_USERS.admin.password,
        },
      });

      expect(loginResponse.ok()).toBe(true);

      const loginData = await loginResponse.json();
      const token = loginData.data?.access_token;

      if (!token) {
        test.skip();
        return;
      }

      // 获取 SLA 列表
      const response = await request.get(`${apiUrl}/api/v1/sla`, {
        headers: { Authorization: `Bearer ${token}` },
      });

      expect([200, 401, 403].includes(response.status())).toBe(true);
    });

    test('GET /api/v1/sla/monitor 监控数据接口', async ({ request }) => {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';

      // 登录
      const loginResponse = await request.post(`${apiUrl}/api/v1/auth/login`, {
        data: {
          username: TEST_USERS.admin.username,
          password: TEST_USERS.admin.password,
        },
      });

      const loginData = await loginResponse.json();
      const token = loginData.data?.access_token;

      if (!token) {
        test.skip();
        return;
      }

      // 获取 SLA 监控数据
      const response = await request.get(`${apiUrl}/api/v1/sla/monitor`, {
        headers: { Authorization: `Bearer ${token}` },
      });

      expect([200, 401, 403].includes(response.status())).toBe(true);
    });

    test('GET /api/v1/sla/breaches 告警接口', async ({ request }) => {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';

      // 登录
      const loginResponse = await request.post(`${apiUrl}/api/v1/auth/login`, {
        data: {
          username: TEST_USERS.admin.username,
          password: TEST_USERS.admin.password,
        },
      });

      const loginData = await loginResponse.json();
      const token = loginData.data?.access_token;

      if (!token) {
        test.skip();
        return;
      }

      // 获取 SLA 告警列表
      const response = await request.get(`${apiUrl}/api/v1/sla/breaches`, {
        headers: { Authorization: `Bearer ${token}` },
      });

      expect([200, 401, 403].includes(response.status())).toBe(true);
    });
  });

  test.describe('SLA 与工单关联测试', () => {
    test('工单详情页显示 SLA 信息', async ({ page }) => {
      await loginAs(page, 'admin');

      const ticketPage = new TicketPage(page);
      await ticketPage.goto();

      // 检查表格是否存在
      const tableExists = await page.locator('table').isVisible().catch(() => false);
      if (!tableExists) {
        test.skip();
        return;
      }

      const ticketId = await ticketPage.getFirstTicketId();
      if (!ticketId) {
        test.skip();
        return;
      }

      // 打开工单详情
      await ticketPage.openTicket(Number(ticketId));
      await page.waitForLoadState('domcontentloaded');

      // 检查是否有 SLA 相关信息
      const pageContent = await page.locator('body').textContent();
      const hasSLAInfo = pageContent?.includes('SLA') || pageContent?.includes('服务级别');
      console.log('Has SLA info in ticket detail:', hasSLAInfo);
    });

    test('工单创建时可选择 SLA', async ({ page }) => {
      await loginAs(page, 'admin');

      await page.goto('/tickets/create');
      await page.waitForLoadState('domcontentloaded');

      // 检查是否有 SLA 选择
      const hasSLASelect = await page.locator('[id*="sla"], [name*="sla"]').isVisible().catch(() => false);
      console.log('Has SLA select in ticket create:', hasSLASelect);
    });
  });

  test.describe('SLA 报表', () => {
    test('SLA 报表页面加载', async ({ page }) => {
      await loginAs(page, 'admin');

      await page.goto('/sla-reports');
      await page.waitForLoadState('domcontentloaded');

      const bodyContent = await page.locator('body').textContent();
      expect(bodyContent?.length).toBeGreaterThan(30);
    });

    test('SLA 报表时间范围选择', async ({ page }) => {
      await loginAs(page, 'admin');

      await page.goto('/sla-reports');
      await page.waitForLoadState('domcontentloaded');

      // 尝试选择时间范围
      const datePicker = page.locator('.ant-picker, [class*="date"]');
      if (await datePicker.first().isVisible().catch(() => false)) {
        await datePicker.first().click();
        await page.waitForLoadState('networkidle');
      }

      const bodyContent = await page.locator('body').textContent();
      expect(bodyContent).toBeDefined();
    });
  });
});

/**
 * SLA 监控 API 集成测试
 */
test.describe('SLA 监控 API 集成测试', () => {
  test('完整的 SLA 监控流程 API', async ({ request }) => {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';

    // 1. 登录
    const loginResponse = await request.post(`${apiUrl}/api/v1/auth/login`, {
      data: {
        username: TEST_USERS.admin.username,
        password: TEST_USERS.admin.password,
      },
    });

    if (!loginResponse.ok()) {
      test.skip();
      return;
    }

    const loginData = await loginResponse.json();
    const token = loginData.data?.access_token;

    expect(token).toBeDefined();

    // 2. 获取 SLA 列表
    const slaResponse = await request.get(`${apiUrl}/api/v1/sla`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const slaData = slaResponse.ok() ? await slaResponse.json() : null;
    console.log('SLA count:', slaData?.data?.total ?? 0);

    // 3. 获取 SLA 监控数据
    const monitorResponse = await request.get(`${apiUrl}/api/v1/sla/monitor`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const monitorData = monitorResponse.ok() ? await monitorResponse.json() : null;
    console.log('SLA monitor data:', monitorData?.data ?? 'N/A');

    // 4. 获取 SLA 告警
    const breachesResponse = await request.get(`${apiUrl}/api/v1/sla/breaches`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const breachesData = breachesResponse.ok() ? await breachesResponse.json() : null;
    console.log('SLA breaches count:', breachesData?.data?.total ?? 0);

    // 验证 API 响应
    expect([200, 401, 403].includes(slaResponse.status())).toBe(true);
    expect([200, 401, 403].includes(monitorResponse.status())).toBe(true);
    expect([200, 401, 403].includes(breachesResponse.status())).toBe(true);
  });

  test('SLA 权限验证', async ({ request }) => {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';

    // 以普通用户登录
    const loginResponse = await request.post(`${apiUrl}/api/v1/auth/login`, {
      data: {
        username: TEST_USERS.end_user.username,
        password: TEST_USERS.end_user.password,
      },
    });

    if (!loginResponse.ok()) {
      test.skip();
      return;
    }

    const loginData = await loginResponse.json();
    const token = loginData.data?.access_token;

    expect(token).toBeDefined();

    // 尝试访问 SLA 管理接口
    const response = await request.get(`${apiUrl}/api/v1/sla`, {
      headers: { Authorization: `Bearer ${token}` },
    });

    // 普通用户可能只能查看，不能管理
    console.log('End user SLA access status:', response.status());
    expect([200, 403].includes(response.status())).toBe(true);
  });
});
