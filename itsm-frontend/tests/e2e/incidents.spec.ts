import { expect, test } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

test.setTimeout(60_000);

test.describe('事件管理页面功能', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page, 'admin', 'admin123', '/incidents');
  });

  test('管理员可以查看统计、刷新列表并进入详情', async ({ page }) => {
    await expect(page.getByRole('heading', { name: '事件管理' })).toBeVisible({ timeout: 15_000 });

    const statsResponse = await page.request.get(
      'http://localhost:8090/api/v1/incidents/stats'
    );
    expect(statsResponse.ok()).toBeTruthy();
    await expect(page.getByText('总事件数', { exact: true })).toBeVisible();

    await page.getByRole('button', { name: '刷新', exact: true }).click();
    await expect(page.getByRole('heading', { name: '事件管理' })).toBeVisible({
      timeout: 15_000,
    });

    const detailLink = page.getByRole('link', { name: /查看事件/ }).first();
    if (await detailLink.isVisible().catch(() => false)) {
      await detailLink.click();
      await expect(page).toHaveURL(/\/incidents\/\d+$/);
    }
  });

  test('创建事件会校验必填项并在成功后返回列表', async ({ page }) => {
    await page.getByRole('button', { name: '新建事件' }).click();
    await expect(page).toHaveURL(/\/incidents\/create$/);
    await expect(page.getByTestId('incident-create-form')).toBeVisible();

    await page.getByTestId('incident-submit-button').click();
    await expect(page.getByText('请输入事件标题')).toBeVisible();
    await expect(page.getByText('请输入事件描述')).toBeVisible();

    const title = `E2E 事件 ${Date.now()}`;
    await page.getByTestId('incident-title-input').fill(title);
    await page.getByTestId('incident-description-input').fill('用于验证事件创建、跳转和列表持久化。');

    const createResponse = page.waitForResponse(
      response =>
        response.url().endsWith('/api/v1/incidents') &&
        response.request().method() === 'POST'
    );
    await page.getByTestId('incident-submit-button').click();
    expect((await createResponse).ok()).toBeTruthy();
    await expect(page).toHaveURL(/\/incidents$/);
    await expect(page.getByText(title)).toBeVisible();
  });
});
