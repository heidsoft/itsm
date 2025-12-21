'use client';

import React from 'react';
import { Layout, Menu, theme, Badge } from 'antd';
import {
  DashboardOutlined,
  FileTextOutlined,
  ExclamationCircleOutlined,
  BookOutlined,
  BarChartOutlined,
  DatabaseOutlined,
  QuestionCircleOutlined,
  CalendarOutlined,
  DeploymentUnitOutlined,
  SafetyOutlined,
  RiseOutlined,
} from '@ant-design/icons';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore } from '@/lib/store/auth-store';
import { LAYOUT_CONFIG } from '@/config/layout.config';
import styles from './Sidebar.module.css';
import { useI18n } from '@/lib/i18n';

const { Sider } = Layout;

// 菜单配置
const getMenuConfig = (t: any) => ({
  main: [
    {
      key: '/dashboard',
      icon: <DashboardOutlined />,
      label: t('dashboard.title'),
      path: '/dashboard',
      permission: 'dashboard:view',
      description: t('dashboard.description'),
    },
    {
      key: '/tickets',
      icon: <FileTextOutlined />,
      label: t('tickets.title'),
      path: '/tickets',
      permission: 'ticket:view',
      description: t('tickets.description'),
      badge: 'New',
    },
    {
      key: '/incidents',
      icon: <ExclamationCircleOutlined />,
      label: t('incidents.title'),
      path: '/incidents',
      permission: 'incident:view',
      description: t('incidents.description'),
    },
    {
      key: '/problems',
      icon: <QuestionCircleOutlined />,
      label: t('problems.title'),
      path: '/problems',
      permission: 'problem:view',
      description: t('problems.description'),
    },
    {
      key: '/changes',
      icon: <BarChartOutlined />,
      label: t('changes.title'),
      path: '/changes',
      permission: 'change:view',
      description: t('changes.description'),
    },
    {
      key: '/cmdb',
      icon: <DatabaseOutlined />,
      label: t('cmdb.title'),
      path: '/cmdb',
      permission: 'cmdb:view',
      description: t('cmdb.description'),
    },
    {
      key: '/service-catalog',
      icon: <BookOutlined />,
      label: t('serviceCatalog.title'),
      path: '/service-catalog',
      permission: 'service:view',
      description: t('serviceCatalog.description'),
    },
    {
      key: '/knowledge-base',
      icon: <QuestionCircleOutlined />,
      label: t('knowledgeBase.title'),
      path: '/knowledge-base',
      permission: 'knowledge:view',
      description: t('knowledgeBase.description'),
    },
    {
      key: '/sla',
      icon: <CalendarOutlined />,
      label: t('sla.title'),
      path: '/sla',
      permission: 'sla:view',
      description: t('sla.description'),
    },
    {
      key: '/reports',
      icon: <RiseOutlined />,
      label: t('reports.title'),
      path: '/reports',
      permission: 'report:view',
      description: t('reports.description'),
    },
  ],
  admin: [
    {
      key: '/workflow',
      icon: <DeploymentUnitOutlined />,
      label: t('workflow.title'),
      path: '/workflow',
      permission: 'workflow:config',
      description: t('workflow.description'),
    },
    // {
    //   key: '/enterprise/departments',
    //   icon: <DeploymentUnitOutlined />,
    //   label: t('enterprise.departments.title'),
    //   path: '/enterprise/departments',
    //   permission: 'admin:view',
    //   description: t('enterprise.departments.description'),
    // },
    // {
    //   key: '/enterprise/teams',
    //   icon: <DeploymentUnitOutlined />,
    //   label: t('enterprise.teams.title'),
    //   path: '/enterprise/teams',
    //   permission: 'admin:view',
    //   description: t('enterprise.teams.description'),
    // },
    // {
    //   key: '/projects',
    //   icon: <DeploymentUnitOutlined />,
    //   label: t('enterprise.projects.title'),
    //   path: '/projects',
    //   permission: 'admin:view',
    //   description: t('enterprise.projects.description'),
    // },
    // {
    //   key: '/applications',
    //   icon: <DeploymentUnitOutlined />,
    //   label: t('enterprise.applications.title'),
    //   path: '/applications',
    //   permission: 'admin:view',
    //   description: t('enterprise.applications.description'),
    // },
    // {
    //   key: '/tags',
    //   icon: <DeploymentUnitOutlined />,
    //   label: t('enterprise.tags.title'),
    //   path: '/tags',
    //   permission: 'admin:view',
    //   description: t('enterprise.tags.description'),
    // },
    {
      key: '/admin',
      icon: <SafetyOutlined />,
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
