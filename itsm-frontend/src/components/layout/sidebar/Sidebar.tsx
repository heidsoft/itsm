'use client';

/**
 * 侧边栏组件
 * 负责展示主导航菜单
 */

import React, { useState, useEffect } from 'react';
import { Layout, theme, message } from 'antd';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore, useAuthStoreHydration } from '@/lib/store/auth-store';
import { LAYOUT_CONFIG } from '@/config/layout.config';
import styles from './Sidebar.module.css';
import { getMenuConfig, type MenuItem } from './menu-config';
import { getIconByName } from './icons';
import { MenuItems, renderMenuItems } from './MenuItems';
import {
  getUserMenus,
  type MenuItem as MenuItemType,
  type MenuTreeResponse,
} from '@/lib/api/menu-api';

const { Sider } = Layout;

interface SidebarProps {
  collapsed: boolean;
  onCollapse: (collapsed: boolean) => void;
}

/**
 * 将 API 菜单转换为 Sidebar 格式
 */
function convertApiMenuToSidebar(menus: MenuItemType[]): MenuItem[] {
  return menus.map(menu => {
    const icon = getIconByName(menu.icon);
    const item: MenuItem = {
      key: menu.path,
      icon: icon || undefined,
      label: menu.name,
      path: menu.path,
      permission: menu.permission_code,
      description: menu.description,
      children: menu.children ? convertApiMenuToSidebar(menu.children) : undefined,
    };
    return item;
  });
}

/**
 * 侧边栏组件
 */
export const Sidebar: React.FC<SidebarProps> = ({ collapsed, onCollapse }) => {
  const { token } = theme.useToken();
  const router = useRouter();
  const pathname = usePathname();
  const { user } = useAuthStore();

  // 触发 auth store 的 hydration
  useAuthStoreHydration();

  // 动态菜单状态
  const [dynamicMenus, setDynamicMenus] = useState<MenuTreeResponse | null>(null);
  const [menuLoading, setMenuLoading] = useState(false);
  const [menuError, setMenuError] = useState<string | null>(null);

  // 加载用户菜单
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
  }, [user]);

  // 显示错误提示
  useEffect(() => {
    if (menuError) {
      message.error(menuError);
    }
  }, [menuError]);

  // 菜单点击处理
  const handleMenuClick = (key: string) => {
    router.push(key);
  };

  // 转换动态菜单（当 API 返回空时使用静态配置作为 fallback）
  // TODO: 后端修复菜单数据后，移除 FORCE_STATIC_MENU 标志
  const FORCE_STATIC_MENU = false; // 临时强制使用静态菜单，避免后端数据重复key问题
  const mainMenus =
    dynamicMenus && !FORCE_STATIC_MENU
      ? convertApiMenuToSidebar(dynamicMenus.main)
      : getMenuConfig().main;
  const adminMenus =
    dynamicMenus && !FORCE_STATIC_MENU
      ? convertApiMenuToSidebar(dynamicMenus.admin)
      : getMenuConfig().admin;

  const isAdmin = user?.role === 'admin' || user?.role === 'super_admin';

  return (
    <Sider
      trigger={null}
      collapsible
      collapsed={collapsed}
      onCollapse={onCollapse}
      breakpoint={LAYOUT_CONFIG.sider.breakpoint}
      collapsedWidth={LAYOUT_CONFIG.sider.collapsedWidth}
      width={LAYOUT_CONFIG.sider.width}
      className={styles.sider}
      style={{
        borderRight: `1px solid ${token.colorBorder}`,
        zIndex: LAYOUT_CONFIG.zIndex.sider,
      }}
    >
      {/* Logo 区域 */}
      <div className={`${styles.logoArea} ${collapsed ? styles.logoAreaCollapsed : ''}`}>
        <div className={styles.logoIcon}>IT</div>
        {!collapsed && (
          <div className={styles.logoTextContainer}>
            <div className={styles.logoText}>ITSM</div>
            <div className={styles.logoSubtext}>系统</div>
          </div>
        )}
      </div>

      {/* 主菜单 */}
      <div className={styles.mainMenu} style={{ flex: 1, overflowY: 'auto' }}>
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
