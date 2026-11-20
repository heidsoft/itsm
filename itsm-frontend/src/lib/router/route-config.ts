/**
 * 路由配置和权限管理
 *
 * 权限码契约（2026-11-14 权限码单点化批次）：
 * - 路由权限码一律通过 rp(PERM.x.y) 声明，PERM 来自后端码空间生成物；
 *   引用码空间不存在的码会在 TypeScript 编译期失败（幽灵码零容忍）；
 * - 每个页面挂的码 = 后端对应路由实际 RequirePermission 的码，
 *   不再使用 manage/view 等前端臆造动作，也不再需要别名映射。
 */

import { PERM } from '@/lib/permissions';
import type { PermissionCode } from '@/lib/permissions';
import { matchesPermission } from '@/lib/permissions/utils';

// 路由权限类型
export interface RoutePermission {
  resource: string;
  action: string;
}

/**
 * 把生成的权限码构造成路由权限声明。
 * 这是路由配置里权限码的唯一合法写法（compile-time 校验）。
 */
const rp = (code: PermissionCode): RoutePermission[] => {
  const separatorIndex = code.indexOf(':');
  if (separatorIndex <= 0 || separatorIndex === code.length - 1) {
    throw new Error(`非法权限码: ${code}`);
  }
  return [
    {
      resource: code.slice(0, separatorIndex),
      action: code.slice(separatorIndex + 1),
    },
  ];
};

// 路由配置接口
export interface RouteConfig {
  path: string;
  name: string;
  title: string;
  icon?: string;
  component?: string;
  children?: RouteConfig[];
  permissions?: RoutePermission[];
  hidden?: boolean;
  redirect?: string;
  meta?: {
    requireAuth?: boolean;
    roles?: string[];
    keepAlive?: boolean;
    breadcrumb?: boolean;
  };
}

