'use client';

/**
 * 侧边栏组件
 * 负责展示主导航菜单
 */

import React, { useEffect } from 'react';
import { App, Layout, theme } from 'antd';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore, useAuthStoreHydration } from '@/lib/store/auth-store';
import { LAYOUT_CONFIG } from '@/config/layout.config';
import styles from './Sidebar.module.css';
import type { MenuItem } from './menu-config';
import { getIconByName } from './icons';
import { MenuItems } from './MenuItems';
import { useUserMenusQuery } from '@/lib/hooks/useUserMenusQuery';
import { useCapabilities } from '@/lib/hooks/useCapabilities';
import type { MenuItem as MenuItemType } from '@/lib/api/menu-api';

const { Sider } = Layout;

interface SidebarProps {
  collapsed: boolean;
  onCollapse: (collapsed: boolean) => void;
  mobile?: boolean;
}

/**
 * 将 API 菜单转换为 Sidebar 格式
 */
function convertApiMenuToSidebar(menus: MenuItemType[]): MenuItem[] {
  if (!menus) return [];
  return menus.map(menu => {
    const icon = getIconByName(menu.icon);
    const item: MenuItem = {
      key: menu.path,
      icon: icon || undefined,
      label: menu.name,
      path: menu.path,
      permission: menu.permissionCode ?? undefined,
      description: menu.description,
      capabilityKey: capabilityForPath(menu.path),
      children: menu.children ? convertApiMenuToSidebar(menu.children) : undefined,
    };
    return item;
  });
}

