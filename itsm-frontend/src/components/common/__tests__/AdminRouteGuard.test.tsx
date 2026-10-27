/**
 * AdminRouteGuard 回归测试
 *
 * 锁定准入口径：/admin 页面对非 admin/super_admin 角色按后端菜单判定——
 * 菜单树里有该路径即放行（服务端已按权限码过滤菜单），否则 403。
 * 例外：ADMIN_ROUTE_PERMISSIONS 声明的页面（如 /admin/cmdb-types）按
 * 后端路由同款权限码放行——该页真实授权码是 cmdb:write，与「系统管理」
 * 分组自 2026-10-26 起统一取的 system_config:read 不同，
 * 菜单树不会向只持单页权限码的用户返回该子路径。
 */

import React from 'react';
import { render, screen } from '@testing-library/react';

import { AdminRouteGuard } from '../AdminRouteGuard';
import type { MenuTreeResponse } from '@/lib/api/menu-api';

jest.mock('next/navigation', () => ({
  usePathname: jest.fn(),
  useRouter: () => ({ push: jest.fn() }),
}));

const mockUsePathname = jest.requireMock('next/navigation').usePathname as jest.Mock;

const mockPermissions = { isAdmin: jest.fn(), hasPermission: jest.fn() };
jest.mock('@/lib/hooks/use-permissions', () => ({
  usePermissions: () => mockPermissions,
}));

const mockMenusQuery = { data: undefined as MenuTreeResponse | undefined, isLoading: false };
jest.mock('@/lib/hooks/useUserMenusQuery', () => ({
  useUserMenusQuery: () => mockMenusQuery,
}));

const leaf = (path: string) => ({ id: 1, name: path, path, sortOrder: 1, tenantId: 1, isVisible: true, isEnabled: true });