// 主要路由配置
export const routes: RouteConfig[] = [
  {
    path: '/',
    name: 'dashboard',
    title: '仪表盘',
    icon: 'LayoutDashboard',
    component: 'Dashboard',
    permissions: rp(PERM.dashboard.read),
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
  },
  {
    path: '/tickets',
    name: 'tickets',
    title: '工单管理',
    icon: 'Ticket',
    permissions: rp(PERM.ticket.read),
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/tickets',
        name: 'ticket-list',
        title: '工单列表',
        component: 'TicketList',
        permissions: rp(PERM.ticket.read),
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/tickets/create',
        name: 'ticket-create',
        title: '创建工单',
        component: 'TicketCreate',
        permissions: rp(PERM.ticket.create),
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/tickets/:id',
        name: 'ticket-detail',
        title: '工单详情',
        component: 'TicketDetail',
        permissions: rp(PERM.ticket.read),
        hidden: true,
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/tickets/:id/edit',
        name: 'ticket-edit',
        title: '编辑工单',
        component: 'TicketEdit',
        permissions: rp(PERM.ticket.update),
        hidden: true,
        meta: {
          requireAuth: true,
        },
      },
    ],
  },
  {
    path: '/incidents',
    name: 'incidents',
    title: '事件管理',
    icon: 'AlertTriangle',
    permissions: rp(PERM.incident.read),
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/incidents',
        name: 'incident-list',
        title: '事件列表',
        component: 'IncidentList',
        permissions: rp(PERM.incident.read),
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/incidents/create',
        name: 'incident-create',
        title: '创建事件',
        component: 'IncidentCreate',
        permissions: rp(PERM.incident.write),
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/incidents/:id',
        name: 'incident-detail',
        title: '事件详情',
        component: 'IncidentDetail',
        permissions: rp(PERM.incident.read),
        hidden: true,
        meta: {
          requireAuth: true,
        },
      },
    ],
  },
  {
    path: '/problems',
    name: 'problems',
    title: '问题管理',
    icon: 'Bug',
    permissions: rp(PERM.problem.read),
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/problems',
        name: 'problem-list',
        title: '问题列表',
        component: 'ProblemList',
        permissions: rp(PERM.problem.read),
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/problems/create',
        name: 'problem-create',
        title: '创建问题',
        component: 'ProblemCreate',
        permissions: rp(PERM.problem.write),
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/problems/:id',
        name: 'problem-detail',
        title: '问题详情',
        component: 'ProblemDetail',
        permissions: rp(PERM.problem.read),
        hidden: true,
        meta: {
          requireAuth: true,
        },
      },
    ],
  },
  {
    path: '/releases',
    name: 'releases',
    title: '发布管理',
    icon: 'Rocket',
    permissions: rp(PERM.release.read),
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/releases',
        name: 'release-list',
        title: '发布列表',
        component: 'ReleaseList',
        permissions: rp(PERM.release.read),
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/releases/new',
        name: 'release-create',
        title: '创建发布',
        component: 'ReleaseForm',
        permissions: rp(PERM.release.write),
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/releases/:id',
        name: 'release-detail',
        title: '发布详情',
        component: 'ReleaseDetail',
        permissions: rp(PERM.release.read),
        hidden: true,
        meta: {
          requireAuth: true,
        },
      },
    ],
  },
  {
    path: '/changes',
    name: 'changes',
    title: '变更管理',
    icon: 'GitBranch',
    permissions: rp(PERM.change.read),
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/changes',
        name: 'change-list',
        title: '变更列表',
        component: 'ChangeList',
        permissions: rp(PERM.change.read),
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/changes/new',
        name: 'change-create',
        title: '创建变更',
        component: 'ChangeCreate',
        permissions: rp(PERM.change.write),
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/changes/:id',
        name: 'change-detail',
        title: '变更详情',
        component: 'ChangeDetail',
        permissions: rp(PERM.change.read),
        hidden: true,
        meta: {
          requireAuth: true,
        },
      },
    ],
  },
  {
    path: '/assets',
    name: 'assets',
    title: '资产管理',
    icon: 'Desktop',
    permissions: rp(PERM.asset.read),
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/assets',
        name: 'asset-list',
        title: '资产列表',
        component: 'AssetList',
        permissions: rp(PERM.asset.read),
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/assets/new',
        name: 'asset-create',
        title: '创建资产',
        component: 'AssetForm',
        permissions: rp(PERM.asset.write),
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/assets/:id',
        name: 'asset-detail',
        title: '资产详情',
        component: 'AssetDetail',
        permissions: rp(PERM.asset.read),
        hidden: true,
        meta: {
          requireAuth: true,
        },
      },
    ],
  },
  {
    path: '/licenses',
    name: 'licenses',
    title: '许可证管理',
    icon: 'Key',
    permissions: rp(PERM.license.read),
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/licenses',
        name: 'license-list',
        title: '许可证列表',
        component: 'LicenseList',
        permissions: rp(PERM.license.read),
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/licenses/new',
        name: 'license-create',
        title: '创建许可证',
        component: 'LicenseForm',
        permissions: rp(PERM.license.write),
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/licenses/:id',
        name: 'license-detail',
        title: '许可证详情',
        component: 'LicenseDetail',
        permissions: rp(PERM.license.read),
        hidden: true,
        meta: {
          requireAuth: true,
        },
      },
    ],
  },
  {
    path: '/knowledge',
    name: 'knowledge',
    title: '知识库',
    icon: 'BookOpen',
    permissions: rp(PERM.knowledge.read),
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/knowledge',
        name: 'knowledge-list',
        title: '知识文章',
        component: 'KnowledgeList',
        permissions: rp(PERM.knowledge.read),
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/knowledge/create',
        name: 'knowledge-create',
        title: '创建文章',
        component: 'KnowledgeCreate',
        permissions: rp(PERM.knowledge.write),
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/knowledge/:id',
        name: 'knowledge-detail',
        title: '文章详情',
        component: 'KnowledgeDetail',
        permissions: rp(PERM.knowledge.read),
        hidden: true,
        meta: {
          requireAuth: true,
        },
      },
    ],
  },
  // 2026-11-21 方案 A：原「智能助手」(/ai) + 「AI 问答」(/ai/chat) 死配置已删除。
  // 菜单完全由后端 /api/v1/auth/menus 驱动；问答入口是顶栏全局「AI 助手」，
  // 三个业务能力页的可见性由后端菜单 + ai capability 门控决定。
  // 旧名「智能助手」带机械前缀，留在配置里就是命名噪声；页面与路由零改动。
  {
    path: '/cmdb',
    name: 'cmdb',
    title: '配置管理',
    icon: 'Database',
    permissions: rp(PERM.cmdb.read),
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/cmdb/cis',
        name: 'ci-list',
        title: '配置项',
        component: 'CIList',
        permissions: rp(PERM.cmdb.read),
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/cmdb/ci-types',
        name: 'ci-type-list',
        title: '配置项类型',
        component: 'CITypeList',
        permissions: rp(PERM.cmdb.write),
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/cmdb/relationships',
        name: 'ci-relationship-list',
        title: '关系管理',
        component: 'CIRelationshipList',
        permissions: rp(PERM.cmdb.read),
        meta: {
          requireAuth: true,
        },
      },
    ],
  },
  {
    path: '/reports',
    name: 'reports',
    title: '报表分析',
    icon: 'BarChart3',
    permissions: rp(PERM.report.read),
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/reports/dashboard',
        name: 'report-dashboard',
        title: '报表仪表盘',
        component: 'ReportDashboard',
        permissions: rp(PERM.report.read),
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/reports/tickets',
        name: 'ticket-reports',
        title: '工单报表',
        component: 'TicketReports',
        permissions: rp(PERM.report.read),
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/reports/sla',
        name: 'sla-reports',
        title: 'SLA报表',
        component: 'SLAReports',
        permissions: rp(PERM.report.read),
        meta: {
          requireAuth: true,
        },
      },
    ],
  },
  {
    path: '/admin',
    name: 'admin',
    title: '系统管理',
    icon: 'Settings',
    permissions: rp(PERM.system_config.read),
    meta: {
      requireAuth: true,
      roles: ['admin', 'super_admin'],
      breadcrumb: true,
    },
    children: [
      {
        path: '/admin',
        name: 'admin-overview',
        title: '系统概览',
        component: 'AdminDashboard',
        permissions: rp(PERM.system_config.read),
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin'],
          keepAlive: true,
        },
      },
      {
        path: '/admin/users',
        name: 'user-management',
        title: '用户管理',
        component: 'UserManagement',
        permissions: rp(PERM.user.read),
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin'],
        },
      },
      {
        path: '/admin/change-review',
        name: 'change-review-management',
        title: '评审组管理',
        component: 'ChangeReviewManagement',
        permissions: rp(PERM.change.read),
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin', 'change_manager'],
        },
      },
      {
        path: '/admin/roles',
        name: 'role-management',
        title: '角色管理',
        component: 'RoleManagement',
        permissions: rp(PERM.role.read),
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin'],
        },
      },
      {
        path: '/admin/tenants',
        name: 'tenant-management',
        title: '租户管理',
        component: 'TenantManagement',
        permissions: rp(PERM.tenant.read),
        meta: {
          requireAuth: true,
          roles: ['super_admin'],
        },
      },
      {
        path: '/notifications',
        name: 'notification-settings',
        title: '通知配置',
        component: 'NotificationsPage',
        // 通知偏好读写挂 notification:read（与后端通知路由同码）
        permissions: rp(PERM.notification.read),
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin'],
        },
      },
      {
        path: '/workflow/audit',
        name: 'audit-logs',
        title: '操作日志',
        component: 'AuditLogsPage',
        permissions: rp(PERM.audit.read),
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin'],
        },
      },
      {
        path: '/admin/cmdb-types',
        name: 'cmdb-types',
        title: 'CMDB 类型',
        component: 'CMDBTypesManagement',
        permissions: rp(PERM.cmdb.write),
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin'],
        },
      },
      {
        path: '/admin/system-config',
        name: 'system-settings',
        title: '系统设置',
        component: 'SystemConfiguration',
        permissions: rp(PERM.system_config.read),
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin'],
        },
      },
    ],
  },
];

