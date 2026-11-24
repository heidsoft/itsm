/**
 * 页面容器深色模式回归锁。
 *
 * 缺陷：两个同名的 PageContainer 都把颜色写进了**内联 style**。
 *   `app/components/PageContainer.tsx`  →  `background: '#fff'`
 *   `components/layout/PageContainer.tsx` →  `borderBottom: '1px solid #e5e7eb'`
 *
 * 为什么 globals.css 的 `.dark .bg-white` / `.dark .border-gray-*` 救不了它们：
 * 那套 retro-fit 只作用于 **Tailwind 类名**；内联 style 的优先级来自元素本身，
 * 类选择器无论怎么排都盖不住。于是深色模式下这两个容器会留下白底 + 亮灰分隔线。
 *
 * 判据：颜色必须交给 CSS 变量（`--color-background-primary` /
 * `--color-border-primary` / `--color-text-*`），由主题切换提供深浅两套值。
 */

import fs from 'node:fs';
import path from 'node:path';

const ROOT = path.resolve(__dirname, '../../../..');
const read = (p: string) => fs.readFileSync(path.join(ROOT, p), 'utf8');

/** 剥离注释：注释里会提到 '#fff' / '#e5e7eb' 作为反例说明，不能算实现。 */
function stripComments(src: string): string {
  const out: string[] = [];
  let inBlock = false;
  for (const line of src.split('\n')) {
    let rest = line;
    let keep = '';
    while (rest.length > 0) {
      if (inBlock) {
        const end = rest.indexOf('*/');
        if (end === -1) { rest = ''; break; }
        rest = rest.slice(end + 2);
        inBlock = false;
        continue;
      }
      const block = rest.indexOf('/*');
      const single = rest.indexOf('//');
      if (single !== -1 && (block === -1 || single < block)) {
        keep += rest.slice(0, single);
        rest = '';
        break;
      }
      if (block !== -1) {
        keep += rest.slice(0, block);
        rest = rest.slice(block + 2);
        inBlock = true;
        continue;
      }
      keep += rest;
      rest = '';
    }
    if (keep.trim()) out.push(keep);
  }
  return out.join('\n');
}

const APP_CONTAINER = 'src/app/components/PageContainer.tsx';
const LAYOUT_CONTAINER = 'src/components/layout/PageContainer.tsx';

describe('页面容器深色模式', () => {
  it('🔴 app/components/PageContainer 不再把背景写死为 #fff', () => {
    const code = stripComments(read(APP_CONTAINER));
    // 内联 style 里的写死白底（'#fff' / '#ffffff' / 'white'）
    expect(code).not.toMatch(/background:\s*'#(?:fff|ffffff)'/i);
    expect(code).not.toMatch(/background:\s*'white'/i);
    // 必须改用主题变量
    expect(code).toMatch(/background:\s*'var\(--color-background-primary\)'/);
  });

  it('🔴 layout/PageContainer 不再用写死的亮灰边框与灰阶文字', () => {
    const code = stripComments(read(LAYOUT_CONTAINER));
    // 内联 style 的写死亮灰边框
    expect(code).not.toMatch(/borderBottom:\s*'1px solid #e5e7eb'/i);
    expect(code).toMatch(/borderBottom:\s*'1px solid var\(--color-border-primary\)'/);
    // Tailwind 裸灰阶（在深色下虽会被 retro-fit 兜住，但容器层应直接用语义变量，
    // 避免依赖 retro-fit 这条兜底机制）
    expect(code).not.toMatch(/text-gray-900|text-gray-500/);
  });

  it('两个容器都用语义色变量而非写死色值', () => {
    for (const rel of [APP_CONTAINER, LAYOUT_CONTAINER]) {
      const code = stripComments(read(rel));
      expect(code).toMatch(/var\(--color-(?:background|surface|border|text)-[a-z]+\S*\)/);
    }
  });

  it('变量在 CSS 里确实有深浅两套取值（否则改了等于没改）', () => {
    const css = read('src/app/globals.css');
    // --color-background-primary 与 --color-border-primary 必须至少各出现两次
    // （浅色 :root 一套 + .dark 一套），否则用的是同一个值，切主题不生效。
    for (const v of ['--color-background-primary', '--color-border-primary']) {
      const count = (css.match(new RegExp(`${v}\\s*:`, 'g')) ?? []).length;
      expect({ v, count }).toEqual({ v, count: expect.any(Number) });
      expect(count).toBeGreaterThanOrEqual(2);
    }
  });
});
