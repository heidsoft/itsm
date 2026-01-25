'use client';

import React from 'react';
import { Layout, Menu, theme, Badge } from 'antd';
import {
  LayoutDashboard,
  FileText,
  AlertCircle,
  HelpCircle,
  BarChart3,
  Database,
  Book,
  Calendar,
  GitMerge,
  Shield,
  TrendingUp,
} from 'lucide-react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore } from '@/lib/store/auth-store';
import { LAYOUT_CONFIG } from '@/config/layout.config';
import styles from './Sidebar.module.css';
import { useI18n } from '@/lib/i18n';

const { Sider } = Layout;

// 图标样式，统一大小
const iconStyle = { width: 16, height: 16 };

// 菜单配置
const getMenuConfig = (t: any) => ({
  main: [
    {
      key: '/dashboard',
      icon: <LayoutDashboard style={iconStyle} />,
      label: t('dashboard.title'),
      path: '/dashboard',
      permission: 'dashboard:view',
      description: t('dashboard.description'),
    },
    {
      key: '/tickets',
      icon: <FileText style={iconStyle} />,
      label: t('tickets.title'),
      path: '/tickets',
      permission: 'ticket:view',
      description: t('tickets.description'),
      badge: 'New',
    },
    {
      key: '/incidents',
      icon: <AlertCircle style={iconStyle} />,
      label: t('incidents.title'),
      path: '/incidents',
      permission: 'incident:view',
      description: t('incidents.description'),
    },
    {
      key: '/problems',
      icon: <HelpCircle style={iconStyle} />,
      label: t('problems.title'),
      path: '/problems',
      permission: 'problem:view',
      description: t('problems.description'),
    },
    {
      key: '/changes',
      icon: <BarChart3 style={iconStyle} />,
      label: t('changes.title'),
      path: '/changes',
      permission: 'change:view',
      description: t('changes.description'),
    },
    {
      key: '/cmdb',
      icon: <Database style={iconStyle} />,
      label: 'CMDB',
      path: '/cmdb',
      permission: 'cmdb:view',
      description: t('cmdb.description'),
    },
    {
      key: '/service-catalog',
      icon: <Book style={iconStyle} />,
      label: t('serviceCatalog.title'),
      path: '/service-catalog',
      permission: 'service:view',
      description: t('serviceCatalog.description'),
    },
    {
      key: '/knowledge',
      icon: <HelpCircle style={iconStyle} />,
      label: '知识库',
      path: '/knowledge',
      permission: 'knowledge:view',
      description: t('knowledgeBase.description'),
    },
    {
      key: '/sla-dashboard',
      icon: <Calendar style={iconStyle} />,
      label: 'SLA监控',
      path: '/sla-dashboard',
      permission: 'sla:view',
      description: t('sla.description'),
    },
    {
      key: '/reports',
      icon: <TrendingUp style={iconStyle} />,
      label: t('reports.title'),
      path: '/reports',
      permission: 'report:view',
      description: t('reports.description'),
    },
  ],
  admin: [
    {
      key: '/workflow',
      icon: <GitMerge style={iconStyle} />,
      label: t('workflow.title'),
      path: '/workflow',
      permission: 'workflow:config',
      description: t('workflow.description'),
    },
    {
      key: '/admin',
      icon: <Shield style={iconStyle} />,
      label: t('admin.title'),
      path: '/admin',
      permission: 'admin:view',
      description: t('admin.description'),
    },
  ],
});

interface SidebarProps {
  collapsed: boolean;
  onCollapse: (collapsed: boolean) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({ collapsed, onCollapse }) => {
  const { token } = theme.useToken();
  const router = useRouter();
  const pathname = usePathname();
  const { user } = useAuthStore();
  const { t } = useI18n();
  const MENU_CONFIG = getMenuConfig(t);

  const handleMenuClick = ({ key }: { key: string }) => {
    router.push(key);
  };

  const isAdmin = user?.role === 'admin' || user?.role === 'super_admin';

  // 渲染菜单项，支持徽章和描述
  const renderMenuItems = (items: typeof MENU_CONFIG.main) => {
    return items.map(item => ({
      key: item.key,
      icon: item.icon,
      label: (
        <div className={styles.menuItemLabel}>
          <span>{item.label}</span>
          {item.badge && <Badge count={item.badge} size='small' className={styles.menuItemBadge} />}
        </div>
      ),
      onClick: () => handleMenuClick({ key: item.key }),
      className: styles.menuItem,
    }));
  };

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
        <Menu
          mode='inline'
          selectedKeys={[pathname]}
          className={styles.customMenu}
          items={renderMenuItems(MENU_CONFIG.main)}
          theme='light'
        />
      </div>

      {/* 管理员菜单 */}
      {isAdmin && (
        <div className={styles.adminMenuContainer}>
          {!collapsed && <div className={styles.adminMenuHeader}>管理功能</div>}
          <div className={styles.adminMenu}>
            <Menu
              mode='inline'
              selectedKeys={[pathname]}
              className={styles.customMenu}
              items={renderMenuItems(MENU_CONFIG.admin)}
              theme='light'
            />
          </div>
        </div>
      )}

      {/* 底部用户信息 */}
      {!collapsed && (
        <div className={styles.userInfoContainer}>
          <div className={styles.userInfo}>
            <div className={styles.userAvatar}>
              {user?.name?.[0] || user?.username?.[0] || 'U'}
            </div>
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