// 权限检查工具函数
export class RoutePermissionChecker {
  /**
   * 检查用户是否有访问路由的权限
   *
   * 判定共用 permissions/utils.matchesPermission（与后端 checkPermissionMatch
   * 同一套通配/管理员覆盖语义），不再维护前端私有的别名映射与手写匹配规则。
   */
  static hasRoutePermission(
    route: RouteConfig,
    userPermissions: RoutePermission[],
    userRoles: string[] = []
  ): boolean {
    // 检查角色权限
    if (route.meta?.roles && route.meta.roles.length > 0) {
      const hasRole = route.meta.roles.some(role => userRoles.includes(role));
      if (!hasRole) {
        return false;
      }
    }

    // 检查资源权限
    if (route.permissions && route.permissions.length > 0) {
      const permissionSet = new Set(
        userPermissions.map(p => `${p.resource}:${p.action}`)
      );
      return route.permissions.every(requiredPermission =>
        matchesPermission(
          permissionSet,
          requiredPermission.resource,
          requiredPermission.action
        )
      );
    }

    return true;
  }

  /**
   * 过滤用户可访问的路由
  /**
   * 过滤用户可访问的路由
   */
  static filterAccessibleRoutes(
    routes: RouteConfig[],
    userPermissions: RoutePermission[],
    userRoles: string[] = []
  ): RouteConfig[] {
    return routes
      .filter(route => this.hasRoutePermission(route, userPermissions, userRoles))
      .map(route => ({
        ...route,
        children: route.children
          ? this.filterAccessibleRoutes(route.children, userPermissions, userRoles)
          : undefined,
      }))
      .filter(route => !route.hidden);
  }