const capabilityPathRules: Array<[string, string]> = [
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

function capabilityForPath(path?: string): string | undefined {
  if (!path) return undefined;
  return capabilityPathRules.find(([prefix]) => path === prefix || path.startsWith(`${prefix}/`))?.[1];
}

/**
 * 菜单路径规范化：修正历史遗留路径，避免导航到 404
 * - xxx/list 后缀：在 App Router 下通常等价于 /xxx（列表页即首页）
 * - 其他明确命名错误：create→new、index→overview 等
 * - 该映射在前端 Sidebar 点击时立即生效，无需等待后端菜单数据订正
 */
const MENU_PATH_NORMALIZATIONS: Record<string, string> = {
  // /list 后缀 → 对应基础路由（App Router 下 xxx/page.tsx 即列表首页）
  '/service-requests/list': '/service-requests',
  '/incidents/list': '/incidents',
  '/problems/list': '/problems',
  '/changes/list': '/changes',
  '/knowledge/list': '/knowledge',
  '/service-catalog/list': '/service-catalog',
  '/assets/list': '/assets',
  '/workflow/list': '/workflow',
  '/ai/chat/list': '/ai/chat',
  '/msp/list': '/msp',
  '/releases/list': '/releases',
  // 明确的路径命名错误：/admin/overview 页面加载后会客户端跳转到 /admin（系统管理首页），直接指向 /admin 避免两跳
  '/admin/index': '/admin',
  '/knowledge/articles/create': '/knowledge/articles/new',
  // 缺少独立路由页面的概览入口 → 跳转到模块主页面（主页面本身就是概览）
  '/sla/overview': '/sla',
  '/email-intake/conversations': '/email-intake',
  '/knowledge/articles': '/knowledge',
};

function normalizeMenuPath(raw: string): string {
  if (!raw) return raw;
  const direct = MENU_PATH_NORMALIZATIONS[raw];
  if (direct) return direct;
  // 兜底：若目标路径形如 /xxx/list 且存在精确匹配规则以外的 /list，也尝试剥离 /list
  if (raw.endsWith('/list') && raw.length > 6) {
    return raw.slice(0, -5);
  }
  return raw;
}

/**
 * 侧边栏组件
 */
export const Sidebar: React.FC<SidebarProps> = ({ collapsed, onCollapse, mobile = false }) => {
  const { token } = theme.useToken();
  const { message } = App.useApp();
  const router = useRouter();
  const pathname = usePathname();
  const { user } = useAuthStore();
  const { capabilities, isLoading: capabilitiesLoading } = useCapabilities();

  // 触发 auth store 的 hydration
  useAuthStoreHydration();

  // 动态菜单状态：完全来自后端 /api/v1/auth/menus（seedMenus 写入 DB），
  // 通过 React Query 共享缓存，避免与面包屑、菜单管理端重复请求。
  const menusQuery = useUserMenusQuery({ enabled: !!user });
  const dynamicMenus = menusQuery.data;
  const menuLoading = menusQuery.isLoading;
  const menuError =
    menusQuery.error ||
    (menusQuery.data && menusQuery.data.main.length === 0 && menusQuery.data.admin.length === 0
      ? new Error('菜单为空，请检查角色权限或刷新页面重试')
      : null);

  // 显示错误提示
  useEffect(() => {
    if (menuError) {
      message.error(menuError.message || '菜单加载失败，请刷新页面重试');
    }
  }, [menuError, message]);

  // 菜单点击处理：先做路径规范化，再执行路由跳转
  const handleMenuClick = (key: string) => {
    if (!key) {
      console.warn('Menu item has no path:', key);
      return;
    }
    const normalizedPath = normalizeMenuPath(key);
    if (normalizedPath !== key) {
      console.debug('[Sidebar] 菜单路径已规范化', { from: key, to: normalizedPath });
    }
    try {
      router.push(normalizedPath);
    } catch (error) {
      console.error('Menu navigation error:', error);
      message.error('导航失败，请稍后重试');
    }
  };

  // 菜单全部来源于后端 /api/v1/auth/menus（seedMenus 写入 DB），不再使用前端静态 fallback。
  // /api/v1/auth/menus 返回空时会通过 menuError 提示用户刷新，避免静默吃失败。
  const rawMainMenus = dynamicMenus ? convertApiMenuToSidebar(dynamicMenus.main) : [];
  const rawAdminMenus = dynamicMenus ? convertApiMenuToSidebar(dynamicMenus.admin) : [];

  // 菜单 key 去重逻辑 — 避免后端返回重复 key 导致 React 警告
  const deduplicateMenus = (menus: MenuItem[]): MenuItem[] => {
    const seen = new Set<string>();
    const result: MenuItem[] = [];
    for (const menu of menus) {
      if (!seen.has(menu.key)) {
        seen.add(menu.key);
        // 递归去重子菜单
        const dedupedMenu = menu.children
          ? { ...menu, children: deduplicateMenus(menu.children) }
          : menu;
        result.push(dedupedMenu);
      }
    }
    return result;
  };

  const filterByCapability = (menus: MenuItem[]): MenuItem[] =>
    menus.flatMap(menu => {
      const key = menu.capabilityKey || capabilityForPath(menu.path);
      const capability = key ? capabilities.find(item => item.key === key) : undefined;
      if (key && (!capability || capability.maturity === 'disabled' || !capability.buildAvailable || !capability.deploymentReady || !capability.tenantReady)) {
        return [];
      }
      const children = menu.children ? filterByCapability(menu.children) : undefined;
      return [{
        ...menu,
        badge: capability?.maturity === 'pilot' ? 'Pilot' : menu.badge,
        children,
      }];
    });

  // Capability is fail-closed for governed entries. During the first request only
  // ungoverned core navigation is rendered, avoiding a flash of disabled features.
  const mainMenus = deduplicateMenus(filterByCapability(rawMainMenus));
  const adminMenus = deduplicateMenus(filterByCapability(rawAdminMenus));

  const { hasPermission } = useAuthStore.getState();
  const isAdmin =
    hasPermission('user:write') ||
    hasPermission('role:write') ||
    hasPermission('system_config:write') ||
    hasPermission('ticket_type:manage');

  return (
    <Sider
      trigger={null}
      collapsible
      collapsed={collapsed}
      onCollapse={onCollapse}
      breakpoint={LAYOUT_CONFIG.sider.breakpoint}
      collapsedWidth={mobile ? 0 : LAYOUT_CONFIG.sider.collapsedWidth}
      width={LAYOUT_CONFIG.sider.width}
      className={styles.sider}
      style={{
        borderRight: `1px solid ${token.colorBorder}`,
        zIndex: LAYOUT_CONFIG.zIndex.sider,
      }}
    >
      {/* Logo 区域 */}
      <div className={`${styles.logoArea} ${collapsed ? styles.logoAreaCollapsed : ''}`}>
        <div className={styles.logoIcon}>AI</div>
        {!collapsed && (
          <div className={styles.logoTextContainer}>
            <div className={styles.logoText}>AI-Native</div>
            <div className={styles.logoSubtext}>ITSM</div>
          </div>
        )}
      </div>

      {/* 主菜单 */}
      <div className={styles.mainMenu} style={{ flex: 1, overflowY: 'auto', opacity: capabilitiesLoading ? 0.85 : 1 }}>
        <MenuItems items={mainMenus} selectedKeys={[pathname]} onMenuClick={handleMenuClick} />
      </div>

      {/* 管理员菜单 */}
      {isAdmin && (
        <div className={styles.adminMenuContainer}>
          {!collapsed && <div className={styles.adminMenuHeader}>管理功能</div>}
          <div className={styles.adminMenu}>
            <MenuItems items={adminMenus} selectedKeys={[pathname]} onMenuClick={handleMenuClick} />
          </div>
        </div>
      )}

      {/* 底部用户信息 */}
      {!collapsed && (
        <div className={styles.userInfoContainer}>
          <div className={styles.userInfo}>
            <div className={styles.userAvatar}>{user?.name?.[0] || user?.username?.[0] || 'U'}</div>
            <div className={styles.userDetails}>
              <div className={styles.userName}>{user?.name || user?.username}</div>
              <div className={styles.userRole}>{user?.role || 'user'}</div>
            </div>
          </div>
        </div>
      )}
    </Sider>
  );
};
