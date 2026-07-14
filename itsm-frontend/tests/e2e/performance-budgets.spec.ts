import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

function getNumberEnv(name: string, fallback: number) {
  const raw = process.env[name];
  if (!raw) return fallback;
  const parsed = Number(raw);
  return Number.isFinite(parsed) ? parsed : fallback;
}

test.describe('Performance - 浏览器端关键指标', () => {
  test.skip(!process.env.PERF_TESTS, 'PERF_TESTS is not enabled');

  test('First screen navigation should meet budget', async ({ page }) => {
    const budgetMs = getNumberEnv('PERF_BUDGET_FIRST_SCREEN_MS', 2000);

    await page.goto('/login');
    await page.waitForSelector('input.ant-input', { timeout: 15000 });

    const inputs = page.locator('input.ant-input');
    await inputs.nth(0).fill('admin');
    await inputs.nth(1).fill('admin123');

    const start = Date.now();
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(dashboard|tickets|incidents|problems|changes)/, { timeout: 30000 });
    await page.waitForLoadState('networkidle');
    const elapsed = Date.now() - start;

    expect(elapsed).toBeLessThanOrEqual(budgetMs);
  });

  test('Ticket submit response time should meet budget', async ({ page }) => {
    const budgetMs = getNumberEnv('PERF_BUDGET_TICKET_SUBMIT_MS', 1000);

    await loginAndReturn(page, 'end_user', 'admin123');
    await page.goto('/tickets/create');
    await page.waitForSelector('form, [data-testid="ticket-form"]', { timeout: 15000 });

    await page
      .locator('input[id*="title"], input[name*="title"], input[placeholder*="标题"]')
      .first()
      .fill(`Perf Ticket ${Date.now()}`);
    await page
      .locator('textarea[id*="description"], textarea[name*="description"], textarea[placeholder*="描述"]')
      .first()
      .fill('performance test');

    const start = Date.now();
    const respPromise = page
      .waitForResponse(
        r => r.url().includes('/api/v1/tickets') && r.request().method() === 'POST',
        { timeout: 15000 }
      )
      .catch(() => null);

    await page
      .locator(
        'button[type="submit"], button:has-text("提交"), button:has-text("创建"), button:has-text("创建工单")'
      )
      .first()
      .click();

    const response = await respPromise;
    const elapsed = Date.now() - start;

    if (response) {
      expect(response.status()).toBeGreaterThanOrEqual(200);
      expect(response.status()).toBeLessThan(500);
    }

    expect(elapsed).toBeLessThanOrEqual(budgetMs);
  });

  test('Batch export 1000 tickets should meet budget (requires PERF_EXPORT_PAYLOAD_JSON)', async ({ page }) => {
    const raw = process.env.PERF_EXPORT_PAYLOAD_JSON;
    test.skip(!raw, 'PERF_EXPORT_PAYLOAD_JSON is not set');

    const budgetMs = getNumberEnv('PERF_BUDGET_EXPORT_1000_MS', 10000);
    const payload = JSON.parse(raw as string) as Record<string, unknown>;

    await loginAndReturn(page, 'admin', 'admin123');

    const start = Date.now();
    const response = await page.request.post('/api/v1/tickets/batch/export', { data: payload });
    const elapsed = Date.now() - start;

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(500);
    expect(elapsed).toBeLessThanOrEqual(budgetMs);
  });
});