  /**
   * 获取面包屑导航
   */
  static getBreadcrumbs(
    currentPath: string,
    routes: RouteConfig[] = []
  ): Array<{ name: string; title: string; path: string }> {
    const breadcrumbs: Array<{ name: string; title: string; path: string }> = [];

    const findRoute = (
      routes: RouteConfig[],
      path: string,
      parents: RouteConfig[] = []
    ): boolean => {
      for (const route of routes) {
        const fullPath = route.path;

        if (fullPath === path || (route.children && path.startsWith(fullPath))) {
          // 添加父级路由到面包屑
          parents.forEach(parent => {
            if (parent.meta?.breadcrumb !== false) {
              breadcrumbs.push({
                name: parent.name,
                title: parent.title,
                path: parent.path,
              });
            }
          });

          // 添加当前路由到面包屑
          if (route.meta?.breadcrumb !== false) {
            breadcrumbs.push({
              name: route.name,
              title: route.title,
              path: route.path,
            });
          }

          // 如果有子路由，继续查找
          if (route.children && fullPath !== path) {
            return findRoute(route.children, path, [...parents, route]);
          }

          return true;
        }
      }
      return false;
    };

    findRoute(routes, currentPath);
    return breadcrumbs;
  }

  /**
   * 根据路由名称查找路由配置
   */
  static findRouteByName(name: string, routes: RouteConfig[] = []): RouteConfig | null {
    for (const route of routes) {
      if (route.name === name) {
        return route;
      }
      if (route.children) {
        const found = this.findRouteByName(name, route.children);
        if (found) {
          return found;
        }
      }
    }
    return null;
  }

  /**
   * 根据路径查找路由配置
   */
  static findRouteByPath(path: string, routes: RouteConfig[] = []): RouteConfig | null {
    for (const route of routes) {
      if (route.path === path) {
        return route;
      }
      if (route.children) {
        const found = this.findRouteByPath(path, route.children);
        if (found) {
          return found;
        }
      }
    }
    return null;
  }
}

// 导航菜单生成器
export class NavigationBuilder {
  /**
   * 生成导航菜单数据
   */
  static buildNavigation(
    routes: RouteConfig[],
    userPermissions: RoutePermission[],
    userRoles: string[] = []
  ): RouteConfig[] {
    return RoutePermissionChecker.filterAccessibleRoutes(routes, userPermissions, userRoles)
      .filter(route => !route.hidden)
      .map(route => ({
        ...route,
        children: route.children?.filter(child => !child.hidden),
      }));
  }

  /**
   * 获取扁平化的路由列表（用于权限检查）
   */
  static getFlatRoutes(routes: RouteConfig[]): RouteConfig[] {
    const flatRoutes: RouteConfig[] = [];

    const flatten = (routes: RouteConfig[]) => {
      routes.forEach(route => {
        flatRoutes.push(route);
        if (route.children) {
          flatten(route.children);
        }
      });
    };

    flatten(routes);
    return flatRoutes;
  }
}

// 默认导出
// 导出配置对象
export const routeConfig = {
  routes,
  RoutePermissionChecker,
  NavigationBuilder,
};

export default routeConfig;
