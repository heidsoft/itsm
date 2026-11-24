/**
 * 菜单「归属一致性」回归锁（面包屑 vs 侧栏）。
 *
 * 背景：菜单做过业务归位，有 29 个页面的 URL 不在其父级路径之下，例如
 *   /admin/ticket-categories  挂在「工单管理」下（不是「系统管理」）
 *   /admin/sla-templates      挂在「SLA 管理」下
 *   /admin/service-catalogs   挂在「服务目录」下
 *   /admin/cmdb-types         挂在「配置管理」下
 *   /ai/approval              挂在「我的待办」下
 *
 * 面包屑原先按 URL 段逐段翻译，于是 /admin/ticket-categories 渲染成
 * 「首页 / 系统管理 / 工单分类」，而侧栏把它显示在「工单管理 → 工单分类」。
 * 同一个页面在两个常驻位置给出两个不同的上级。
 *
 * 本测试**直接调用真实实现** `resolveBreadcrumb`（不是自写副本）：
 * 曾经的写法是测试里手抄一份算法，把真实实现短路后测试照样全绿（已实测），
 * 那种「测试证明自己」的形态没有价值。
 */

import fs from 'node:fs';
import path from 'node:path';

import { resolveBreadcrumb } from '@/components/layout/header/breadcrumb-utils';
import type { MenuItem as ApiMenuItem } from '@/lib/api/menu-api';

const FRONTEND_ROOT = process.cwd();
const REPO_ROOT = path.resolve(FRONTEND_ROOT, '..');
const BASELINE = path.join(REPO_ROOT, 'itsm-backend/pkg/menubaseline/baseline.go');

interface MenuSpec {
  name: string;
  path: string;
  parentPath: string;
}

/** 从 baseline.go 解析菜单条目（Name / Path / ParentPath）。 */
export function menuSpecs(): MenuSpec[] {
  const src = fs.readFileSync(BASELINE, 'utf8');
  // 不用 `s` 标志（tsconfig target 为 es2017，正则 dotAll 会报 TS1501），
  // 改为显式的 [\s\S] 匹配跨行内容。
  const re = /\{Name:\s*"([^"]+)",\s*Path:\s*"([^"]+)",\s*Icon:\s*"[^"]*",([\s\S]*?)\}/g;
  const out: MenuSpec[] = [];
  for (const m of src.matchAll(re)) {
    const rest = m[3];
    const parent =
      /ParentPath:\s*"([^"]*)"/.exec(rest)?.[1] ??
      (/ParentPath:\s*([A-Za-z]+)/.exec(rest)?.[1] ?? '');
    out.push({ name: m[1], path: m[2], parentPath: parent });
  }
  return out;
}

/** 把 baseline 的扁平清单还原成「main / admin」两棵树，模拟后端 /auth/menus 的返回。 */
export function buildMenuTrees(specs: MenuSpec[]): {
  main: ApiMenuItem[];
  admin: ApiMenuItem[];
} {
  const ADMIN_ROOTS = ['/admin', '/system', '/workflow'];
  const byPath = new Map<string, ApiMenuItem>();

  for (const spec of specs) {
    byPath.set(spec.path, {
      id: 0,
      name: spec.name,
      path: spec.path,
      sortOrder: 0,
      tenantId: 1,
      isVisible: true,
      isEnabled: true,
      children: [],
    });
  }
  for (const spec of specs) {
    if (!spec.parentPath) continue;
    const parent = byPath.get(spec.parentPath);
    const child = byPath.get(spec.path);
    if (parent && child) parent.children!.push(child);
  }

  const roots = specs.filter(s => !s.parentPath).map(s => byPath.get(s.path)!);
  const isAdminRoot = (p: string) => ADMIN_ROOTS.some(a => p === a || p.startsWith(`${a}/`));
  return {
    main: roots.filter(r => !isAdminRoot(r.path)),
    admin: roots.filter(r => isAdminRoot(r.path)),
  };
}

