/**
 * 路由配置和权限管理
 */

// 路由权限类型
export interface RoutePermission {
  resource: string;
  action: string;
}

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
    title: 'Dashboard',
    icon: 'LayoutDashboard',
    component: 'Dashboard',
    permissions: [{ resource: 'dashboard', action: 'read' }],
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
  },
  {
    path: '/tickets',
    name: 'tickets',
    title: 'Ticket Management',
    icon: 'Ticket',
    permissions: [{ resource: 'ticket', action: 'read' }],
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/tickets',
        name: 'ticket-list',
        title: 'Ticket List',
        component: 'TicketList',
        permissions: [{ resource: 'ticket', action: 'read' }],
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/tickets/create',
        name: 'ticket-create',
        title: 'Create a work order',
        component: 'TicketCreate',
        permissions: [{ resource: 'ticket', action: 'create' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/tickets/:id',
        name: 'ticket-detail',
        title: 'Ticket Detail',
        component: 'TicketDetail',
        permissions: [{ resource: 'ticket', action: 'read' }],
        hidden: true,
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/tickets/:id/edit',
        name: 'ticket-edit',
        title: 'Edit Ticket',
        component: 'TicketEdit',
        permissions: [{ resource: 'ticket', action: 'update' }],
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
    title: 'Incidents',
    icon: 'AlertTriangle',
    permissions: [{ resource: 'incident', action: 'read' }],
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/incidents',
        name: 'incident-list',
        title: 'Event List',
        component: 'IncidentList',
        permissions: [{ resource: 'incident', action: 'read' }],
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/incidents/create',
        name: 'incident-create',
        title: 'Create event',
        component: 'IncidentCreate',
        permissions: [{ resource: 'incident', action: 'create' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/incidents/:id',
        name: 'incident-detail',
        title: 'Incident Detail',
        component: 'IncidentDetail',
        permissions: [{ resource: 'incident', action: 'read' }],
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
    title: 'Problems',
    icon: 'Bug',
    permissions: [{ resource: 'problem', action: 'read' }],
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/problems',
        name: 'problem-list',
        title: 'Problem List',
        component: 'ProblemList',
        permissions: [{ resource: 'problem', action: 'read' }],
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/problems/create',
        name: 'problem-create',
        title: 'Create issue',
        component: 'ProblemCreate',
        permissions: [{ resource: 'problem', action: 'create' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/problems/:id',
        name: 'problem-detail',
        title: 'Problem Detail',
        component: 'ProblemDetail',
        permissions: [{ resource: 'problem', action: 'read' }],
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
    title: 'Release Management',
    icon: 'Rocket',
    permissions: [{ resource: 'release', action: 'read' }],
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/releases',
        name: 'release-list',
        title: 'Release list',
        component: 'ReleaseList',
        permissions: [{ resource: 'release', action: 'read' }],
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/releases/new',
        name: 'release-create',
        title: 'Create publication',
        component: 'ReleaseForm',
        permissions: [{ resource: 'release', action: 'create' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/releases/:id',
        name: 'release-detail',
        title: 'Release details',
        component: 'ReleaseDetail',
        permissions: [{ resource: 'release', action: 'read' }],
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
    title: 'Changes',
    icon: 'GitBranch',
    permissions: [{ resource: 'change', action: 'read' }],
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/changes',
        name: 'change-list',
        title: 'Change List',
        component: 'ChangeList',
        permissions: [{ resource: 'change', action: 'read' }],
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/changes/new',
        name: 'change-create',
        title: 'Create Change',
        component: 'ChangeCreate',
        permissions: [{ resource: 'change', action: 'create' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/changes/:id',
        name: 'change-detail',
        title: 'Change Detail',
        component: 'ChangeDetail',
        permissions: [{ resource: 'change', action: 'read' }],
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
    title: 'Asset Management',
    icon: 'Desktop',
    permissions: [{ resource: 'asset', action: 'read' }],
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/assets',
        name: 'asset-list',
        title: 'Asset List',
        component: 'AssetList',
        permissions: [{ resource: 'asset', action: 'read' }],
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/assets/new',
        name: 'asset-create',
        title: 'Create assets',
        component: 'AssetForm',
        permissions: [{ resource: 'asset', action: 'create' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/assets/:id',
        name: 'asset-detail',
        title: 'Asset details',
        component: 'AssetDetail',
        permissions: [{ resource: 'asset', action: 'read' }],
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
    title: 'License Management',
    icon: 'Key',
    permissions: [{ resource: 'license', action: 'read' }],
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/licenses',
        name: 'license-list',
        title: 'License List',
        component: 'LicenseList',
        permissions: [{ resource: 'license', action: 'read' }],
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/licenses/new',
        name: 'license-create',
        title: 'Create license',
        component: 'LicenseForm',
        permissions: [{ resource: 'license', action: 'create' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/licenses/:id',
        name: 'license-detail',
        title: 'License details',
        component: 'LicenseDetail',
        permissions: [{ resource: 'license', action: 'read' }],
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
    title: 'Knowledge Base',
    icon: 'BookOpen',
    permissions: [{ resource: 'knowledge', action: 'read' }],
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/knowledge',
        name: 'knowledge-list',
        title: 'Knowledge Article',
        component: 'KnowledgeList',
        permissions: [{ resource: 'knowledge', action: 'read' }],
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/knowledge/create',
        name: 'knowledge-create',
        title: 'Create article',
        component: 'KnowledgeCreate',
        permissions: [{ resource: 'knowledge', action: 'create' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/knowledge/:id',
        name: 'knowledge-detail',
        title: 'Article details',
        component: 'KnowledgeDetail',
        permissions: [{ resource: 'knowledge', action: 'read' }],
        hidden: true,
        meta: {
          requireAuth: true,
        },
      },
    ],
  },
  {
    path: '/ai',
    name: 'ai',
    title: 'Intelligent Assistant',
    icon: 'Bot',
    permissions: [{ resource: 'ai', action: 'read' }],
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/ai/chat',
        name: 'ai-chat',
        title: 'AI Q&A',
        component: 'AIChat',
        permissions: [{ resource: 'ai', action: 'read' }],
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
    ],
  },
  {
    path: '/cmdb',
    name: 'cmdb',
    title: 'Configuration Management',
    icon: 'Database',
    permissions: [{ resource: 'cmdb', action: 'read' }],
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/cmdb/cis',
        name: 'ci-list',
        title: 'Configuration item',
        component: 'CIList',
        permissions: [{ resource: 'cmdb', action: 'read' }],
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/cmdb/ci-types',
        name: 'ci-type-list',
        title: 'Configuration item type',
        component: 'CITypeList',
        permissions: [{ resource: 'cmdb', action: 'manage' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/cmdb/relationships',
        name: 'ci-relationship-list',
        title: 'Relationship Management',
        component: 'CIRelationshipList',
        permissions: [{ resource: 'cmdb', action: 'read' }],
        meta: {
          requireAuth: true,
        },
      },
    ],
  },
  {
    path: '/reports',
    name: 'reports',
    title: 'Reports & Analytics',
    icon: 'BarChart3',
    permissions: [{ resource: 'report', action: 'read' }],
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/reports/dashboard',
        name: 'report-dashboard',
        title: 'Report Dashboard',
        component: 'ReportDashboard',
        permissions: [{ resource: 'report', action: 'read' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/reports/tickets',
        name: 'ticket-reports',
        title: 'Work order report',
        component: 'TicketReports',
        permissions: [{ resource: 'report', action: 'read' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/reports/sla',
        name: 'sla-reports',
        title: 'SLA report',
        component: 'SLAReports',
        permissions: [{ resource: 'report', action: 'read' }],
        meta: {
          requireAuth: true,
        },
      },
    ],
  },
  {
    path: '/testing',
    name: 'testing',
    title: 'Test Management',
    icon: 'Code',
    permissions: [{ resource: 'testing', action: 'read' }],
    meta: {
      requireAuth: true,
      roles: ['admin', 'developer'],
      breadcrumb: true,
    },
    children: [
      {
        path: '/testing/runner',
        name: 'test-runner',
        title: 'test runner',
        component: 'TestRunner',
        permissions: [{ resource: 'testing', action: 'execute' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/testing/coverage',
        name: 'test-coverage',
        title: 'Test coverage',
        component: 'TestCoverage',
        permissions: [{ resource: 'testing', action: 'read' }],
        meta: {
          requireAuth: true,
        },
      },
    ],
  },
  {
    path: '/admin',
    name: 'admin',
    title: 'System Admin',
    icon: 'Settings',
    permissions: [{ resource: 'admin', action: 'read' }],
    meta: {
      requireAuth: true,
      roles: ['admin', 'super_admin'],
      breadcrumb: true,
    },
    children: [
      {
        path: '/admin',
        name: 'admin-overview',
        title: 'System Overview',
        component: 'AdminDashboard',
        permissions: [{ resource: 'admin', action: 'read' }],
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin'],
          keepAlive: true,
        },
      },
      {
        path: '/admin/users',
        name: 'user-management',
        title: 'User Management',
        component: 'UserManagement',
        permissions: [{ resource: 'user', action: 'manage' }],
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin'],
        },
      },
      {
        path: '/admin/change-review',
        name: 'change-review-management',
        title: 'Review team management',
        component: 'ChangeReviewManagement',
        permissions: [{ resource: 'change', action: 'read' }],
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin', 'change_manager'],
        },
      },
      {
        path: '/admin/roles',
        name: 'role-management',
        title: 'Role Management',
        component: 'RoleManagement',
        permissions: [{ resource: 'role', action: 'manage' }],
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin'],
        },
      },
      {
        path: '/admin/tenants',
        name: 'tenant-management',
        title: 'Tenant Management',
        component: 'TenantManagement',
        permissions: [{ resource: 'tenant', action: 'manage' }],
        meta: {
          requireAuth: true,
          roles: ['super_admin'],
        },
      },
      {
        path: '/notifications',
        name: 'notification-settings',
        title: 'Notification Configuration',
        component: 'NotificationsPage',
        permissions: [{ resource: 'system', action: 'manage' }],
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin'],
        },
      },
      {
        path: '/workflow/audit',
        name: 'audit-logs',
        title: 'Operation log',
        component: 'AuditLogsPage',
        permissions: [{ resource: 'system', action: 'view_logs' }],
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin'],
        },
      },
      {
        path: '/admin/cmdb-types',
        name: 'cmdb-types',
        title: 'CMDB type',
        component: 'CMDBTypesManagement',
        permissions: [{ resource: 'cmdb', action: 'manage' }],
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin'],
        },
      },
      {
        path: '/admin/system-config',
        name: 'system-settings',
        title: 'System settings',
        component: 'SystemConfiguration',
        permissions: [{ resource: 'system', action: 'manage' }],
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin'],
        },
      },
    ],
  },
];

// 资源别名映射：前端使用的资源名 → 后端定义的资源名
const resourceAliasMap: Record<string, string> = {
  service: 'service_catalog', // 前端 service:read → 后端 service_catalog:read
  workflow: 'bpmn', // 前端 workflow:read → 后端 bpmn:read
};

// Action 别名映射：前端使用的 action → 后端定义的 action
const actionAliasMap: Record<string, string> = {
  view: 'read', // view 是 read 的别名
  use: 'read', // use 是 read 的别名
  manage: 'admin', // manage 是 admin 的别名
};

// 解析资源别名
function resolveResourceAlias(resource: string): string {
  return resourceAliasMap[resource] || resource;
}

// 解析 Action 别名
function resolveActionAlias(action: string): string {
  return actionAliasMap[action] || action;
}

// 权限检查工具函数
export class RoutePermissionChecker {
  /**
   * 检查用户是否有访问路由的权限
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
      return route.permissions.every(requiredPermission => {
        const resolvedResource = resolveResourceAlias(requiredPermission.resource);
        const resolvedAction = resolveActionAlias(requiredPermission.action);

        return userPermissions.some(userPermission => {
          // 检查精确匹配（后端返回的权限已经是解析后的）
          if (
            userPermission.resource === resolvedResource &&
            userPermission.action === resolvedAction
          ) {
            return true;
          }
          // 检查通配符权限（如 service_catalog:* 可以访问 service_catalog:read）
          if (userPermission.resource === resolvedResource && userPermission.action === '*') {
            return true;
          }
          // 检查原始格式匹配（未经过别名解析的权限）
          if (
            userPermission.resource === requiredPermission.resource &&
            userPermission.action === requiredPermission.action
          ) {
            return true;
          }
          return false;
        });
      });
    }

    return true;
  }

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
