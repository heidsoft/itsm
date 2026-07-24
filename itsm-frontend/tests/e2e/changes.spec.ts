import { expect, test } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

test.setTimeout(60_000);

test.describe('变更管理页面功能', () => {
  test.beforeEach(async ({ page }) => {
    await loginAndReturn(page, 'admin', 'admin123', '/changes');
  });

  test('管理员可以从列表进入变更详情', async ({ page }) => {
    await expect(page.getByRole('heading', { name: '变更管理' })).toBeVisible({ timeout: 15_000 });
    const detailLink = page.getByRole('link', { name: /查看变更/ }).first();
    if (await detailLink.isVisible().catch(() => false)) {
      await detailLink.click();
      await expect(page).toHaveURL(/\/changes\/\d+$/);
    }
  });

  test('新建变更校验 ITIL 必填信息并持久化到列表', async ({ page }) => {
    await page.getByRole('button', { name: '新建变更' }).click();
    await expect(page).toHaveURL(/\/changes\/new$/);
    await expect(page.getByTestId('change-create-form')).toBeVisible();

    await page.getByTestId('change-submit-button').click();
    await expect(page.getByText('请输入变更标题')).toBeVisible();
    await expect(page.getByText('请填写变更理由')).toBeVisible();
    await expect(page.getByText('请填写回滚计划')).toBeVisible();

    const title = `E2E 变更 ${Date.now()}`;
    await page.getByTestId('change-title-input').fill(title);
    await page.getByTestId('change-description-input').fill('验证变更创建及详情入口。');
    await page.getByTestId('change-justification-input').fill('用于页面功能回归测试。');
    await page.getByTestId('change-implementation-input').fill('一、备份；二、实施；三、验证。');
    await page.getByTestId('change-rollback-input').fill('验证失败时恢复备份并确认业务健康。');

    const createResponse = page.waitForResponse(
      response =>
        response.url().endsWith('/api/v1/changes') &&
        response.request().method() === 'POST'
    );
    await page.getByTestId('change-submit-button').click();
    expect((await createResponse).ok()).toBeTruthy();
    await expect(page).toHaveURL(/\/changes$/);
    await expect(page.getByText(title)).toBeVisible();
  });
});
