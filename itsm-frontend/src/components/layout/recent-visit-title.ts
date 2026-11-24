'use client';

/**
 * 最近访问记录：把「用户去过哪些页面」这件事收敛成一个可测试的纯函数模块。
 *
 * 背景：此前 `RecentVisitTracker` 只在每个页面 `addVisit` 写 localStorage，
 * 全仓**没有任何一处读取 visits** —— 一段只写不读的死代码；同时它的
 * `pathMeta` 硬编码了 11 条标题，其余 90 条菜单会回退成 URL 尾段英文
 * （`workbench`、`service-requests`、`noc`…），即使将来接上也显示错误的名字。
 *
 * 现在改成两个明确的职责：
 *  1. `resolveVisitTitle(pathname, menuLabels)` —— 页面标题从菜单单一事实源取，
 *     取不到才退回 URL 段翻译，不再维护第二份硬编码表。
 *  2. `RecentVisitTracker` 只负责写，`GlobalSearch` 的空态负责读，
 *     形成真正的闭环（写有消费方，才不是死代码）。
 */

import type { MenuItem as ApiMenuItem } from '@/lib/api/menu-api';

/** 这些路径不算「业务页面」，不记入最近访问。 */
const IGNORED_PREFIXES = ['/login', '/auth', '/register', '/reset-password'];

/** 路径 → 菜单名（含深层路径的最长前缀匹配用的扁平映射）。 */
export function collectPathLabels(
  items: ApiMenuItem[],
  map: Map<string, string> = new Map()
): Map<string, string> {
  for (const item of items) {
    if (item.path && item.name) map.set(item.path, item.name);
    if (item.children) collectPathLabels(item.children, map);
  }
  return map;
}

/**
 * 解析一个路径的显示标题。
 *
 * 优先级：
 *  1. 菜单里的精确条目名（单一事实源，中英文已本地化）
 *  2. 菜单里的最长前缀条目名 + 剩余段（如 /tickets/12 → 「工单管理 · 详情 #12」）
 *  3. URL 段翻译兜底（不在菜单里的页面）
 */
export function resolveVisitTitle(pathname: string, menuLabels: Map<string, string>): string {
  const exact = menuLabels.get(pathname);
  if (exact) return exact;

  let bestPrefix = '';
  for (const key of menuLabels.keys()) {
    if (pathname.startsWith(`${key}/`) && key.length > bestPrefix.length) {
      bestPrefix = key;
    }
  }
  if (bestPrefix) {
    const rest = pathname.slice(bestPrefix.length).split('/').filter(Boolean);
    const label = menuLabels.get(bestPrefix)!;
    if (rest.length === 0) return label;
    const tail = rest[rest.length - 1];
    const tailTitle = /^\d+$/.test(tail) ? `详情 #${tail}` : tail;
    return `${label} · ${tailTitle}`;
  }

  // 兜底：URL 最后一段。用中文占位而不是英文段名——英文段名对业务用户无意义，
  // 且会与真正的菜单名混在一起让人以为是两个页面。
  const segments = pathname.split('/').filter(Boolean);
  if (segments.length === 0) return '首页';
  const last = segments[segments.length - 1];
  return /^\d+$/.test(last) ? `详情 #${last}` : last;
}

/** 是否应记入最近访问（过滤登录/鉴权类路由）。 */
export function shouldTrackVisit(pathname: string): boolean {
  if (!pathname || pathname === '/') return false;
  return !IGNORED_PREFIXES.some(prefix => pathname === prefix || pathname.startsWith(`${prefix}/`));
}
