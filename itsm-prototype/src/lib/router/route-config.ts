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
    title: '仪表盘',
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
    title: '工单管理',
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
        title: '工单列表',
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
        title: '创建工单',
        component: 'TicketCreate',
        permissions: [{ resource: 'ticket', action: 'create' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/tickets/:id',
        name: 'ticket-detail',
        title: '工单详情',
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
        title: '编辑工单',
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
    title: '事件管理',
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
        title: '事件列表',
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
        title: '创建事件',
        component: 'IncidentCreate',
        permissions: [{ resource: 'incident', action: 'create' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/incidents/:id',
        name: 'incident-detail',
        title: '事件详情',
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
    title: '问题管理',
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
        title: '问题列表',
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
        title: '创建问题',
        component: 'ProblemCreate',
        permissions: [{ resource: 'problem', action: 'create' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/problems/:id',
        name: 'problem-detail',
        title: '问题详情',
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
    path: '/changes',
    name: 'changes',
    title: '变更管理',
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
        title: '变更列表',
        component: 'ChangeList',
        permissions: [{ resource: 'change', action: 'read' }],
        meta: {
          requireAuth: true,
          keepAlive: true,
        },
      },
      {
        path: '/changes/create',
        name: 'change-create',
        title: '创建变更',
        component: 'ChangeCreate',
        permissions: [{ resource: 'change', action: 'create' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/changes/:id',
        name: 'change-detail',
        title: '变更详情',
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
    path: '/knowledge',
    name: 'knowledge',
    title: '知识库',
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
        title: '知识文章',
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
        title: '创建文章',
        component: 'KnowledgeCreate',
        permissions: [{ resource: 'knowledge', action: 'create' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/knowledge/:id',
        name: 'knowledge-detail',
        title: '文章详情',
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
    path: '/cmdb',
    name: 'cmdb',
    title: '配置管理',
    icon: 'Database',
    permissions: [{ resource: 'cmdb', action: 'read' }],
    meta: {
      requireAuth: true,
      breadcrumb: true,
    },
    children: [
      {
        path: '/cmdb/ci',
        name: 'ci-list',
        title: '配置项',
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
        title: '配置项类型',
        component: 'CITypeList',
        permissions: [{ resource: 'cmdb', action: 'manage' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/cmdb/relationships',
        name: 'ci-relationship-list',
        title: '关系管理',
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
    title: '报表分析',
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
        title: '报表仪表盘',
        component: 'ReportDashboard',
        permissions: [{ resource: 'report', action: 'read' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/reports/tickets',
        name: 'ticket-reports',
        title: '工单报表',
        component: 'TicketReports',
        permissions: [{ resource: 'report', action: 'read' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/reports/sla',
        name: 'sla-reports',
        title: 'SLA报表',
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
    title: '测试管理',
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
        title: '测试运行器',
        component: 'TestRunner',
        permissions: [{ resource: 'testing', action: 'execute' }],
        meta: {
          requireAuth: true,
        },
      },
      {
        path: '/testing/coverage',
        name: 'test-coverage',
        title: '测试覆盖率',
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
    title: '系统管理',
    icon: 'Settings',
    permissions: [{ resource: 'admin', action: 'read' }],
    meta: {
      requireAuth: true,
      roles: ['admin', 'super_admin'],
      breadcrumb: true,
    },
    children: [
      {
        path: '/admin/users',
        name: 'user-management',
        title: '用户管理',
        component: 'UserManagement',
        permissions: [{ resource: 'user', action: 'manage' }],
        meta: {
          requireAuth: true,
          roles: ['admin', 'super_admin'],
        },
      },
      {
        path: '/admin/roles',
        name: 'role-management',
        title: '角色管理',
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
        title: '租户管理',
        component: 'TenantManagement',
        permissions: [{ resource: 'tenant', action: 'manage' }],
        meta: {
          requireAuth: true,
          roles: ['super_admin'],
        },
      },
      {
        path: '/admin/system',
        name: 'system-settings',
        title: '系统设置',
        component: 'SystemSettings',
        permissions: [{ resource: 'system', action: 'manage' }],
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
      return route.permissions.every(requiredPermission =>
        userPermissions.some(userPermission =>
          userPermission.resource === requiredPermission.resource &&
          userPermission.action === requiredPermission.action
        )
      );
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

    const findRoute = (routes: RouteConfig[], path: string, parents: RouteConfig[] = []): boolean => {
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