/**
 * Tests for Router route-config
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

import { routes, RoutePermissionChecker, NavigationBuilder } from '../route-config';
import { PERMISSION_CATALOG } from '@/lib/permissions';

describe('Route Configuration', () => {
  describe('routes', () => {
    it('should be an array with route configs', () => {
      expect(Array.isArray(routes)).toBe(true);
      expect(routes.length).toBeGreaterThan(0);
    });

    it('should have dashboard as first route', () => {
      expect(routes[0].path).toBe('/');
      expect(routes[0].name).toBe('dashboard');
    });

    it('should have tickets route with children', () => {
      const ticketRoute = routes.find(r => r.name === 'tickets');
      expect(ticketRoute).toBeDefined();
      expect(ticketRoute?.children).toBeDefined();
      expect(ticketRoute?.children?.length).toBeGreaterThan(0);
    });

    it('should have admin route with role restrictions', () => {
      const adminRoute = routes.find(r => r.name === 'admin');
      expect(adminRoute?.meta?.roles).toContain('admin');
    });
  });

  describe('RoutePermissionChecker', () => {
    describe('hasRoutePermission', () => {
      it('should allow access when no permissions required', () => {
        const route = { path: '/', name: 'home', title: 'Home' };
        const result = RoutePermissionChecker.hasRoutePermission(route, []);
        expect(result).toBe(true);
      });

      it('should check resource permissions', () => {
        const route = {
          path: '/tickets',
          name: 'tickets',
          title: 'Tickets',
          permissions: [{ resource: 'ticket', action: 'read' }],
        };
        const userPerms = [{ resource: 'ticket', action: 'read' }];
        expect(RoutePermissionChecker.hasRoutePermission(route, userPerms)).toBe(true);
      });

      it('should deny when missing permissions', () => {
        const route = {
          path: '/tickets',
          name: 'tickets',
          title: 'Tickets',
          permissions: [{ resource: 'ticket', action: 'delete' }],
        };
        const userPerms = [{ resource: 'ticket', action: 'read' }];
        expect(RoutePermissionChecker.hasRoutePermission(route, userPerms)).toBe(false);
      });

      it('should allow wildcard action permissions', () => {
        const route = {
          path: '/tickets',
          name: 'tickets',
          title: 'Tickets',
          permissions: [{ resource: 'ticket', action: 'delete' }],
        };
        const userPerms = [{ resource: 'ticket', action: '*' }];
        expect(RoutePermissionChecker.hasRoutePermission(route, userPerms)).toBe(true);
      });

      it('should check role restrictions', () => {
        const route = {
          path: '/admin',
          name: 'admin',
          title: 'Admin',
          meta: { roles: ['admin', 'super_admin'] },
        };
        expect(RoutePermissionChecker.hasRoutePermission(route, [], ['user'])).toBe(false);
        expect(RoutePermissionChecker.hasRoutePermission(route, [], ['admin'])).toBe(true);
      });

      it('should allow admin-override permissions like the backend matcher', () => {
        const route = {
          path: '/tickets',
          name: 'tickets',
          title: 'Tickets',
          permissions: [{ resource: 'ticket', action: 'delete' }],
        };
        const userPerms = [{ resource: 'ticket', action: 'admin' }];
        expect(RoutePermissionChecker.hasRoutePermission(route, userPerms)).toBe(true);
      });

      it('should not translate phantom actions (manage codes no longer aliased)', () => {
        // 回归锁：别名映射（manage→admin、service→service_catalog）已删除。
        // 持 user:write 的用户对「user:manage」声明不再被静默翻译放行——
        // 路由配置里的码必须是后端码空间的真实码（类型层由 rp(PERM.x.y) 锁定）。
        const route = {
          path: '/admin/users',
          name: 'users',
          title: 'Users',
          permissions: [{ resource: 'user', action: 'manage' }],
        };
        const userPerms = [{ resource: 'user', action: 'write' }];
        expect(RoutePermissionChecker.hasRoutePermission(route, userPerms)).toBe(false);
      });

      it('should only declare codes from the generated catalog', () => {
        // 路由权限码 ⊆ 后端码空间（构建期由 rp(PERM.x.y) 类型保证，这里再运行时锁一次）
        const flatten = (rs: typeof routes): string[] =>
          rs.flatMap(r => [
            ...(r.permissions ?? []).map(p => `${p.resource}:${p.action}`),
            ...(r.children ? flatten(r.children) : []),
          ]);
        const catalogCodes = new Set<string>(PERMISSION_CATALOG.map(e => e.code));
        for (const code of flatten(routes)) {
          expect(catalogCodes.has(code)).toBe(true);
        }
      });
    });

    describe('filterAccessibleRoutes', () => {
      it('should filter routes based on permissions', () => {
        const testRoutes = [
          { path: '/', name: 'home', title: 'Home', permissions: [{ resource: 'dashboard', action: 'read' }] },
          { path: '/admin', name: 'admin', title: 'Admin', permissions: [{ resource: 'admin', action: 'read' }] },
        ];
        const userPerms = [{ resource: 'dashboard', action: 'read' }];
        const result = RoutePermissionChecker.filterAccessibleRoutes(testRoutes, userPerms);
        expect(result).toHaveLength(1);
        expect(result[0].name).toBe('home');
      });

      it('should filter out hidden routes', () => {
        const testRoutes = [
          { path: '/', name: 'home', title: 'Home', hidden: true },
        ];
        const result = RoutePermissionChecker.filterAccessibleRoutes(testRoutes, []);
        expect(result).toHaveLength(0);
      });

      it('should filter children recursively', () => {
        const testRoutes = [
          {
            path: '/tickets',
            name: 'tickets',
            title: 'Tickets',
            permissions: [{ resource: 'ticket', action: 'read' }],
            children: [
              { path: '/tickets/create', name: 'create', title: 'Create', permissions: [{ resource: 'ticket', action: 'create' }] },
            ],
          },
        ];
        const userPerms = [{ resource: 'ticket', action: 'read' }];
        const result = RoutePermissionChecker.filterAccessibleRoutes(testRoutes, userPerms);
        expect(result[0].children).toHaveLength(0);
      });
    });

    describe('findRouteByName', () => {
      it('should find route by name', () => {
        const result = RoutePermissionChecker.findRouteByName('dashboard', routes);
        expect(result).toBeDefined();
        expect(result?.path).toBe('/');
      });

      it('should find nested route by name', () => {
        const result = RoutePermissionChecker.findRouteByName('ticket-create', routes);
        expect(result).toBeDefined();
        expect(result?.path).toBe('/tickets/create');
      });

      it('should return null for non-existent name', () => {
        const result = RoutePermissionChecker.findRouteByName('nonexistent', routes);
        expect(result).toBeNull();
      });
    });

    describe('findRouteByPath', () => {
      it('should find route by path', () => {
        const result = RoutePermissionChecker.findRouteByPath('/', routes);
        expect(result).toBeDefined();
        expect(result?.name).toBe('dashboard');
      });

      it('should find nested route by path', () => {
        const result = RoutePermissionChecker.findRouteByPath('/tickets/create', routes);
        expect(result).toBeDefined();
        expect(result?.name).toBe('ticket-create');
      });

      it('should return null for non-existent path', () => {
        const result = RoutePermissionChecker.findRouteByPath('/nonexistent', routes);
        expect(result).toBeNull();
      });
    });

    describe('getBreadcrumbs', () => {
      it('should return breadcrumbs for top-level route', () => {
        const result = RoutePermissionChecker.getBreadcrumbs('/', routes);
        expect(result.length).toBeGreaterThan(0);
        expect(result[0].name).toBe('dashboard');
      });

      it('should return empty for unknown path', () => {
        const result = RoutePermissionChecker.getBreadcrumbs('/totally-unknown-path', routes);
        expect(result).toHaveLength(0);
      });
    });
  });

  describe('NavigationBuilder', () => {
    describe('buildNavigation', () => {
      it('should build navigation based on permissions', () => {
        const userPerms = [{ resource: 'dashboard', action: 'read' }];
        const nav = NavigationBuilder.buildNavigation(routes, userPerms);
        expect(nav.length).toBeGreaterThan(0);
      });

      it('should filter out hidden children', () => {
        const testRoutes = [
          {
            path: '/test',
            name: 'test',
            title: 'Test',
            children: [
              { path: '/test/a', name: 'a', title: 'A', hidden: false },
              { path: '/test/b', name: 'b', title: 'B', hidden: true },
            ],
          },
        ];
        const nav = NavigationBuilder.buildNavigation(testRoutes, []);
        expect(nav[0].children).toHaveLength(1);
      });
    });

    describe('getFlatRoutes', () => {
      it('should flatten route hierarchy', () => {
        const flat = NavigationBuilder.getFlatRoutes(routes);
        expect(flat.length).toBeGreaterThan(routes.length);
      });

      it('should include children in flat list', () => {
        const flat = NavigationBuilder.getFlatRoutes(routes);
        expect(flat.some(r => r.name === 'ticket-create')).toBe(true);
      });
    });
  });
});
