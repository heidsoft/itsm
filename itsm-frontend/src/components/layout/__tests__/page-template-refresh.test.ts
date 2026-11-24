/**
 * 页面模板刷新语义回归锁。
 *
 * 缺陷：`BusinessPageTemplate` 的刷新按钮硬编码 `window.location.reload()`。
 * 整页重载会把用户当前设置的筛选条件、搜索词、页码、滚动位置全部清空——
 * 用户点「刷新」的预期是「重新拉一遍数据」，得到的是「一切从头开始」。
 *
 * 更糟的是同一屏存在两个都叫「刷新」的入口：模板自带的那个整页重载，而
 * incidents / changes / problems 等页面在 extraActions 里另有 handleRefresh
 * （真正的增量刷新）。同一个词对应两种行为，用户无法预期。
 *
 * 修法口径：模板只提供 `onRefresh` 回调；不传则不渲染按钮，
 * **绝不**退化成整页重载。插槽式设计让每个页面自己决定「刷新」做什么。
 */

import fs from 'node:fs';
import path from 'node:path';

const ROOT = path.resolve(__dirname, '../../../..');
const TEMPLATE = path.join(ROOT, 'src/components/layout/BusinessPageTemplate.tsx');
const read = (p: string) => fs.readFileSync(p, 'utf8');

/**
 * 逐行去掉注释，避免注释里提到 reload 造成假红/假绿。
 *
 * 注意：不能用「行首是否以 * 开头」这种朴素规则——JSX 里的 `{/* ... *\/}`
 * 块注释，中间行是 `绝不再退化成 window.location.reload()` 这种**没有 * 前缀**
 * 的自然语言，朴素规则会把它当代码（假红）。
 * 因此这里做真正的状态机剥离：跟踪是否处于 /* … *\/ 块注释内。
 */
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
      // 行注释优先于块注释（谁先出现听谁的），但 // 出现在字符串里的情况
      // 本测试不关心（模板里没有含 // 的字符串字面量）
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

describe('页面模板刷新语义', () => {
  const template = read(TEMPLATE);
  const code = stripComments(template);

  it('🔴 模板不再用 window.location.reload() 实现刷新', () => {
    // 只允许在注释里解释「为什么不用」，实现代码里必须一次都不出现
    const hits = code.match(/window\.location\.reload\(\)/g) ?? [];
    expect(hits).toEqual([]);
  });

  it('模板暴露 onRefresh / refreshing 两个受控插槽', () => {
    expect(template).toMatch(/onRefresh\?:\s*\(\)\s*=>\s*void/);
    expect(template).toMatch(/refreshing\?:\s*boolean/);
  });

  it('刷新按钮由 onRefresh 存在与否决定是否渲染（不传就不出）', () => {
    // 关键结构：{onRefresh && (...)}，而不是无条件渲染
    expect(code).toMatch(/\{onRefresh && \(/);
  });

  it('每个使用模板的页面都接了自己的增量刷新，而不是依赖模板默认行为', () => {
    const pages = ['changes', 'problems', 'incidents', 'workbench', 'assets'];
    const missing: string[] = [];
    for (const page of pages) {
      const p = path.join(ROOT, `src/app/(main)/${page}/page.tsx`);
      if (!fs.existsSync(p)) continue;
      const src = read(p);
      if (!src.includes('<BusinessPageTemplate')) continue;
      if (!/onRefresh=/.test(src)) missing.push(page);
    }
    expect(missing).toEqual([]);
  });

  it('传了 onRefresh 的页面必须同时传 refreshing（否则按钮无法转圈也无法防重复点击）', () => {
    const pages = ['changes', 'problems', 'incidents', 'workbench', 'assets'];
    const missing: string[] = [];
    for (const page of pages) {
      const p = path.join(ROOT, `src/app/(main)/${page}/page.tsx`);
      if (!fs.existsSync(p)) continue;
      const src = read(p);
      if (!/onRefresh=/.test(src)) continue;
      if (!/refreshing=/.test(src)) missing.push(page);
    }
    expect(missing).toEqual([]);
  });
});
