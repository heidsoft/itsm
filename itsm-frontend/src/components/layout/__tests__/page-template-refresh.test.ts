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
 * 从源码中截出 `<BusinessPageTemplate ... >` 这个标签的文本。
 *
 * 截标签而不是整份文件：整份文件里可能还有别的组件也带同名 prop
 * （service-catalog 的 `<ServiceCatalogFilters onRefresh=...>` 就是实例），
 * 用全文匹配会把「传给了别的组件」误判成「模板插槽已接线」（假绿）。
 *
 * 🔴 不能扫 `>` 找标签结束：属性值里的箭头函数 `onViewChange={view => ...}`
 * 和比较运算符都含 `>`，会提前截断（实测把 incidents 判成「未接线」）。
 * 正确做法 = 括号深度平衡 + 只在深度 0 且**不在字符串内**时认 `>`。
 */
function templateTag(src: string): string | null {
  const start = src.indexOf('<BusinessPageTemplate');
  if (start === -1) return null;
  let depth = 0;
  let quote: string | null = null;
  for (let i = start; i < src.length; i += 1) {
    const ch = src[i];
    if (quote) {
      if (ch === quote && src[i - 1] !== '\\') quote = null;
      continue;
    }
    if (ch === '"' || ch === "'" || ch === '`') { quote = ch; continue; }
    if (ch === '{') depth += 1;
    else if (ch === '}') depth -= 1;
    else if (ch === '>' && depth === 0) return src.slice(start, i + 1);
  }
  return null;
}

/**
 * 取 `<BusinessPageTemplate ...>` 的**顶层 prop 名集合**。
 *
 * 为什么不能直接在标签文本里找 `onRefresh=`：incidents 的标签里有
 *   filters={{ content: (<IncidentFilters onRefresh={handleRefresh} />) }}
 * —— 这个 onRefresh 传给的是嵌套的筛选组件，不是模板插槽。在标签文本里
 * 搜子串会把它当成「模板插槽接线了」（实测假绿）。必须只在 `{ }` 深度为 0
 * 的位置识别 `name=`，嵌套对象里的同名 prop 一律不计。
 */
function topLevelProps(tag: string): Set<string> {
  const names = new Set<string>();
  let depth = 0;
  let quote: string | null = null;
  // 跳过开头的 "<BusinessPageTemplate"
  for (let i = tag.indexOf('BusinessPageTemplate') + 'BusinessPageTemplate'.length; i < tag.length; i += 1) {
    const ch = tag[i];
    if (quote) {
      if (ch === quote && tag[i - 1] !== '\\') quote = null;
      continue;
    }
    if (ch === '"' || ch === "'" || ch === '`') { quote = ch; continue; }
    if (ch === '{') { depth += 1; continue; }
    if (ch === '}') { depth -= 1; continue; }
    if (depth !== 0) continue;
    // 顶层位置：识别 `ident=`（前面必须是空白，避免匹配到 a-b 里的片段）
    if (ch === '=') {
      const before = tag.slice(0, i);
      const m = before.match(/([A-Za-z_$][\w$]*)\s*$/);
      if (m) names.add(m[1]);
    }
  }
  return names;
}

/** 扫描真实源码找出所有使用模板的业务页面，不写死清单。 */
function discoverTemplatePages(): string[] {
  const mainDir = path.join(ROOT, 'src/app/(main)');
  const found: string[] = [];
  const walk = (dir: string) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      if (entry.name === '__tests__' || entry.name === 'node_modules') continue;
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) { walk(full); continue; }
      if (entry.name !== 'page.tsx') continue;
      if (read(full).includes('<BusinessPageTemplate')) {
        found.push(path.relative(mainDir, full));
      }
    }
  };
  if (fs.existsSync(mainDir)) walk(mainDir);
  return found;
}

/** 消费模板的页面（相对 `src/app/(main)` 的目录名），运行期发现而非硬编码。 */
const TEMPLATE_PAGES = discoverTemplatePages().map((rel) =>
  rel.replace(/^\.\//, '').replace(/\/page\.tsx$/, ''),
);

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
    const missing: string[] = [];
    for (const page of TEMPLATE_PAGES) {
      const p = path.join(ROOT, `src/app/(main)/${page}/page.tsx`);
      if (!fs.existsSync(p)) continue;
      const tag = templateTag(read(p));
      if (!tag) { missing.push(`${page}(未解析到模板标签)`); continue; }
      // 必须是模板自己的顶层 prop，不能是嵌套组件里的同名 prop
      if (!topLevelProps(tag).has('onRefresh')) missing.push(page);
    }
    expect(missing).toEqual([]);
  });

  it('传了 onRefresh 的页面必须同时传 refreshing（否则按钮无法转圈也无法防重复点击）', () => {
    const missing: string[] = [];
    for (const page of TEMPLATE_PAGES) {
      const p = path.join(ROOT, `src/app/(main)/${page}/page.tsx`);
      if (!fs.existsSync(p)) continue;
      const tag = templateTag(read(p));
      if (!tag) continue;
      const props = topLevelProps(tag);
      if (!props.has('onRefresh')) continue;
      if (!props.has('refreshing')) missing.push(page);
    }
    expect(missing).toEqual([]);
  });

  it('扫描面非空且与真实消费方一致（防止发现逻辑失效后真空跑全绿）', () => {
    // 动态发现若返回空数组，上面两条会被静默跳过 → 假绿
    expect(TEMPLATE_PAGES.length).toBeGreaterThan(0);
    expect(TEMPLATE_PAGES).toContain('incidents');
  });
});
