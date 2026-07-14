import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

const viewports = [
  { name: '14-inch', width: 1440, height: 900 },
  { name: '1080p', width: 1920, height: 1080 },
  { name: '2k', width: 2560, height: 1440 },
  { name: '4k', width: 3840, height: 2160 },
  { name: 'laptop-small', width: 1366, height: 768 },
  { name: 'tablet-portrait', width: 820, height: 1180 },
  { name: 'tablet-landscape', width: 1024, height: 768 },
];

async function assertNoHorizontalScroll(page: import('@playwright/test').Page) {
  const result = await page.evaluate(() => {
    const doc = document.documentElement;
    const body = document.body;
    const scrollWidth = Math.max(doc?.scrollWidth || 0, body?.scrollWidth || 0);
    const clientWidth = window.innerWidth;
    return { scrollWidth, clientWidth };
  });
  expect(result.scrollWidth).toBeLessThanOrEqual(result.clientWidth + 2);
}

test.describe('Compatibility - 响应式布局', () => {
  test('Dashboard layout should be stable across common resolutions', async ({ page }) => {
    await loginAndReturn(page, 'admin', 'admin123');

    for (const vp of viewports) {
      await test.step(`viewport: ${vp.name} ${vp.width}x${vp.height}`, async () => {
        await page.setViewportSize({ width: vp.width, height: vp.height });
        await page.goto('/dashboard');
        await page.waitForLoadState('networkidle');
        await expect(page.locator('body')).toBeVisible();
        await assertNoHorizontalScroll(page);
      });
    }
  });

  test('Ticket list layout should be stable across common resolutions', async ({ page }) => {
    await loginAndReturn(page, 'admin', 'admin123');

    for (const vp of viewports) {
      await test.step(`viewport: ${vp.name} ${vp.width}x${vp.height}`, async () => {
        await page.setViewportSize({ width: vp.width, height: vp.height });
        await page.goto('/tickets');
        await page.waitForLoadState('networkidle');
        await expect(page.locator('body')).toBeVisible();
        await assertNoHorizontalScroll(page);
      });
    }
  });
});