describe('AdminRouteGuard', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockPermissions.isAdmin = jest.fn().mockReturnValue(false);
    mockPermissions.hasPermission = jest.fn().mockReturnValue(false);
    mockMenusQuery.data = undefined;
    mockMenusQuery.isLoading = false;
  });

  it('admin 角色直通，不依赖菜单数据', () => {
    mockUsePathname.mockReturnValue('/admin/cmdb-types');
    mockPermissions.isAdmin = jest.fn().mockReturnValue(true);

    render(
      <AdminRouteGuard>
        <div>PAGE_CONTENT</div>
      </AdminRouteGuard>
    );

    expect(screen.getByText('PAGE_CONTENT')).toBeInTheDocument();
  });

  it('菜单树包含目标路径时放行', () => {
    mockUsePathname.mockReturnValue('/admin/users');
    mockMenusQuery.data = { main: [], admin: [{ ...leaf('/admin/users'), children: [] }] };

    render(
      <AdminRouteGuard>
        <div>PAGE_CONTENT</div>
      </AdminRouteGuard>
    );

    expect(screen.getByText('PAGE_CONTENT')).toBeInTheDocument();
  });

  it('cmdb-types 持 cmdb:write 时放行，菜单树无需包含该路径', () => {
    mockUsePathname.mockReturnValue('/admin/cmdb-types');
    mockPermissions.hasPermission = jest.fn().mockReturnValue(true);

    render(
      <AdminRouteGuard>
        <div>PAGE_CONTENT</div>
      </AdminRouteGuard>
    );

    expect(mockPermissions.hasPermission).toHaveBeenCalledWith('cmdb', 'write');
    expect(screen.getByText('PAGE_CONTENT')).toBeInTheDocument();
  });

  it('cmdb-types 无 cmdb:write 时 403，不等待菜单请求', () => {
    mockUsePathname.mockReturnValue('/admin/cmdb-types');
    mockMenusQuery.isLoading = true;

    const { container } = render(
      <AdminRouteGuard>
        <div>PAGE_CONTENT</div>
      </AdminRouteGuard>
    );

    expect(screen.queryByText('PAGE_CONTENT')).not.toBeInTheDocument();
    expect(screen.getByText('403')).toBeInTheDocument();
    expect(container.querySelector('.ant-spin')).not.toBeInTheDocument();
  });

  it('菜单树没有目标路径时渲染 403', () => {
    mockUsePathname.mockReturnValue('/admin/tenants');
    mockMenusQuery.data = { main: [], admin: [{ ...leaf('/admin/cmdb-types'), children: [] }] };

    render(
      <AdminRouteGuard>
        <div>PAGE_CONTENT</div>
      </AdminRouteGuard>
    );

    expect(screen.queryByText('PAGE_CONTENT')).not.toBeInTheDocument();
    expect(screen.getByText('403')).toBeInTheDocument();
  });

  it('路径挂在主桶父级下时也放行（/admin/sla-templates 属「SLA 管理」子项）', () => {
    mockUsePathname.mockReturnValue('/admin/sla-templates');
    mockMenusQuery.data = {
      main: [
        {
          ...leaf('/sla'),
          children: [{ ...leaf('/admin/sla-templates'), children: [] }],
        },
      ],
      admin: [],
    };

    render(
      <AdminRouteGuard>
        <div>PAGE_CONTENT</div>
      </AdminRouteGuard>
    );

    expect(screen.getByText('PAGE_CONTENT')).toBeInTheDocument();
  });

  it('两个桶都没有目标路径时仍然 403', () => {
    mockUsePathname.mockReturnValue('/admin/sla-templates');
    mockMenusQuery.data = { main: [{ ...leaf('/sla'), children: [] }], admin: [] };

    render(
      <AdminRouteGuard>
        <div>PAGE_CONTENT</div>
      </AdminRouteGuard>
    );

    expect(screen.queryByText('PAGE_CONTENT')).not.toBeInTheDocument();
    expect(screen.getByText('403')).toBeInTheDocument();
  });

  it('菜单请求在途时等待，不闪 403', () => {
    mockUsePathname.mockReturnValue('/admin/users');
    mockMenusQuery.data = { main: [], admin: [{ ...leaf('/admin/users'), children: [] }] };
    mockMenusQuery.isLoading = true;

    const { container } = render(
      <AdminRouteGuard>
        <div>PAGE_CONTENT</div>
      </AdminRouteGuard>
    );

    expect(screen.queryByText('PAGE_CONTENT')).not.toBeInTheDocument();
    expect(container.querySelector('.ant-spin')).toBeInTheDocument();
  });

  it('非 /admin 路径不受守卫影响', () => {
    mockUsePathname.mockReturnValue('/dashboard');
    mockMenusQuery.data = { main: [], admin: [] };

    render(
      <AdminRouteGuard>
        <div>PAGE_CONTENT</div>
      </AdminRouteGuard>
    );

    expect(screen.getByText('PAGE_CONTENT')).toBeInTheDocument();
  });

  // /reports 是纯分析面（菜单取 report:read），end_user/technician 不持该码，
  // 菜单里没有它。此前没有路由守卫，直连 URL 会渲染一个数据全 403 的空壳。
  describe('/reports 菜单准入', () => {
    it('菜单含 /reports 时放行', () => {
      mockUsePathname.mockReturnValue('/reports');
      mockMenusQuery.data = { main: [{ ...leaf('/reports'), children: [] }], admin: [] };

      render(
        <AdminRouteGuard>
          <div>PAGE_CONTENT</div>
        </AdminRouteGuard>
      );

      expect(screen.getByText('PAGE_CONTENT')).toBeInTheDocument();
    });

    it('菜单不含 /reports 时渲染 403（end_user 直连 URL）', () => {
      mockUsePathname.mockReturnValue('/reports');
      mockMenusQuery.data = { main: [{ ...leaf('/dashboard'), children: [] }], admin: [] };

      render(
        <AdminRouteGuard>
          <div>PAGE_CONTENT</div>
        </AdminRouteGuard>
      );

      expect(screen.queryByText('PAGE_CONTENT')).not.toBeInTheDocument();
      expect(screen.getByText('403')).toBeInTheDocument();
    });

    it('子页面按根路径判定（/reports/tickets 不单独建菜单）', () => {
      mockUsePathname.mockReturnValue('/reports/tickets');
      mockMenusQuery.data = { main: [{ ...leaf('/reports'), children: [] }], admin: [] };

      render(
        <AdminRouteGuard>
          <div>PAGE_CONTENT</div>
        </AdminRouteGuard>
      );

      expect(screen.getByText('PAGE_CONTENT')).toBeInTheDocument();
    });

    it('菜单请求在途时等待，不闪 403', () => {
      mockUsePathname.mockReturnValue('/reports');
      mockMenusQuery.isLoading = true;

      const { container } = render(
        <AdminRouteGuard>
          <div>PAGE_CONTENT</div>
        </AdminRouteGuard>
      );

      expect(screen.queryByText('PAGE_CONTENT')).not.toBeInTheDocument();
      expect(container.querySelector('.ant-spin')).toBeInTheDocument();
    });
  });
});
