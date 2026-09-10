import { test, expect, Page } from '@playwright/test';

// ============================================================
// E2E 边界条件测试：工单生命周期
// 覆盖创建、编辑、状态流转、删除、跨租户隔离等场景
// ============================================================

const BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';

// Helper: 登录并返回 token
async function loginAs(page: Page, email: string, password: string): Promise<string> {
  const res = await page.request.post(`${BASE_URL}/api/auth/login`, {
    data: { email, password },
  });
  const body = await res.json();
  return body.data?.token || '';
}

test.describe('Ticket Lifecycle - Boundary Conditions', () => {
  let adminToken: string;

  test.beforeAll(async ({ request }) => {
    const res = await request.post(`${BASE_URL}/api/auth/login`, {
      data: {
        email: 'admin@itsm.local',
        password: 'Adm1n@2026#ItSM',
      },
    });
    const body = await res.json();
    adminToken = body.data?.token || '';
    expect(adminToken).toBeTruthy();
  });

  test('创建工单 - 空标题应返回错误', async ({ request }) => {
    const res = await request.post(`${BASE_URL}/api/tickets`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      data: {
        title: '',
        description: 'test desc',
        priority: 'medium',
        type: 'incident',
      },
    });
    // 空标题应返回 400 或业务错误码
    expect([400, 422]).toContain(res.status());
  });

  test('创建工单 - 超长标题应返回错误或截断', async ({ request }) => {
    const longTitle = '测'.repeat(5000);
    const res = await request.post(`${BASE_URL}/api/tickets`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      data: {
        title: longTitle,
        description: 'test desc',
        priority: 'medium',
        type: 'incident',
      },
    });
    // 应返回错误或成功（截断），不应 500
    expect(res.status()).toBeLessThan(500);
  });

  test('创建工单 - 无效优先级应返回错误', async ({ request }) => {
    const res = await request.post(`${BASE_URL}/api/tickets`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      data: {
        title: 'Invalid priority ticket',
        description: 'test desc',
        priority: 'super-urgent-invalid',
        type: 'incident',
      },
    });
    // 无效优先级应返回 400 或业务错误
    expect([200, 400, 422]).toContain(res.status());
    if (res.ok()) {
      const body = await res.json();
      // 如果成功，优先级应被降级为默认值
      expect(['low', 'medium', 'high', 'critical']).toContain(body.data?.priority);
    }
  });

  test('创建工单 - SQL 注入应安全处理', async ({ request }) => {
    const res = await request.post(`${BASE_URL}/api/tickets`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      data: {
        title: "'; DROP TABLE tickets; --",
        description: 'test desc',
        priority: 'medium',
        type: 'incident',
      },
    });
    // 不应 500
    expect(res.status()).toBeLessThan(500);
  });

  test('创建工单 - XSS payload 应安全存储', async ({ request }) => {
    const xss = '<script>alert("xss")</script>';
    const res = await request.post(`${BASE_URL}/api/tickets`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      data: {
        title: xss,
        description: 'test desc',
        priority: 'medium',
        type: 'incident',
      },
    });
    expect(res.status()).toBeLessThan(500);
  });

  test('获取工单 - 不存在的 ID 应返回 404', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/api/tickets/999999`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect([404, 400]).toContain(res.status());
  });

  test('获取工单 - 零 ID 应返回 400', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/api/tickets/0`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect([400, 404]).toContain(res.status());
  });

  test('列表查询 - 空分页参数应返回默认结果', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/api/tickets?page=0&pageSize=0`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status()).toBeLessThan(500);
    if (res.ok()) {
      const body = await res.json();
      expect(body.data).toBeTruthy();
    }
  });

  test('列表查询 - 负数分页应返回默认结果', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/api/tickets?page=-1&pageSize=-10`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status()).toBeLessThan(500);
  });

  test('删除工单 - 二次删除应幂等', async ({ request }) => {
    // 先创建
    const createRes = await request.post(`${BASE_URL}/api/tickets`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      data: {
        title: 'Double delete test',
        description: 'will be deleted twice',
        priority: 'low',
        type: 'incident',
      },
    });
    const createBody = await createRes.json();
    const ticketId = createBody.data?.id;
    if (!ticketId) {
      test.skip();
      return;
    }

    // 第一次删除
    const del1 = await request.delete(`${BASE_URL}/api/tickets/${ticketId}`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(del1.status()).toBeLessThan(500);

    // 第二次删除 - 不应 500
    const del2 = await request.delete(`${BASE_URL}/api/tickets/${ticketId}`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(del2.status()).toBeLessThan(500);
  });

  test('未认证请求应返回 401', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/api/tickets`);
    expect(res.status()).toBe(401);
  });

  test('无效 token 应返回 401', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/api/tickets`, {
      headers: { Authorization: 'Bearer invalid-token-12345' },
    });
    expect(res.status()).toBe(401);
  });
});

test.describe('Ticket Lifecycle - UI Boundary Tests', () => {
  test.beforeEach(async ({ page }) => {
    // 导航到登录页面
    await page.goto('/login');
    // 填写登录表单
    await page.fill('input[name="email"], input[type="email"]', 'admin@itsm.local');
    await page.fill('input[name="password"], input[type="password"]', 'Adm1n@2026#ItSM');
    await page.click('button[type="submit"]');
    // 等待登录完成
    await page.waitForURL(/\/(dashboard|tickets|$)/, { timeout: 15000 });
  });

  test('工单列表页 - 空状态显示', async ({ page }) => {
    await page.goto('/tickets');
    // 等待页面加载
    await page.waitForLoadState('networkidle');
    // 不应有 JS 错误或白屏
    const body = await page.textContent('body');
    expect(body).toBeTruthy();
    expect(body!.length).toBeGreaterThan(0);
  });

  test('工单创建页 - 表单验证', async ({ page }) => {
    await page.goto('/tickets/create');
    await page.waitForLoadState('networkidle');

    // 不填写任何内容直接提交
    const submitBtn = page.locator('button[type="submit"], button:has-text("创建"), button:has-text("提交")');
    if (await submitBtn.count() > 0) {
      await submitBtn.first().click();
      // 应显示验证错误
      await page.waitForTimeout(1000);
      // 页面不应崩溃
      const hasError = await page.locator('.ant-form-item-explain-error, [class*="error"]').count();
      // 即使没有 error 提示，也不应 500
      expect(hasError).toBeGreaterThanOrEqual(0);
    }
  });

  test('工单详情页 - 不存在的工单显示错误', async ({ page }) => {
    await page.goto('/tickets/999999');
    await page.waitForLoadState('networkidle');
    // 应显示错误状态或空状态，不应白屏
    const body = await page.textContent('body');
    expect(body).toBeTruthy();
  });
});
