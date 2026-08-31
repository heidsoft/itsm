/**
 * 菜单类型定义
 *
 * 菜单数据完全由后端提供（GET /api/v1/auth/menus → itsm-backend/pkg/seeder seedMenus），
 * 前端不再维护静态菜单配置。本文件仅保留 Sidebar 渲染所需的 MenuItem 类型，
 * 以及 capability 与 path 的对应关系（用于能力治理矩阵）。
 *
 * 如需在运行时获取菜单，请使用：
 *   import { useUserMenusQuery } from '@/lib/hooks/useUserMenusQuery';
 *   const { data: menus } = useUserMenusQuery();
 */

import type { ReactNode } from 'react';

// Sidebar 内部使用的菜单项接口，与 lib/api/menu-api.MenuItem 对应但允许 ReactNode
// 形式的 icon/label，便于 AntD Menu 渲染。
export interface MenuItem {
  key: string;
  icon?: ReactNode;
  label: string | ReactNode;
  path?: string;
  permission?: string;
  description?: string;
  badge?: string;
  capabilityKey?: string;
  children?: MenuItem[];
}

// Capability 与菜单路径的映射：用于 capability 治理（disabled / unready 屏蔽入口）
// 与菜单的实际 list/children 解耦，新增菜单只需补充路径规则即可，无需触碰后端。
export const capabilityPathRules: Array<[string, string]> = [
  ['/service-requests', 'serviceRequest'],
  ['/incidents', 'incident'],
  ['/problems', 'problem'],
  ['/changes', 'change'],
  ['/knowledge', 'knowledge'],
  ['/cmdb', 'cmdb'],
  ['/sla', 'sla'],
  ['/workflow', 'workflow'],
  ['/ai', 'ai'],
  ['/marketplace', 'marketplace'],
  ['/installations', 'marketplace'],
  ['/admin/connectors', 'marketplace'],
];

export function capabilityForPath(path?: string): string | undefined {
  if (!path) return undefined;
  return capabilityPathRules.find(([prefix]) => path === prefix || path.startsWith(`${prefix}/`))?.[1];
}
