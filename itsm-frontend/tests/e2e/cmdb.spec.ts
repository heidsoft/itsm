import { test, expect } from '@playwright/test';

const ciItem = {
  id: 1,
  name: '应用服务器-01',
  description: '测试资产',
  type: 'server',
  status: 'active',
  ci_type_id: 1,
  tenant_id: 1,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

const ciTypes = [
  {
    id: 1,
    name: '服务器',
    description: '',
    is_active: true,
    tenant_id: 1,
  },
];

const corsHeaders = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Methods': 'GET,POST,PUT,DELETE,PATCH,OPTIONS',
  'Access-Control-Allow-Headers': '*',
};

const fulfillPreflight = async (route: any) => {
  await route.fulfill({
    status: 204,
    headers: corsHeaders,
  });
};

test.beforeEach(async ({ context, baseURL, page }) => {
  if (baseURL) {
    await context.addCookies([
      {
        name: 'auth-token',
        value: 'test-token',
        url: baseURL,
      },
    ]);
  }
  await page.addInitScript(token => {
    window.localStorage.setItem('access_token', token);
    window.localStorage.setItem(
      'auth-storage',
      JSON.stringify({
        state: {
          user: {
            id: 1,
            username: 'test-user',
            email: 'test@example.com',
            name: 'Test User',
            role: 'admin',
          },
          token,
          currentTenant: null,
          isAuthenticated: true,
        },
        version: 0,
      })
    );
  }, 'mock_test_token');

  await page.setExtraHTTPHeaders({ Origin: 'http://localhost:3000' });

  await context.route('**/api/v1/cmdb/types', async (route) => {
    if (route.request().method() === 'OPTIONS') {
      await fulfillPreflight(route);
      return;
    }
    await route.fulfill({
      status: 200,
      headers: corsHeaders,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, message: 'ok', data: ciTypes }),
    });
  });

  await context.route('**/api/v1/cmdb/cloud-services', async (route) => {
    if (route.request().method() === 'OPTIONS') {
      await fulfillPreflight(route);
      return;
    }
    await route.fulfill({
      status: 200,
      headers: corsHeaders,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, message: 'ok', data: [] }),
    });
  });

  await context.route('**/api/v1/cmdb/cloud-resources', async (route) => {
    if (route.request().method() === 'OPTIONS') {
      await fulfillPreflight(route);
      return;
    }
    await route.fulfill({
      status: 200,
      headers: corsHeaders,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, message: 'ok', data: [] }),
    });
  });

});

test('cmdb list renders and search triggers request', async ({ page, context }) => {
  let lastSearch = '';
  await context.route('**/api/v1/cmdb/cis**', async (route) => {
    const requestUrl = new URL(route.request().url());
    if (route.request().method() === 'OPTIONS') {
      await fulfillPreflight(route);
      return;
    }
    if (!/\/cmdb\/cis\/?$/.test(requestUrl.pathname)) {
      await route.fallback();
      return;
    }
    lastSearch = requestUrl.searchParams.get('search') || '';
    await route.fulfill({
      status: 200,
      headers: corsHeaders,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        message: 'ok',
        data: { items: [ciItem], total: 1, page: 1, size: 10 },
      }),
    });
  });

  await page.goto('/cmdb');
  await expect(page.getByRole('button', { name: /录入资产/ })).toBeVisible();
  await expect(page.getByRole('button', { name: '应用服务器-01' })).toBeVisible();

  await page.getByPlaceholder('搜索名称/序列号').fill('db');
  await page.getByRole('button', { name: '查询' }).click();

  await expect.poll(() => lastSearch).toBe('db');
});

test('cmdb delete flow sends request', async ({ page, context }) => {
  let deleteCalled = false;
  await context.route('**/api/v1/cmdb/cis**', async (route) => {
    const requestUrl = new URL(route.request().url());
    if (route.request().method() === 'OPTIONS') {
      await fulfillPreflight(route);
      return;
    }
    if (/\/cmdb\/cis\/1\/?$/.test(requestUrl.pathname)) {
      if (route.request().method() === 'DELETE') {
        deleteCalled = true;
        await route.fulfill({
          status: 200,
          headers: corsHeaders,
          contentType: 'application/json',
          body: JSON.stringify({ code: 0, message: 'ok', data: null }),
        });
        return;
      }
      await route.fulfill({
        status: 200,
        headers: corsHeaders,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, message: 'ok', data: ciItem }),
      });
      return;
    }
    if (/\/cmdb\/cis\/?$/.test(requestUrl.pathname)) {
      await route.fulfill({
        status: 200,
        headers: corsHeaders,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          message: 'ok',
          data: { items: [ciItem], total: 1, page: 1, size: 10 },
        }),
      });
      return;
    }
    await route.fallback();
  });

  await page.goto('/cmdb');
  await page.getByRole('button', { name: '删除' }).click();
  await page.getByRole('button', { name: /确定|OK/ }).click();

  await expect.poll(() => deleteCalled).toBe(true);
});

test('cmdb create form validation and submit', async ({ page, context }) => {
  let createCalled = false;
  await context.route('**/api/v1/cmdb/cis**', async (route) => {
    const requestUrl = new URL(route.request().url());
    if (route.request().method() === 'OPTIONS') {
      await fulfillPreflight(route);
      return;
    }
    if (!/\/cmdb\/cis\/?$/.test(requestUrl.pathname)) {
      await route.fallback();
      return;
    }
    if (route.request().method() === 'POST') {
      createCalled = true;
      await route.fulfill({
        status: 200,
        headers: corsHeaders,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, message: 'ok', data: ciItem }),
      });
      return;
    }
    await route.fallback();
  });

  await page.goto('/cmdb/cis/create');
  await page.getByRole('button', { name: '保存' }).click();

  await expect(page.getByText('请输入资产名称')).toBeVisible();
  await expect(page.getByText('请选择资产类型')).toBeVisible();
  await expect(page.getByText('请选择资产状态')).toBeVisible();

  await page.getByPlaceholder('请输入资产名称').fill('新资产');
  await page.getByText('请选择资产类型').click();
  await page.getByRole('option', { name: '服务器' }).click();
  await page.getByText('请选择资产状态').click();
  await page.getByRole('option', { name: '使用中' }).click();
  await page.getByRole('button', { name: '保存' }).click();

  await expect.poll(() => createCalled).toBe(true);
});
