/**
 * 字面导航目标必须落在真实页面上
 *
 * 背景（2026-10-31 生产实测）：服务请求列表空态跳 `/service-requests/new`，
 * 而当时仓库里只有 `service-requests/[id]/page.tsx`。Next 把 `new` 当动态段，
 * `Number("new")` 得到 NaN，前端随后请求 `GET /api/v1/service-requests/NaN`。
 * `/tickets/new`、`/knowledge/articles/create` 是同一类。现在三个入口都已指向
 * 真实页面；本守卫防止下一次再靠动态段「假装路由存在」。
 *
 * 规则：`router.push/replace` 的字面路径如果只能被动态段接住，而被吃掉的段是
 * new/create/edit 这类静态关键字，就说明页面漏建了。
 */

import fs from 'fs';
import path from 'path';

const SRC_ROOT = path.join(__dirname, '..', '..');
const APP_DIR = path.join(SRC_ROOT, 'app');

const RESERVED_STATIC_SEGMENTS = new Set(['new', 'create', 'edit', 'list', 'detail']);

function collectSourceFiles(dir: string, out: string[] = []): string[] {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === '__tests__' || entry.name === 'node_modules') continue;
      collectSourceFiles(full, out);
    } else if (/\.(tsx|ts)$/.test(entry.name) && !/\.(test|spec)\.(tsx|ts)$/.test(entry.name)) {
      out.push(full);
    }
  }
  return out;
}

function collectRoutePatterns(dir: string, out: string[][] = []): string[][] {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      collectRoutePatterns(full, out);
    } else if (entry.name === 'page.tsx') {
      const rel = path.relative(APP_DIR, full).replace(/[\\/]?page\.tsx$/, '');
      out.push(
        rel
          .split(path.sep)
          .filter(segment => segment && !(segment.startsWith('(') && segment.endsWith(')')))
      );
    }
  }
  return out;
}

function matchPattern(pattern: string[], target: string[]): string[] | null {
  if (pattern.length !== target.length) return null;
  const dynamicHits: string[] = [];
  for (let i = 0; i < pattern.length; i += 1) {
    if (pattern[i].startsWith('[') && pattern[i].endsWith(']')) {
      dynamicHits.push(target[i]);
    } else if (pattern[i] !== target[i]) {
      return null;
    }
  }
  return dynamicHits;
}

function isDeadTarget(target: string, patterns: string[][]): boolean {
  const segments = target.split('/').filter(Boolean);
  const matches = patterns
    .map(pattern => matchPattern(pattern, segments))
    .filter((hits): hits is string[] => hits !== null);

  if (matches.length === 0) return true;
  return matches.every(hits => hits.length > 0 && hits.some(hit => RESERVED_STATIC_SEGMENTS.has(hit)));
}

describe('导航目标指向真实页面', () => {
  it('router.push/replace 的字面路径不得落进动态段后被当成 ID', () => {
    const patterns = collectRoutePatterns(APP_DIR);
    const offenders: string[] = [];

    for (const file of collectSourceFiles(SRC_ROOT)) {
      const content = fs.readFileSync(file, 'utf8');
      for (const match of content.matchAll(/router\.(?:push|replace)\(\s*(['"])([^'"]*)\1/g)) {
        const raw = match[2];
        if (!raw.startsWith('/') || raw.includes('${')) continue;
        const target = raw.split('?')[0].split('#')[0];
        if (isDeadTarget(target, patterns)) {
          offenders.push(`${path.relative(SRC_ROOT, file)} -> ${raw}`);
        }
      }
    }

    expect(offenders).toEqual([]);
  });

  it('匹配器本身有效（阳性对照）', () => {
    const patterns = [['service-requests', '[id]'], ['tickets', '[ticketId]'], ['tickets', 'create']];

    expect(isDeadTarget('/service-requests/new', patterns)).toBe(true);
    expect(isDeadTarget('/service-requests/42', patterns)).toBe(false);
    expect(isDeadTarget('/tickets/new', patterns)).toBe(true);
    expect(isDeadTarget('/tickets/create', patterns)).toBe(false);
  });
});
