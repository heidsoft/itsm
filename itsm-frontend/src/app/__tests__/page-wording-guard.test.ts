/**
 * 页面头部文案不得使用装饰性措辞（回归锁）。
 *
 * 做法：从真实页面源码里抽取用户可见的 title/description/placeholder 字面量，
 * 再交给 wording-guard 的判定函数。这样断言的对象是「页面上真正会显示的文字」，
 * 而不是「文件里是否出现过某个词」——后者加一行注释就能骗过。
 *
 * 反向验证：把 noc/page.tsx 的描述改回「重大事件作战室：实时跟踪…」，
 * 本测试会在该文件上报出「作战室」，断言转红。
 */

import fs from 'node:fs';
import path from 'node:path';

import { findDecorativeWording } from '@/lib/copy/wording-guard';

const SRC = path.join(process.cwd(), 'src');

/** 本轮审查过的页面：这些页面的头部文案必须干净。 */
const PAGES = [
  'app/(main)/noc/page.tsx',
  'app/(main)/cmdb/registry/page.tsx',
  'app/(main)/cmdb/certification/page.tsx',
  'app/(main)/ai/approval/page.tsx',
  'app/(main)/customers/page.tsx',
];

/**
 * 抽取用户可见文案。
 *
 * 三种形态都要覆盖，缺一类就会漏检——本轮「作战室」就恰好写在 JSX 文本子节点里
 * （`<Text type="secondary">重大事件作战室：…</Text>`），只扫属性字面量的话
 * 测试会全程绿，反向验证也绿（假绿）。
 *   ① 属性形式：title= / description= / placeholder=
 *   ② 大括号字符串：{'…'}
 *   ③ JSX 文本子节点：>…中文…<（排除花括号，跳过注释 {* … *} 与三元表达式）
 */
function visibleLiterals(source: string): string[] {
  const out: string[] = [];
  const re = /\b(?:title|description|placeholder)=(?:\{)?\s*['"`]([^'"`]*)['"`]/g;
  for (const m of source.matchAll(re)) {
    if (m[1].trim()) out.push(m[1]);
  }
  const re2 = /\{\s*['"`]([^'"`]{12,})['"`]\s*\}/g;
  for (const m of source.matchAll(re2)) {
    out.push(m[1]);
  }
  const re3 = />([^<>{}]*[\u4e00-\u9fa5][^<>{}]*)</g;
  for (const m of source.matchAll(re3)) {
    const text = m[1].trim();
    if (text) out.push(text);
  }
  return out;
}

describe('页面头部文案：去装饰性措辞', () => {
  it.each(PAGES)('%s 的可见文案不含装饰性词与能力夸大', file => {
    const full = path.join(SRC, file);
    if (!fs.existsSync(full)) throw new Error(`缺少页面文件 ${file}`);
    const texts = visibleLiterals(fs.readFileSync(full, 'utf8'));
    const hits = findDecorativeWording(texts);
    expect({ file, hits }).toEqual({ file, hits: [] });
  });

  it('抽取器必须覆盖 JSX 文本子节点（否则守卫只能扫到属性，形成假绿）', () => {
    const sample = `<Text type="secondary">重大事件作战室：实时跟踪企业级重大事件处置进展</Text>`;
    expect(visibleLiterals(sample)).toEqual(
      expect.arrayContaining(['重大事件作战室：实时跟踪企业级重大事件处置进展'])
    );
  });

  it('判定函数本身能命中典型违规（防止函数恒返回空导致全部假绿）', () => {
    const hits = findDecorativeWording([
      '重大事件作战室：实时跟踪企业级重大事件处置进展',
      '逾期的待办会持续提醒',
      '把发现与对账串成可持续的图谱治理闭环',
    ]);
    expect(hits.map(h => h.term)).toEqual(
      expect.arrayContaining(['作战室', '持续提醒', '闭环'])
    );
  });
});
