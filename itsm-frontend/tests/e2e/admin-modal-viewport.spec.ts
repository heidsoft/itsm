import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

/** 共享弹窗 shell 的视口边界回归：1280x720 下页脚不得掉出视口，body 必须可滚动。 */
test.describe('Admin modal shell - viewport fit', () => {
  test('new menu dialog stays inside the viewport at 1280x720', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 720 });
    await loginAndReturn(page);
    await page.goto('/admin/menus');
    await expect(page.getByRole('heading', { name: '菜单管理' })).toBeVisible();

    await page.getByRole('button', { name: '新建菜单' }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    // 等 Ant Design 的进入动画结束再测量，否则拿到的是缩放中间态。
    await page.waitForTimeout(600);

    const geometry = await dialog.evaluate((el) => {
      const body = el.querySelector('.ant-modal-body') as HTMLElement | null;
      const footer = el.querySelector('.ant-modal-footer') as HTMLElement | null;
      const rect = el.getBoundingClientRect();
      return {
        viewportHeight: window.innerHeight,
        modalBottom: rect.bottom,
        footerBottom: footer?.getBoundingClientRect().bottom ?? Infinity,
        bodyOverflowY: body ? getComputedStyle(body).overflowY : 'visible',
      };
    });

    expect(geometry.modalBottom).toBeLessThanOrEqual(geometry.viewportHeight);
    expect(geometry.footerBottom).toBeLessThanOrEqual(geometry.viewportHeight);
    expect(geometry.bodyOverflowY).toBe('auto');
  });

  test('select dropdown inside a bounded dialog is not clipped', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 720 });
    await loginAndReturn(page);
    await page.goto('/admin/menus');
    await page.getByRole('button', { name: '新建菜单' }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    // 等进入动画结束，否则 Playwright 会判定元素“不稳定”而拒绝点击。
    await page.waitForTimeout(700);
    // antd v6 的 search input 覆盖在 placeholder 上，直接点原生 combobox 更稳。
    await dialog.locator('#parentId').click();

    const dropdown = page.locator('.ant-select-dropdown:not(.ant-select-dropdown-hidden)');
    await expect(dropdown).toBeVisible();
    const box = await dropdown.boundingBox();
    expect(box).not.toBeNull();
    expect(box!.y).toBeGreaterThanOrEqual(0);
    expect(box!.y + box!.height).toBeLessThanOrEqual(720);
  });
});
