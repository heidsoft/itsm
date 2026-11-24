/**
 * 深色模式「不可读」回归锁 —— 只锁真正不可读的少数位置，不做全站扫荡。
 *
 * ── 判据来源（实测对比度，不是感觉） ──────────────────────────────
 * 深色页面底色 `--color-background-primary` = #0f172a。以它为背景计算
 * WCAG 相对亮度对比度（AA 正文 4.5:1 / 大字 3:1）：
 *
 *   text-gray-400  #9ca3af →  7.03  通过（所以「109 处灰字看不清」是错的）
 *   text-slate-400 #94a3b8 →  6.96  通过
 *   text-slate-500 #64748b →  3.75  大字勉强
 *   text-slate-600 #475569 →  2.36  不可读
 *   text-slate-700 #334155 →  1.72  不可读
 *   text-slate-800 #1e293b →  1.22  不可读
 *   text-slate-900 #0f172a →  1.00  完全不可见
 *
 * ── 根本原因（这才是本锁真正要守的东西） ──────────────────────────
 * `globals.css` 的 `.dark` retro-fit 只覆盖 **gray 系列**：
 *   .dark .bg-white / .bg-gray-* / .text-gray-* / .border-gray-* / .divide-gray-*
 * **`slate` 系列一个都没有**。所以深色下：
 *   - `text-slate-600/700` 保持深色 → 深底深字，看不清
 *   - `bg-slate-50`(#f8fafc) 保持浅底，而它的内部文字 `text-gray-900`
 *     却被 retro-fit 改成 `--color-text-primary`(#f8fafc) → 浅底浅字，
 *     对比度 1.00，**完全看不见**（这比单纯灰字更严重）
 *
 * 修法：这些位置改用语义变量（`--color-background-secondary` 浅色值恰为
 * #f8fafc，与 bg-slate-50 等价；深色自动变 #1e293b）。
 */

import fs from 'node:fs';
import path from 'node:path';
import { execSync } from 'node:child_process';

const ROOT = path.resolve(__dirname, '../../../..');

/** 深色背景上不可读的 slate 裸类（已由对比度实测判定）。 */
const UNREADABLE = [
  'text-slate-600',
  'text-slate-700',
  'text-slate-800',
  'text-slate-900',
  'bg-slate-50',
  'bg-slate-100',
  'border-slate-100',
  'border-slate-200',
];

describe('深色模式不可读缺口', () => {
  it('🔴 retro-fit 确实不含 slate 系列（这是本缺陷存在的根本前提）', () => {
    const css = fs.readFileSync(path.join(ROOT, 'src/app/globals.css'), 'utf8');
    const idx = css.indexOf('.dark .bg-white');
    expect(idx).toBeGreaterThan(-1);
    const segment = css.slice(idx, idx + 4000);
    // 若哪天有人给 slate 补了 retro-fit，本文件的其他断言就可以放宽，
    // 但在此之前必须保持「slate 无兜底」这个事实，否则下面的锁会变成假绿。
    expect(segment).not.toMatch(/\.dark\s+\.(?:text|bg|border)-slate-/);
    // 同时确认 gray 系列确实被覆盖（对照组，证明扫描逻辑本身有效）
    expect(segment).toMatch(/\.dark\s+\.text-gray-/);
  });

  it('🔴 源码里不再出现深色下不可读的 slate 裸类', () => {
    // 用 grep 扫真实源码（排除测试与注释行），比逐文件 readFile 更不容易漏面
    const hits: string[] = [];
    for (const cls of UNREADABLE) {
      let out = '';
      try {
        out = execSync(
          `grep -rn "className.*${cls}" src --include="*.tsx" 2>/dev/null || true`,
          { cwd: ROOT, encoding: 'utf8' },
        );
      } catch { out = ''; }
      for (const line of out.split('\n').filter(Boolean)) {
        if (line.includes('__tests__')) continue;
        // 已换成 var(--color-...) 的行可能仍在同一字符串里出现关键词
        if (line.includes('var(--color-')) continue;
        hits.push(`[${cls}] ${line.trim().slice(0, 120)}`);
      }
    }
    expect(hits).toEqual([]);
  });

  it('这批位置已接入语义变量（背景/边框真的是变量而不是写死色）', () => {
    const s = fs.readFileSync(
      path.join(ROOT, 'src/app/(main)/cmdb/registry/page.tsx'),
      'utf8',
    );
    expect(s).toContain('bg-[var(--color-background-secondary)]');
    const guide = fs.readFileSync(
      path.join(ROOT, 'src/app/(main)/admin/components/AdminSetupGuide.tsx'),
      'utf8',
    );
    expect(guide).toContain('text-[var(--color-text-secondary)]');
    expect(guide).not.toContain('text-slate-700');
  });

  it('语义变量在深浅两套主题下都有定义且取值不同（否则切主题不生效）', () => {
    const css = fs.readFileSync(path.join(ROOT, 'src/app/globals.css'), 'utf8');
    for (const v of ['--color-background-secondary', '--color-text-secondary']) {
      const vals = [
        ...css.matchAll(new RegExp(`${v}\\s*:\\s*(#\\w+)`, 'g')),
      ].map((m) => m[1].toLowerCase());
      expect({ v, distinct: new Set(vals).size }).toEqual({ v, distinct: 2 });
    }
  });
});