/**
 * 侧栏的祖先链名（从顶到底）。
 *
 * `/dashboard`（服务台）这一级要剔掉：面包屑的第一项固定是「首页」且 href 就是
 * `/dashboard`，同一个位置用「首页」表示；再要求面包屑出现「服务台」等于要求
 * 它自我重复。这条口径由菜单树结构独立推出，不依赖被测实现。
 */
function sidebarAncestorNames(specs: MenuSpec[], target: string): string[] {
  const chain: string[] = [];
  let cursor = specs.find(s => s.path === target);
  const guard = new Set<string>();
  while (cursor?.parentPath && !guard.has(cursor.parentPath)) {
    guard.add(cursor.parentPath);
    const parent = specs.find(s => s.path === cursor!.parentPath);
    if (!parent) break;
    if (parent.path !== '/dashboard') chain.unshift(parent.name);
    cursor = parent;
  }
  return chain;
}

/** 取面包屑的中间各项（去掉「首页」与末项自身）。 */
function crumbMiddle(titles: string[]): string[] {
  return titles.slice(1, -1);
}

describe('菜单归属一致性：面包屑必须与侧栏给出同一套上级', () => {
  const specs = menuSpecs();
  const trees = buildMenuTrees(specs);

  it('基线里确实存在「URL 前缀与父级不符」的归位条目（否则本测试无意义）', () => {
    const relocated = specs.filter(
      s =>
        s.parentPath &&
        !s.path.startsWith(`${s.parentPath}/`) &&
        !s.path.startsWith(`${s.parentPath}-`)
    );
    // 精确计数而不是 >=：归位条目的数量是这条判据的覆盖面，写成 >=10 会让
    // 「新增若干归位页但面包屑没跟上」这类回归悄悄通过。
    // 当前 29 条（/workbench、/noc、/admin/ticket-categories、/ai/approval …）。
    expect(relocated.length).toBe(29);
  });

  it('每一个归位条目的面包屑上级链都等于其侧栏祖先链', () => {
    const relocated = specs.filter(
      s =>
        s.parentPath &&
        !s.path.startsWith(`${s.parentPath}/`) &&
        !s.path.startsWith(`${s.parentPath}-`)
    );

    const mismatches: string[] = [];
    for (const spec of relocated) {
      const expected = sidebarAncestorNames(specs, spec.path);
      const crumb = resolveBreadcrumb(spec.path, trees).map(i => String(i.title ?? ''));
      const middle = crumbMiddle(crumb);
      if (JSON.stringify(middle) !== JSON.stringify(expected)) {
        mismatches.push(
          `${spec.path}: 侧栏上级=[${expected.join(',')}] 面包屑上级=[${middle.join(',')}]`
        );
      }
    }
    expect(mismatches).toEqual([]);
  });

  it('典型归位页的面包屑里不得出现「系统管理」这个错误上级', () => {
    // 反向场景：把 resolveBreadcrumb 的菜单祖先链分支去掉、退回 URL 段翻译，
    // /admin/ticket-categories 立刻变回「系统管理 / 工单分类」，本断言即红。
    const cases = [
      '/admin/ticket-categories',
      '/admin/service-catalogs',
      '/admin/cmdb-types',
      '/admin/sla-templates',
      '/admin/marketplace',
    ];
    for (const target of cases) {
      const crumb = resolveBreadcrumb(target, trees).map(i => String(i.title ?? ''));
      expect(crumb).not.toContain('系统管理');
    }
  });

  it('末项是页面自己的名字，不是 URL 尾段', () => {
    const crumb = resolveBreadcrumb('/admin/ticket-categories', trees).map(i => String(i.title ?? ''));
    expect(crumb[crumb.length - 1]).toBe('工单分类');
  });

  it('不在菜单里的深层路径仍能生成合理面包屑（回退分支未被破坏）', () => {
    // /tickets/12 —— 详情页，菜单里没有这条，须回退到「最长的菜单前缀 + 剩余段」
    const crumb = resolveBreadcrumb('/tickets/12', trees).map(i => String(i.title ?? ''));
    expect(crumb[0]).toBe('首页');
    expect(crumb).toContain('工单管理');
    expect(crumb[crumb.length - 1]).toBe('详情 #12');
  });
});
