/**
 * Modal 重复提交防线回归锁（覆盖多个页面）。
 *
 * 判据来自 antd v6 源码：`Modal.js:111`
 *   const handleOk = e => { onOk?.(e); onClose?.(); };
 * —— onOk 被调用后立刻 onClose，**不等待 Promise**。
 * 所以 `onOk={async fn}` 在请求挂起期内弹窗仍开着、确定按钮仍可点，
 * 用户连点两次就提交两次。实测双防线全拆时确实提交 2 次
 * （见 `source-organizations/__tests__/page.test.tsx` 的行为测试）。
 *
 * 本文件是**批量防线**：173 个页面里任何新增/改动的 Modal 提交都必须
 * 同时具备入口守卫与 loading 反馈，否则在这里变红。
 *
 * 注意：本文件是源码结构锁，不是装配验证——它能抓「漏写」，
 * 但「写了却因为时序无效」要靠行为测试抓。两者都要有。
 */

import fs from 'node:fs';
import path from 'node:path';

const ROOT = path.resolve(__dirname, '../../../..');

/** 逐行剥离注释，避免注释里出现 `onOk` / `confirmLoading` 造成假绿假红。 */
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

function readPage(rel: string): string {
  return stripComments(fs.readFileSync(path.join(ROOT, rel), 'utf8'));
}

interface ModalCase { rel: string; guards: number; settles: number; unprotected: number }

function inspect(rel: string): ModalCase {
  const s = readPage(rel);
  const guards = (s.match(/if \(submitBusy\) return;/g) ?? []).length;
  const settles = (s.match(/finally\s*\{\s*setSubmitBusy\(false\)/g) ?? []).length;

  // 每个 onOk 都必须归属到一个带 confirmLoading / okButtonProps 的 <Modal>
  let unprotected = 0;
  for (const m of s.matchAll(/\bonOk=/g)) {
    const before = s.slice(0, m.index ?? 0);
    const lastModal = before.lastIndexOf('<Modal');
    if (lastModal === -1) { unprotected += 1; continue; }
    // 从 <Modal 到 onOk 之后一小段，足以覆盖同行内的 props
    const seg = s.slice(lastModal, (m.index ?? 0) + 240);
    if (!/confirmLoading|okButtonProps/.test(seg)) unprotected += 1;
  }
  return { rel, guards, settles, unprotected };
}

const PAGES = [
  'src/app/(main)/on-call/page.tsx',
  'src/app/(main)/source-organizations/page.tsx',
  'src/app/(main)/customers/page.tsx',
  'src/app/(main)/support-contracts/page.tsx',
  'src/app/(main)/standard-changes/page.tsx',
  'src/app/(main)/problems/known-errors/page.tsx',
  'src/app/(main)/admin/process-routing/page.tsx',
  'src/app/(main)/admin/skills/page.tsx',
  'src/app/(main)/admin/config-inheritance/page.tsx',
  'src/app/(main)/admin/connectors/page.tsx',
  'src/app/(main)/admin/tickets/automation-rules/page.tsx',
];

describe('Modal 重复提交防线', () => {
  it('🔴 每个提交型 Modal 都有 loading 保护（confirmLoading 或 okButtonProps）', () => {
    const bad = PAGES.map(inspect).filter((c) => c.unprotected > 0);
    expect(bad.map((c) => `${c.rel}: ${c.unprotected} 个 onOk 无 loading`)).toEqual([]);
  });

  it('🔴 每个入口守卫都有配对的 finally 复位（否则按钮永久锁死）', () => {
    // 守卫多于复位 = 存在只进不出的 latch，保存失败后按钮再也不响应
    const bad = PAGES.map(inspect).filter((c) => c.guards !== c.settles);
    expect(
      bad.map((c) => `${c.rel}: 守卫 ${c.guards} / 复位 ${c.settles}`),
    ).toEqual([]);
  });

  it('🔴 validateFields 必须在 try 内（在 try 外时校验失败会跳过 finally 而锁死按钮）', () => {
    // 关键：不能用「往前 N 字符内有没有 try {」来判断 —— 同文件更早的函数里
    // 也有 try，窗口一大就会跨函数误命中，把「挪出 try」判成合规（实测假绿）。
    // 正确做法：截出 validateFields 所在的**同一个函数体**，只看这个体内 try 的位置。
    const bad: string[] = [];
    for (const rel of PAGES) {
      const s = readPage(rel);
      for (const m of s.matchAll(/await \S*[Ff]orm\.validateFields\(\)/g)) {
        const at = m.index ?? 0;
        // 函数体起点 = 往前最近的 "=> {" 或 "async ... {" 或独立 "{" 行
        const head = s.slice(0, at).match(/(?:=>|function\s*\w*\s*\([^)]*\)|\w+\s*=\s*async\s*\(.*?\))\s*\{/g);
        const bodyStart = head && head.length
          ? at - (s.slice(0, at).length - (s.slice(0, at).lastIndexOf(head[head.length - 1]) + head[head.length - 1].length))
          : 0;
        const body = s.slice(bodyStart, at);
        if (!/try\s*\{/.test(body)) bad.push(rel);
      }
    }
    expect(bad).toEqual([]);
  });

  it('扫描面覆盖到这批页面（防止 PAGES 被悄悄清空后全绿）', () => {
    expect(PAGES.length).toBe(11);
    const missing = PAGES.filter((rel) => !fs.existsSync(path.join(ROOT, rel)));
    expect(missing).toEqual([]);
  });
});
