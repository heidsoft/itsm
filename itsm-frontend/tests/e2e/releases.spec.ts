import { expect, test } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

test.describe('发布管理页面功能', () => {
  test.describe.configure({ timeout: 60_000 });

  test.beforeEach(async ({ page }) => {
    test.slow();
    await loginAndReturn(page, 'admin', 'admin123', '/releases');
  });

  test('发布经理可以创建发布并在刷新后查看记录', async ({ page }) => {
    test.slow();
    await page.getByRole('button', { name: '创建发布' }).click();
    await expect(page).toHaveURL(/\/releases\/new$/);
    await expect(page.getByTestId('release-form')).toBeVisible();

    await page.getByTestId('release-submit-button').click();
    await expect(page.getByText('请输入发布编号')).toBeVisible();
    await expect(page.getByText('请输入发布标题')).toBeVisible();

    const suffix = Date.now();
    const releaseNumber = `REL-E2E-${suffix}`;
    const title = `E2E 发布 ${suffix}`;
    await page.getByTestId('release-number-input').fill(releaseNumber);
    await page.getByTestId('release-title-input').fill(title);

    const createResponse = page.waitForResponse(
      response =>
        response.url().endsWith('/api/v1/releases') &&
        response.request().method() === 'POST'
    );
    await page.getByTestId('release-submit-button').click();
    expect((await createResponse).ok()).toBeTruthy();
    await expect(page).toHaveURL(/\/releases$/);
    await expect(page.getByText(releaseNumber)).toBeVisible({ timeout: 15_000 });

    const reloadResponse = page.waitForResponse(
      response =>
        response.url().includes('/api/v1/releases?') &&
        response.request().method() === 'GET'
    );
    await page.reload({ waitUntil: 'domcontentloaded' });
    expect((await reloadResponse).ok()).toBeTruthy();
    await expect(page.getByText(releaseNumber)).toBeVisible({ timeout: 15_000 });
  });

  test('发布列表的查看与编辑操作进入各自页面', async ({ page }) => {
    test.slow();
    const viewLink = page.getByRole('link', { name: /查看发布/ }).first();
    await expect(viewLink).toBeVisible({ timeout: 15_000 });
    await viewLink.click();
    await expect(page).toHaveURL(/\/releases\/\d+$/);
    await page.goBack();

    const editLink = page.getByRole('link', { name: /编辑发布/ }).first();
    await expect(editLink).toBeVisible();
    await editLink.click();
    await expect(page).toHaveURL(/\/releases\/\d+\/edit$/);
    await expect(page.getByTestId('release-form')).toBeVisible({ timeout: 15_000 });
  });
});
