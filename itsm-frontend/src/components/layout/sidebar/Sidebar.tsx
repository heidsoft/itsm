'use client';

/**
 * 侧边栏组件
 * 负责展示主导航菜单
 */

import React, { useState, useEffect } from 'react';
import { App, Layout, theme } from 'antd';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore, useAuthStoreHydration } from '@/lib/store/auth-store';
import { LAYOUT_CONFIG } from '@/config/layout.config';
import styles from './Sidebar.module.css';
import { getMenuConfig, type MenuItem } from './menu-config';
import { getIconByName } from './icons';
import { MenuItems, renderMenuItems } from './MenuItems';
import {
  getUserMenus,
  MENUS_UPDATED_EVENT,
  type MenuItem as MenuItemType,
  type MenuTreeResponse,
} from '@/lib/api/menu-api';
import { useCapabilities } from '@/lib/hooks/useCapabilities';

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

  // 动态菜单状态
  const [dynamicMenus, setDynamicMenus] = useState<MenuTreeResponse | null>(null);
  const [menuLoading, setMenuLoading] = useState(false);
  const [menuError, setMenuError] = useState<string | null>(null);

  // 加载用户菜单（并监听菜单管理变更事件即时刷新）
  useEffect(() => {
    const loadMenus = async () => {
      if (!user) return;

      try {
        setMenuLoading(true);
        setMenuError(null);
        const menus = await getUserMenus();
        if (menus && (menus.main?.length > 0 || menus.admin?.length > 0)) {
          setDynamicMenus(menus);
        } else {
          setMenuError('菜单加载失败，请刷新页面重试');
        }
      } catch (error) {
        console.error('Failed to load dynamic menus:', error);
        setMenuError('菜单加载失败，请刷新页面重试');
      } finally {
        setMenuLoading(false);
      }
    };

    loadMenus();

    const handleMenusUpdated = () => {
      loadMenus();
    };
    window.addEventListener(MENUS_UPDATED_EVENT, handleMenusUpdated);
    return () => {
      window.removeEventListener(MENUS_UPDATED_EVENT, handleMenusUpdated);
    };
  }, [user]);

  // 显示错误提示
  useEffect(() => {
    if (menuError) {
      message.error(menuError);
    }
  }, [menuError]);

  // 菜单点击处理
  const handleMenuClick = (key: string) => {
    if (!key) {
      console.warn('Menu item has no path:', key);
      return;
    }
    try {
      router.push(key);
    } catch (error) {
      console.error('Menu navigation error:', error);
      message.error('导航失败，请稍后重试');
    }
  };

  // 转换动态菜单（当 API 返回空时使用静态配置作为 fallback）
  const FORCE_STATIC_MENU = false;
  const rawMainMenus =
    dynamicMenus && !FORCE_STATIC_MENU
      ? convertApiMenuToSidebar(dynamicMenus.main)
      : getMenuConfig().main;
  const rawAdminMenus =
    dynamicMenus && !FORCE_STATIC_MENU
      ? convertApiMenuToSidebar(dynamicMenus.admin)
      : getMenuConfig().admin;

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

  const isAdmin = user?.role === 'admin' || user?.role === 'super_admin';

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
