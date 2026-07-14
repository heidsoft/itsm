import { test, expect } from '@playwright/test';

test.describe('Register + Login Flow - 注册登录流程', () => {
  test('should register a new user and then login', async ({ page }) => {
    const suffix = `${Date.now()}-${Math.floor(Math.random() * 1000)}`;
    const username = `e2e_user_${suffix}`;
    const email = `e2e_${suffix}@example.com`;
    const password = `E2e!Pass${suffix}`;

    await page.goto('/register');
    await page.waitForLoadState('domcontentloaded');

    const inputs = page.locator('input.ant-input');
    await expect(inputs.first()).toBeVisible({ timeout: 15000 });

    await inputs.nth(0).fill(username);
    await inputs.nth(1).fill(email);
    await page.locator('input[type="password"]').nth(0).fill(password);
    await page.locator('input[type="password"]').nth(1).fill(password);

    await page.getByRole('button', { name: /下一步/i }).click();
    await page.waitForLoadState('domcontentloaded');

    const step2Inputs = page.locator('input.ant-input');
    await expect(step2Inputs.first()).toBeVisible({ timeout: 15000 });

    await step2Inputs.nth(0).fill(`E2E User ${suffix}`);
    await step2Inputs.nth(1).fill(`138${suffix.slice(0, 8).padEnd(8, '0')}`);
    await step2Inputs.nth(2).fill('E2E Company');

    const roleSelect = page.locator('.ant-select-selector').first();
    await roleSelect.click();
    await page.getByRole('option', { name: /普通用户/i }).click();

    const registerResponsePromise = page
      .waitForResponse(
        resp => resp.url().includes('/api/v1/auth/register') && resp.request().method() === 'POST',
        { timeout: 30000 }
      )
      .catch(() => null);

    await page.getByRole('button', { name: /注册|Register/i }).click();

    const registerResponse = await registerResponsePromise;
    if (registerResponse) {
      expect(registerResponse.status()).toBeGreaterThanOrEqual(200);
      expect(registerResponse.status()).toBeLessThan(500);
    }

    await page.waitForLoadState('networkidle');

    const failedAlert = page.locator('.ant-alert-error, .ant-message-error, text=注册失败');
    if (await failedAlert.count()) {
      const details = (await failedAlert.first().textContent()) || '注册失败';
      throw new Error(details);
    }

    await expect(page).toHaveURL(/\/login/, { timeout: 30000 });

    await page.waitForSelector('input.ant-input', { timeout: 15000 });
    const loginInputs = page.locator('input.ant-input');
    await loginInputs.nth(0).fill(username);
    await loginInputs.nth(1).fill(password);

    const loginResponsePromise = page
      .waitForResponse(
        resp => resp.url().includes('/api/v1/auth/login') && resp.request().method() === 'POST',
        { timeout: 30000 }
      )
      .catch(() => null);

    await page.click('button[type="submit"]');

    const loginResponse = await loginResponsePromise;
    if (loginResponse) {
      expect(loginResponse.status()).toBeGreaterThanOrEqual(200);
      expect(loginResponse.status()).toBeLessThan(500);
    }

    await page.waitForURL(/\/(dashboard|tickets|incidents|problems|changes|service-catalog)/, {
      timeout: 30000,
    });
  });
});
