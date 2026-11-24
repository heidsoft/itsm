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

/**
 * 业务归位后的精确路径面包屑（优先于逐段兜底）。
 *
 * 四个 AI 能力页的 URL 仍是历史技术路径 /ai/*（页面路由零改动），但菜单已按业务
 * 归位到工单管理 / 我的待办 / 审计日志。若继续按 URL 段逐段翻译，/ai/audit 会显示
 * 「AI 助手 / 操作日志」（那是 /workflow/audit 的段名），与菜单叫法自相矛盾；
 * /ai/chat 也会出现「AI 助手 / AI 助手」重复段。精确匹配直接给业务面包屑，
 * 不暴露已下线的一级技术聚合层。
 */
const BUSINESS_PATH_BREADCRUMBS: Record<string, BreadcrumbItem[]> = {
  '/ai/chat': [{ title: 'AI 助手' }],
  '/tickets/ai-create': [
    { title: '工单管理', href: '/tickets' },
    { title: 'AI 创建工单' },
  ],
  '/ai/approval': [
    { title: '我的待办', href: '/approvals' },
    { title: '工具调用审批' },
  ],
  '/ai/audit': [
    { title: '审计日志', href: '/audit-logs' },
    { title: 'AI 效果评估' },
  ],
};

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
  'cloud-signals': '云安全与告警',
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
  'escalation-rules': '超时升级',
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
  ai: 'AI 助手',
  chat: 'AI 助手',
  'ai-create': 'AI 创建工单',
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
 * 递归收集 path → 祖先路径数组。
 *
 * 为什么需要它：菜单做过业务归位，有些子页面的 URL 并不在父级路径之下
 * （/admin/ticket-categories 挂在「工单管理」下、/admin/sla-templates 挂在
 * 「SLA 管理」下、/ai/approval 挂在「我的待办」下，共 29 处）。若面包屑按
 * URL 段逐段翻译，/admin/ticket-categories 会渲染成「系统管理 / 工单分类」，
 * 与侧栏把它归在「工单管理」下自相矛盾——同一个页面在两个地方给出两个上级。
 *
 * 菜单树就是归属的权威源，因此优先按菜单的父子关系生成祖先链，只有不在菜单
 * 里的路径才退回 URL 段拆分。
 */
function collectMenuParents(
  items: ApiMenuItem[],
  map: Map<string, ApiMenuItem[]>,
  ancestors: ApiMenuItem[] = []
) {
  for (const item of items) {
    if (item.path) {
      map.set(item.path, ancestors);
    }
    if (item.children) {
      collectMenuParents(item.children, map, item.path ? [...ancestors, item] : ancestors);
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

  const exact = BUSINESS_PATH_BREADCRUMBS[pathname];
  if (exact) return [...items, ...exact];

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
 * 纯函数版面包屑计算：不依赖 React，便于测试直接断言真实实现。
 *
 * 归属优先级：菜单树祖先链 > URL 段拆分。
 * 当目标路径是菜单条目时，用菜单里的父子关系生成祖先（中间层可点击），
 * 保证面包屑与侧栏对同一个页面给出同一套上级；只有不在菜单里的路径
 *（详情页、深层子路径等）才回退到 URL 段翻译。
 *
 * 之所以抽成导出的纯函数而不是内联在 hook 里：此前的回归测试自己手写了一份
 * 「面包屑算法副本」来比对菜单树，结果**真实实现失效时测试仍然全绿**
 * （已实测：把 hook 里的祖先链分支短路成 false，副本测试照样通过）。
 */
export function resolveBreadcrumb(
  pathname: string,
  menus: { main: ApiMenuItem[]; admin: ApiMenuItem[] } | undefined
): BreadcrumbItem[] {
  const items: BreadcrumbItem[] = [{ title: '首页', href: '/dashboard' }];
  if (!pathname || pathname === '/' || pathname === '/dashboard') {
    return items;
  }

  const exact = BUSINESS_PATH_BREADCRUMBS[pathname];
  if (exact) return [...items, ...exact];

  const labelMap = new Map<string, string>();
  const parentMap = new Map<string, ApiMenuItem[]>();
  if (menus) {
    collectMenuLabels(menus.main, labelMap);
    collectMenuLabels(menus.admin, labelMap);
    collectMenuParents(menus.main, parentMap);
    collectMenuParents(menus.admin, parentMap);
  }

  // ① 找出与 pathname 匹配的最长菜单路径（精确命中或路径前缀）。
  //
  // 这里刻意只有一个分支：此前曾把「精确命中」和「深层子路径」写成两个并列
  // 分支，二者的祖先链生成逻辑完全相同——精确命中只是「剩余段为空」的特例。
  // 两个分支并存时，把其中一个短路掉测试也不会红（另一个会给出同样结果），
  // 于是回归锁实际上保护不到任何一条，反向验证也拿不到真实信号。
  let bestPrefix = '';
  for (const key of labelMap.keys()) {
    if ((pathname === key || pathname.startsWith(`${key}/`)) && key.length > bestPrefix.length) {
      bestPrefix = key;
    }
  }

  // ② 用菜单树作为归属的权威源：祖先链来自菜单父子关系，不按 URL 段猜。
  if (bestPrefix) {
    for (const ancestor of parentMap.get(bestPrefix) ?? []) {
      if (!ancestor.path || ancestor.path === '/dashboard') continue;
      items.push({ title: ancestor.name, href: ancestor.path });
    }
    items.push({ title: labelMap.get(bestPrefix)! });

    // 菜单条目之下的深层路径（/tickets/3、/cmdb/cis/create）追加剩余段
    const rest = pathname.slice(bestPrefix.length).split('/').filter(Boolean);
    let acc = bestPrefix;
    rest.forEach((segment, index) => {
      acc += `/${segment}`;
      const isLast = index === rest.length - 1;
      const title = /^\d+$/.test(segment) ? `详情 #${segment}` : SEGMENT_LABELS[segment] || segment;
      items.push(isLast ? { title } : { title, href: acc });
    });
    return items;
  }

  // ③ 不在菜单里的路径：退回 URL 段翻译
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

/**
 * Hook 版本：基于动态菜单（后端 /api/v1/auth/menus）生成面包屑。
 * 用于顶部 Header 等需要结合菜单 label 渲染面包屑的场景，
 * 内部与 Sidebar 共享 useUserMenusQuery 的 React Query 缓存。
 */
export function useBuildBreadcrumb(pathname: string): BreadcrumbItem[] {
  const { data: menus } = useUserMenusQuery();
  return resolveBreadcrumb(pathname, menus);
}
