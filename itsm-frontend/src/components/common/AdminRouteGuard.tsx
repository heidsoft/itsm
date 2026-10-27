'use client';

import { Button, Result, Spin } from 'antd';
import { usePathname, useRouter } from 'next/navigation';
import type { MenuItem } from '@/lib/api/menu-api';
import { usePermissions } from '@/lib/hooks/use-permissions';
import { useUserMenusQuery } from '@/lib/hooks/useUserMenusQuery';

// /admin 页面准入与后端菜单口径一致：菜单可见性由服务端按权限码过滤
// （pkg/menubaseline 每个管理页一个 PermissionCode），这里只做同口径的展示拦截，
// 后端 RBAC 仍是唯一授权方。admin/super_admin 直通保持旧行为。
function menuContainsPath(nodes: MenuItem[] | undefined, pathname: string): boolean {
  if (!nodes?.length) return false;
  return nodes.some(node => node.path === pathname || menuContainsPath(node.children, pathname));
}

// 「系统管理」分组节点自 2026-10-26 起统一取 system_config:read（不再用 system:write），
// 但个别子页面的真实授权码与分组不同（如 /admin/cmdb-types 后端路由要求 cmdb:write）。
// 这类页面在此按后端路由同款权限码单独放行，避免为个别页面放宽整个菜单分组语义。
const ADMIN_ROUTE_PERMISSIONS: Record<string, string> = {
  '/admin/cmdb-types': 'cmdb:write',
};

export function AdminRouteGuard({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { isAdmin, hasPermission } = usePermissions();
  const router = useRouter();
  const { data: menus, isLoading } = useUserMenusQuery();

  if (pathname.startsWith('/admin') && !isAdmin()) {
    const routePermission = ADMIN_ROUTE_PERMISSIONS[pathname];
    if (routePermission) {
      const separatorIndex = routePermission.indexOf(':');
      const allowed =
        separatorIndex > 0 &&
        hasPermission(routePermission.slice(0, separatorIndex), routePermission.slice(separatorIndex + 1));
      if (!allowed) {
        return <DeniedResult router={router} />;
      }
      return <>{children}</>;
    }
    // 深链硬刷新时菜单请求在途，先等结果再判定，避免误闪 403
    if (isLoading) {
      return (
        <div className="flex min-h-[50vh] items-center justify-center">
          <Spin />
        </div>
      );
    }
    // 菜单桶按「根祖先」分桶，不按自身 path 前缀：配置项类型/SLA 模板/工单分类等
    // 一批 /admin/* 页面挂在「配置管理」「SLA 管理」「工单管理」等主桶父级下，
    // 只查 admin 桶会把持 sla:write 的 it_admin 挡在 /admin/sla-templates 之外。
    if (!menuContainsPath(menus?.admin, pathname) && !menuContainsPath(menus?.main, pathname)) {
      return <DeniedResult router={router} />;
    }
  }

  return <>{children}</>;
}

function DeniedResult({ router }: { router: ReturnType<typeof useRouter> }) {
  return (
    <Result
      status="403"
      title="403"
      subTitle="抱歉，您没有权限访问此页面。"
      extra={
        <Button type="primary" onClick={() => router.push('/')}>
          返回首页
        </Button>
      }
    />
  );
}
