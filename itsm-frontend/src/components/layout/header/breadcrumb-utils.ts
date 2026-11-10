/**
 * 面包屑自动生成工具
 *
 * 基于 usePathname 的路由段，优先匹配动态菜单中的 label（后端 /api/v1/auth/menus 提供），
 * 其次匹配通用路由段语义映射，动态 ID 段显示为「详情 #id」。
 *
 * 菜单数据完全由后端 seedMenus 提供，前端不再依赖 menu-config.ts。
 * 通过 useUserMenusQuery 与 Sidebar 共享 React Query 缓存，避免重复请求。
 */

import type { BreadcrumbProps } from 'antd';

import { useUserMenusQuery } from '@/lib/hooks/useUserMenusQuery';
import type { MenuItem as ApiMenuItem } from '@/lib/api/menu-api';

export type BreadcrumbItem = NonNullable<BreadcrumbProps['items']>[number];

// 通用路由段语义映射（菜单配置未覆盖的路径段兜底）
const SEGMENT_LABELS: Record<string, string> = {
  dashboard: 'Help Desk',
  tickets: 'Ticket Management',
  incidents: 'Incidents',
  problems: 'Problems',
  changes: 'Changes',
  releases: 'Release Management',
  knowledge: 'Knowledge Base',
  articles: 'Article',
  reviews: 'Review',
  'service-catalog': 'Service Catalog',
  'service-requests': 'Service Request',
  'my-requests': 'My Requests',
  cmdb: 'CMDB',
  cis: 'Configuration item',
  ci: 'Configuration item',
  'ci-types': 'CI Types',
  'cloud-accounts': 'Cloud account',
  'cloud-resources': 'Cloud Resources',
  'cloud-services': 'Cloud services',
  'cloud-signals': 'Cloud Security and Alerting',
  reconciliation: 'Synchronous reconciliation',
  registry: 'Registry',
  relationships: 'CI relationship',
  topology: 'Topology map',
  assets: 'Asset Management',
  licenses: 'Software License',
  sla: 'SLA Management',
  'sla-dashboard': 'SLA Dashboard',
  'sla-monitor': 'SLA Monitor',
  definitions: 'SLA Definition',
  'sla-definitions': 'SLA Definition',
  'sla-templates': 'SLA Template',
  'ticket-categories': 'Ticket Categories',
  'escalation-rules': 'SLA Alert Rule',
  'escalation-matrices': 'Upgrade Matrix',
  workflow: 'Workflows',
  workflows: 'Workflows',
  designer: 'Process Designer',
  instances: 'Process Instances',
  versions: 'Version Management',
  automation: 'Automation Rules',
  bottlenecks: 'Bottleneck Analysis',
  audit: 'Operation log',
  'ticket-approval': 'Approval Center',
  reports: 'Report Center',
  admin: 'System Admin',
  system: 'System Admin',
  enterprise: 'Organizational Management',
  users: 'User Management',
  roles: 'Role Management',
  permissions: 'Permission management',
  groups: 'Group Management',
  teams: 'Team Management',
  departments: 'Department Management',
  tenants: 'Tenant Management',
  menus: 'Menu Management',
  connectors: 'Connectors',
  notifications: 'Notification Center',
  profile: 'Profile',
  approvals: 'Approval',
  pending: 'Pending Approvals',
  msp: 'Customer Management',
  ai: 'AI Assistant',
  chat: 'AI dialogue',
  'ai-create': 'AI creates work order',
  analytics: 'Statistical analysis',
  templates: 'Template management',
  types: 'Type management',
  cc: 'CC me',
  'known-errors': 'Known Error',
  trends: 'Trend Analysis',
  pir: 'Post-Implementation Review',
  pirs: 'Post-Implementation Review',
  'standard-changes': 'Standard Change',
  detail: 'Details',
  request: 'Initiate request',
  create: 'Create',
  new: 'Create',
  edit: 'Edit',
  list: 'List',
};

// 递归收集菜单配置中 path → label 的映射（label 为字符串时）
function collectMenuLabels(items: ApiMenuItem[], map: Map<string, string>) {
  for (const item of items) {
    if (item.path && item.name) {
      map.set(item.path, item.name);
    }
    if (item.children) {
      collectMenuLabels(item.children, map);
    }
  }
}

/**
 * 根据 pathname 自动生成面包屑
 * 例：/incidents/12/edit → 首页 / 事件管理 / 详情 #12 / 编辑
 */
export function buildBreadcrumb(pathname: string): BreadcrumbItem[] {
  const items: BreadcrumbItem[] = [{ title: 'Home page', href: '/dashboard' }];
  if (!pathname || pathname === '/' || pathname === '/dashboard') {
    return items;
  }

  const segments = pathname.split('/').filter(Boolean);
  let acc = '';

  segments.forEach((segment, index) => {
    acc += `/${segment}`;
    // /dashboard 已由「首页」表示，跳过避免重复
    if (acc === '/dashboard') return;

    const isLast = index === segments.length - 1;
    let title = SEGMENT_LABELS[segment];
    if (!title) {
      // 动态段：纯数字 ID 显示为详情，其余（uuid/slug）原样展示
      title = /^\d+$/.test(segment) ? `Details #${segment}` : segment;
    }

    items.push(isLast ? { title } : { title, href: acc });
  });

  return items;
}

/**
 * Hook 版本：基于动态菜单（后端 /api/v1/auth/menus）生成面包屑。
 * 用于顶部 Header 等需要结合菜单 label 渲染面包屑的场景，
 * 内部与 Sidebar 共享 useUserMenusQuery 的 React Query 缓存。
 */
export function useBuildBreadcrumb(pathname: string): BreadcrumbItem[] {
  const { data: menus } = useUserMenusQuery();
  const items: BreadcrumbItem[] = [{ title: 'Home page', href: '/dashboard' }];
  if (!pathname || pathname === '/' || pathname === '/dashboard') {
    return items;
  }

  const labelMap = new Map<string, string>();
  if (menus) {
    collectMenuLabels(menus.main, labelMap);
    collectMenuLabels(menus.admin, labelMap);
  }

  const segments = pathname.split('/').filter(Boolean);
  let acc = '';

  segments.forEach((segment, index) => {
    acc += `/${segment}`;
    if (acc === '/dashboard') return;

    const isLast = index === segments.length - 1;
    let title = labelMap.get(acc) || SEGMENT_LABELS[segment];
    if (!title) {
      title = /^\d+$/.test(segment) ? `Details #${segment}` : segment;
    }

    items.push(isLast ? { title } : { title, href: acc });
  });

  return items;
}
