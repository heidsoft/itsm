import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

/**
 * 共享排版/工具栏回归：短标签不得逐字符折行，工具栏不得溢出卡片，
 * 表格技术标识列在窄视口下必须保持可读宽度（横向滚动发生在表格内部）。
 */
test.describe('Admin shared layout - toolbar typography', () => {
  for (const viewport of [
    { width: 1280, height: 720 },
    { width: 1024, height: 768 },
  ]) {
    test(`permission role label and toolbar fit at ${viewport.width}x${viewport.height}`, async ({
      page,
    }) => {
      await page.setViewportSize(viewport);
      await loginAndReturn(page);
      await page.goto('/admin/permissions');

      const label = page.locator('span.field-label').first();
      await expect(label).toBeVisible();

      const geometry = await label.evaluate((el) => {
        const rect = el.getBoundingClientRect();
        const style = getComputedStyle(el);
        const card = el.closest('.ant-card') as HTMLElement | null;
        const row = el.closest('.ant-row') as HTMLElement | null;
        return {
          labelHeight: rect.height,
          whiteSpace: style.whiteSpace,
          wordBreak: style.wordBreak,
          cardClientWidth: card?.clientWidth ?? Infinity,
          cardScrollWidth: card?.scrollWidth ?? Infinity,
          rowClientWidth: row?.clientWidth ?? Infinity,
          rowScrollWidth: row?.scrollWidth ?? Infinity,
        };
      });

      expect(geometry.whiteSpace).toBe('nowrap');
      expect(geometry.wordBreak).toBe('keep-all');
      // 短标签一旦逐字折行，高度会从 ~22px 涨到 100px 以上。
      expect(geometry.labelHeight).toBeLessThan(30);
      expect(geometry.cardScrollWidth).toBeLessThanOrEqual(geometry.cardClientWidth + 2);
      expect(geometry.rowScrollWidth).toBeLessThanOrEqual(geometry.rowClientWidth + 2);
    });
  }

  test('capability key stays readable at 1024x768 with table-local horizontal scroll', async ({ page }) => {
    await page.setViewportSize({ width: 1024, height: 768 });
    await loginAndReturn(page);
    await page.goto('/admin/capabilities');
    await expect(page.locator('.ant-table-tbody tr').first()).toBeVisible();
    await expect(page.locator('.code-token').first()).toBeVisible();

    const geometry = await page.evaluate(() => {
      const header = [...document.querySelectorAll('.ant-table-thead th')].find(
        (th) => th.textContent?.trim() === '能力',
      ) as HTMLElement | undefined;
      const token = document.querySelector('.code-token') as HTMLElement | null;
      // antd v6 在 scroll.x 下用 .ant-table-content 作为滚动容器。
      const scroller = document.querySelector('.ant-table-content') as HTMLElement | null;
      return {
        keyColumnWidth: header?.getBoundingClientRect().width ?? 0,
        tokenHeight: token?.getBoundingClientRect().height ?? Infinity,
        tokenWhiteSpace: token ? getComputedStyle(token).whiteSpace : 'normal',
        scrollerOverflowX: scroller ? getComputedStyle(scroller).overflowX : 'visible',
        scrollerClientWidth: scroller?.clientWidth ?? Infinity,
        scrollerScrollWidth: scroller?.scrollWidth ?? Infinity,
        scrollerRight: scroller?.getBoundingClientRect().right ?? Infinity,
      };
    });

    expect(geometry.keyColumnWidth).toBeGreaterThanOrEqual(255);
    expect(geometry.tokenWhiteSpace).toBe('nowrap');
    expect(geometry.tokenHeight).toBeLessThan(30);
    expect(geometry.scrollerOverflowX).toBe('auto');
    expect(geometry.scrollerScrollWidth).toBeGreaterThan(geometry.scrollerClientWidth);
    // 表格自身的滚动容器必须留在视口内，不能把横向滚动推给整个页面。
    expect(geometry.scrollerRight).toBeLessThanOrEqual(1024);
  });
});
