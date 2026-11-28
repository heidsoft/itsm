import { expect, test } from '@playwright/test';
import { adminPassword, isolatedBaseURL } from './harness';

test('admin pages fit a 1024px viewport with the sidebar expanded', async ({ page }) => {
  const baseURL = isolatedBaseURL();
  const session = await page.request.post(`${baseURL}/api/v1/auth/login`, {
    data: { username: 'admin', password: adminPassword() },
  });
  expect(session.ok()).toBe(true);
  await page.setViewportSize({ width: 1024, height: 576 });

  for (const path of ['permissions', 'capabilities', 'system-config']) {
    await page.goto(`${baseURL}/admin/${path}`);
    const collapse = page.getByRole('button', { name: '收起侧边栏' });
    if (!(await collapse.isVisible())) {
      await page.getByRole('button', { name: '展开侧边栏' }).click();
    }
    await expect(collapse).toBeVisible();
    await expect(page.getByRole('navigation', { name: '面包屑导航' })).toBeVisible();
    await expect.poll(() => page.evaluate(
      () => document.documentElement.scrollWidth - innerWidth,
    )).toBe(0);
    if (path === 'permissions') {
      for (const name of ['初始化权限字典', '重置', '保存配置']) {
        await expect(page.getByRole('button', { name, exact: true })).toBeInViewport({ ratio: 1 });
      }
    }
  }
});
