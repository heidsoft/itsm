/**
 * 最近访问标题解析：行为测试。
 *
 * 缺陷背景：`RecentVisitTracker` 原来维护一份硬编码 `pathMeta`（仅 11 条），
 * 其余 90 条菜单页面会回退成 URL 尾段英文（`workbench`、`service-requests`、`noc`），
 * 而这 11 条又和菜单真名不同步（菜单叫「服务台」、这里叫「首页」）。
 * 现在标题唯一来源是后端菜单，本测试直接用真实菜单树当输入。
 */

import fs from 'node:fs';
import path from 'node:path';

import {
  collectPathLabels,
  resolveVisitTitle,
  shouldTrackVisit,
} from '@/components/layout/recent-visit-title';
import type { MenuItem as ApiMenuItem } from '@/lib/api/menu-api';

const FRONTEND_ROOT = process.cwd();
const REPO_ROOT = path.resolve(FRONTEND_ROOT, '..');
const BASELINE = path.join(REPO_ROOT, 'itsm-backend/pkg/menubaseline/baseline.go');

/** 用真实基线构造 main/admin 两棵树，与后端 /auth/menus 的形状一致。 */
function realMenuTrees(): { main: ApiMenuItem[]; admin: ApiMenuItem[] } {
  const src = fs.readFileSync(BASELINE, 'utf8');
  const re = /\{Name:\s*"([^"]+)",\s*Path:\s*"([^"]+)",\s*Icon:\s*"[^"]*",([\s\S]*?)\}/g;
  const specs: Array<{ name: string; path: string; parentPath: string }> = [];
  for (const m of src.matchAll(re)) {
    const rest = m[3];
    const parent = /ParentPath:\s*"([^"]*)"/.exec(rest)?.[1] ?? '';
    specs.push({ name: m[1], path: m[2], parentPath: parent });
  }

  const byPath = new Map<string, ApiMenuItem>();
  for (const s of specs) {
    byPath.set(s.path, {
      id: 0, name: s.name, path: s.path, sortOrder: 0, tenantId: 1,
      isVisible: true, isEnabled: true, children: [],
    });
  }
  for (const s of specs) {
    if (!s.parentPath) continue;
    byPath.get(s.parentPath)?.children!.push(byPath.get(s.path)!);
  }
  const roots = specs.filter(s => !s.parentPath).map(s => byPath.get(s.path)!);
  const ADMIN_ROOTS = ['/admin', '/system', '/workflow'];
  const isAdminRoot = (p: string) => ADMIN_ROOTS.some(a => p === a || p.startsWith(`${a}/`));
  return { main: roots.filter(r => !isAdminRoot(r.path)), admin: roots.filter(r => isAdminRoot(r.path)) };
}

describe('最近访问标题解析', () => {
  const trees = realMenuTrees();
  const labels = collectPathLabels(trees.main, collectPathLabels(trees.admin, new Map()));

  it('菜单里有的页面用菜单名，不用英文 URL 段', () => {
    // 这三条正是原来会回退成英文的典型（/workbench、/service-requests、/noc）。
    // 期望值全部取自 baseline.go 的真实 Name，不是凭印象写的。
    expect(resolveVisitTitle('/workbench', labels)).toBe('我的工作台');
    expect(resolveVisitTitle('/service-requests', labels)).toBe('服务请求');
    expect(resolveVisitTitle('/my-requests', labels)).toBe('我的请求');
    expect(resolveVisitTitle('/noc', labels)).toBe('重大事件');
  });

  it('每一个菜单条目的标题都等于它在菜单里的名字（全量校验）', () => {
    const mismatches: string[] = [];
    for (const [p, name] of labels) {
      const resolved = resolveVisitTitle(p, labels);
      if (resolved !== name) mismatches.push(`${p}: 期望「${name}」实际「${resolved}」`);
    }
    expect(mismatches).toEqual([]);
  });

  it('深层子路径用「菜单名 · 尾段」，ID 段显示为详情', () => {
    expect(resolveVisitTitle('/tickets/12', labels)).toBe('工单管理 · 详情 #12');
  });

  it('不在菜单里的路径才回退，且不暴露英文段名给业务用户', () => {
    // 未知页面：返回尾段本身（而不是编造成「首页」或留空）
    expect(resolveVisitTitle('/unknown-page', labels)).toBe('unknown-page');
  });

  it('过滤登录/鉴权类路由，不把它们记进最近访问', () => {
    expect(shouldTrackVisit('/login')).toBe(false);
    expect(shouldTrackVisit('/auth/callback')).toBe(false);
    expect(shouldTrackVisit('/')).toBe(false);
    expect(shouldTrackVisit('/tickets')).toBe(true);
  });

  it('🔴 不再维护第二份硬编码 pathMeta（防止旧实现被复制回来）', () => {
    const tracker = fs.readFileSync(
      path.join(FRONTEND_ROOT, 'src/components/layout/RecentVisitTracker.tsx'),
      'utf8'
    );
    // 旧实现是一张 Record<string,{title,icon}> 常量表；标题必须来自菜单
    expect(tracker).not.toMatch(/const\s+pathMeta\s*[:=]/);
    expect(tracker).toContain('collectPathLabels');
  });
});
