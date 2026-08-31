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
  dashboard: '服务台',
  tickets: '工单管理',
  incidents: '事件管理',
  problems: '问题管理',
  changes: '变更管理',
  releases: '发布管理',
  knowledge: '知识库',
  articles: '文章',
  reviews: '审核',
  'service-catalog': '服务目录',
  'service-requests': '服务请求',
  'my-requests': '我的请求',
  cmdb: 'CMDB',
  cis: '配置项',
  ci: '配置项',
  'ci-types': 'CI 类型',
  'cloud-accounts': '云账号',
  'cloud-resources': '云资源',
  'cloud-services': '云服务',
  reconciliation: '同步对账',
  registry: '注册表',
  relationships: 'CI 关系',
  topology: '拓扑图',
  assets: '资产管理',
  licenses: '软件许可证',
  sla: 'SLA管理',
  'sla-dashboard': 'SLA仪表盘',
  'sla-monitor': 'SLA监控',
  definitions: 'SLA 定义',
  'sla-definitions': 'SLA 定义',
  'sla-templates': 'SLA 模板',
  'ticket-categories': '工单分类',
  'escalation-rules': '升级规则',
  'escalation-matrices': '升级矩阵',
  workflow: '工作流',
  workflows: '工作流',
  designer: '流程设计器',
  instances: '流程实例',
  versions: '版本管理',
  automation: '自动化规则',
  bottlenecks: '瓶颈分析',
  audit: '操作日志',
  'ticket-approval': '审批中心',
  reports: '报表中心',
  admin: '系统管理',
  system: '系统管理',
  enterprise: '组织管理',
  users: '用户管理',
  roles: '角色管理',
  permissions: '权限管理',
  groups: '组管理',
  teams: '团队管理',
  departments: '部门管理',
  tenants: '租户管理',
  menus: '菜单管理',
  connectors: '连接器',
  notifications: '通知中心',
  profile: '个人中心',
  approvals: '审批',
  pending: '待我审批',
  msp: '客户管理',
  ai: 'AI助手',
  chat: 'AI对话',
  'ai-create': 'AI创建工单',
  analytics: '统计分析',
  templates: '模板管理',
  types: '类型管理',
  cc: '抄送我的',
  'known-errors': '已知错误',
  trends: '趋势分析',
  pir: '实施后评审',
  pirs: '实施后评审',
  'standard-changes': '标准变更',
  detail: '详情',
  request: '发起请求',
  create: '新建',
  new: '新建',
  edit: '编辑',
  list: '列表',
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
  const items: BreadcrumbItem[] = [{ title: '首页', href: '/dashboard' }];
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
      title = /^\d+$/.test(segment) ? `详情 #${segment}` : segment;
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
  const items: BreadcrumbItem[] = [{ title: '首页', href: '/dashboard' }];
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
      title = /^\d+$/.test(segment) ? `详情 #${segment}` : segment;
    }

    items.push(isLast ? { title } : { title, href: acc });
  });

  return items;
}
