'use client';

import React, { useState, useEffect } from 'react';
import {
  Layout,
  Button,
  Avatar,
  Dropdown,
  Tooltip,
  Badge,
  Typography,
  Drawer,
  Input,
  message,
  Modal,
} from 'antd';
import {
  User,
  Search,
  LogOut,
  Bell,
  ChevronDown,
  PanelLeftClose,
  PanelLeftOpen,
  Settings,
  CheckCheck,
} from 'lucide-react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore, useAuthStoreHydration } from '@/lib/store/auth-store';
import styles from './Header.module.css';
import { useI18n } from '@/lib/i18n';
import { globalSearchApi, GlobalSearchResult } from '@/lib/api/global-search-api';

const { Header: AntHeader } = Layout;
const { Text } = Typography;

// 图标样式
const iconStyle = { width: 16, height: 16 };
const headerIconStyle = { width: 20, height: 20 };

interface HeaderProps {
  collapsed: boolean;
  onCollapse: (collapsed: boolean) => void;
  title?: string;
  breadcrumb?: Array<{ title: string; href?: string }>;
  showBackButton?: boolean;
  extra?: React.ReactNode;
  showBreadcrumb?: boolean;
}

export const Header: React.FC<HeaderProps> = ({
  collapsed,
  onCollapse,
  breadcrumb,
  showBreadcrumb = true,
}) => {
  const router = useRouter();
  const pathname = usePathname();
  const { user, logout } = useAuthStore();
  const { t } = useI18n();
  // 手动触发 auth store 的 hydration
  useAuthStoreHydration();
  const [notificationsOpen, setNotificationsOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const [notifications, setNotifications] = useState([
    {
      id: 1,
      title: t('header.newTicketAssigned'),
      content: t('header.newTicketAssignedContent'),
      time: t('header.twoMinutesAgo'),
      read: false,
      type: 'ticket',
      priority: 'high',
    },
    {
      id: 2,
      title: t('header.systemMaintenanceNotification'),
      content: t('header.systemMaintenanceNotificationContent'),
      time: t('header.oneHourAgo'),
      read: true,
      type: 'system',
      priority: 'medium',
    },
    {
      id: 3,
      title: t('header.slaWarning'),
      content: t('header.slaWarningContent', { ticketId: '1234' }),
      time: t('header.threeHoursAgo'),
      read: false,
      type: 'sla',
      priority: 'urgent',
    },
  ]);

  const markAsRead = (id: number) => {
    setNotifications(prev => prev.map(n => (n.id === id ? { ...n, read: true } : n)));
  };

  const markAllAsRead = () => {
    setNotifications(prev => prev.map(n => ({ ...n, read: true })));
  };

  const handleNotificationClick = (item: any) => {
    if (!item.read) {
      markAsRead(item.id);
    }
  };
  const [searchModalVisible, setSearchModalVisible] = useState(false);
  const [searchResults, setSearchResults] = useState<GlobalSearchResult | null>(null);
  const [isClient, setIsClient] = useState(false);

  // 处理 hydration 问题
  useEffect(() => {
    setIsClient(true);
  }, []);

  // 用户显示名称（处理空值）
  const displayName = user?.name || user?.username || '';
  // 用户首字母
  const userInitial = displayName.charAt(0).toUpperCase() || 'U';
  // 用户角色显示
  const roleText =
    user?.role === 'admin'
      ? t('header.admin')
      : user?.role === 'super_admin'
        ? t('header.superAdmin')
        : t('header.user');

  const handleLogout = () => {
    logout();
    router.push('/login');
    message.success(t('header.logoutSuccess'));
  };

  const handleSearch = async (value: string) => {
    if (value.trim()) {
      const results = await globalSearchApi.search(value);
      setSearchResults(results);
      setSearchValue('');
      setSearchModalVisible(true);
    }
  };

  const userMenuItems = [
    {
      key: 'profile',
      label: t('header.profile'),
      icon: <User style={iconStyle} />,
      onClick: () => router.push('/profile'),
    },
    {
      key: 'settings',
      label: t('header.settings'),
      icon: <Settings style={iconStyle} />,
      onClick: () => router.push('/profile'),
    },
    {
      type: 'divider' as const,
    },
    {
      key: 'logout',
      label: t('header.logout'),
      icon: <LogOut style={iconStyle} />,
      onClick: handleLogout,
    },
  ];

  const unreadCount = notifications.filter(n => !n.read).length;

  // 获取当前页面标题
  const getCurrentPageTitle = () => {
    if (breadcrumb && breadcrumb.length > 0) {
      return breadcrumb[breadcrumb.length - 1].title;
    }

    // 根据路径获取页面标题
    const pathTitles: Record<string, string> = {
      '/dashboard': t('dashboard.title'),
      '/ticket-management': t('tickets.title'),
      '/incident-management': t('incidents.title'),
      '/problem-management': t('problems.title'),
      '/changes-management': t('changes.title'),
      '/cmdb': t('cmdb.title'),
      '/service-catalog': t('serviceCatalog.title'),
      '/knowledge-base': t('knowledgeBase.title'),
      '/sla-management': t('sla.title'),
      '/reports': t('reports.title'),
      '/workflow': t('workflow.title'),
      '/admin': t('admin.title'),
    };

    // 检查是否有匹配的路径前缀
    for (const [path, title] of Object.entries(pathTitles)) {
      if (pathname.startsWith(path)) {
        return title;
      }
    }

    return t('header.itsmSystem');
  };

  // 生成智能面包屑
  const generateSmartBreadcrumb = (): Array<{
    title: string;
    href?: string;
    current?: boolean;
  }> => {
    if (breadcrumb && breadcrumb.length > 0) {
      return breadcrumb;
    }

    // 根据路径自动生成面包屑
    const pathSegments = pathname.split('/').filter(Boolean);
    const smartBreadcrumb: Array<{
      title: string;
      href?: string;
      current?: boolean;
    }> = [];

    let currentPath = '';
    pathSegments.forEach((segment, index) => {
      currentPath += `/${segment}`;

      // 映射路径到中文名称
      const segmentNames: Record<string, string> = {
        dashboard: t('dashboard.title'),
        'ticket-management': t('tickets.title'),
        'incident-management': t('incidents.title'),
        'problem-management': t('problems.title'),
        'changes-management': t('changes.title'),
        cmdb: t('cmdb.title'),
        'service-catalog': t('serviceCatalog.title'),
        'knowledge-base': t('knowledgeBase.title'),
        'sla-management': t('sla.title'),
        reports: t('reports.title'),
        workflow: t('workflow.title'),
        admin: t('admin.title'),
        create: t('header.create'),
        edit: t('header.edit'),
        detail: t('header.detail'),
        templates: t('header.templates'),
        instances: t('header.instances'),
        versions: t('header.versions'),
        automation: t('header.automation'),
        approval: t('header.approval'),
        users: t('admin.users'),
        roles: t('admin.roles'),
        permissions: t('admin.permissions'),
        groups: t('admin.userGroups'),
        tenants: t('admin.tenantManagement'),
        'escalation-rules': t('admin.escalationRules'),
        'approval-chains': t('admin.approvalChains'),
        'service-catalogs': t('admin.serviceCatalog'),
        'sla-definitions': t('admin.slaDefinitions'),
        'system-config': t('admin.systemConfig'),
        'ticket-categories': t('admin.ticketCategories'),
      };

      const name = segmentNames[segment] || segment;

      // 只有当前页面才显示为不可点击
      const isLast = index === pathSegments.length - 1;

      smartBreadcrumb.push({
        title: name,
        href: isLast ? undefined : currentPath,
        current: isLast,
      });
    });

    return smartBreadcrumb;
  };

  const smartBreadcrumb = generateSmartBreadcrumb();

  return (
    <AntHeader className={styles.header}>
      {/* 左侧区域 */}
      <div className={styles.left}>
        {/* 折叠按钮 */}
        <Button
          type="text"
          icon={
            collapsed ? (
              <PanelLeftOpen style={headerIconStyle} />
            ) : (
              <PanelLeftClose style={headerIconStyle} />
            )
          }
          onClick={() => onCollapse(!collapsed)}
          className={styles.collapseButton}
        />

        {/* 当前页面标题 */}
        <div className={styles.titleContainer}>
          <Text className={styles.title}>{getCurrentPageTitle()}</Text>

          {/* 智能面包屑导航 - 只在有多级路径时显示 */}
          {showBreadcrumb && smartBreadcrumb.length > 1 && (
            <div className={styles.breadcrumb}>
              {smartBreadcrumb.map((item, index) => (
                <React.Fragment key={index}>
                  {index > 0 && <Text className={styles.breadcrumbSeparator}>/</Text>}
                  {item.href ? (
                    <Text className={styles.breadcrumbLink} onClick={() => router.push(item.href!)}>
                      {item.title}
                    </Text>
                  ) : (
                    <Text className={styles.breadcrumbCurrent}>{item.title}</Text>
                  )}
                </React.Fragment>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* 右侧区域 */}
      <div className={styles.right}>
        {/* 搜索框 */}
        <Input
          placeholder={t('header.searchPlaceholder')}
          prefix={<Search style={{ width: 16, height: 16, color: '#9ca3af' }} />}
          value={searchValue}
          onChange={e => setSearchValue(e.target.value)}
          onPressEnter={() => handleSearch(searchValue)}
          size="small"
          className={styles.searchInput}
        />

        {/* 通知 */}
        <Tooltip title={t('header.notificationCenter')}>
          <Badge count={unreadCount} size="small" offset={[-2, 2]}>
            <Button
              type="text"
              icon={<Bell style={headerIconStyle} />}
              onClick={() => setNotificationsOpen(true)}
              className={styles.notificationButton}
            />
          </Badge>
        </Tooltip>

        {/* 用户菜单 */}
        {isClient ? (
          user ? (
            <Dropdown
              menu={{ items: userMenuItems }}
              placement="bottomRight"
              trigger={['click']}
              open={userMenuOpen}
              onOpenChange={setUserMenuOpen}
            >
              <div className={styles.userMenu}>
                <Avatar size={28} className={styles.userAvatar}>
                  {userInitial}
                </Avatar>
                <div className={styles.userInfo}>
                  <Text className={styles.userName}>{displayName}</Text>
                  <Text className={styles.userRole}>{roleText}</Text>
                </div>
                <ChevronDown style={{ width: 14, height: 14, color: '#9ca3af' }} />
              </div>
            </Dropdown>
          ) : (
            <Button type="primary" onClick={() => router.push('/login')}>
              {t('header.login')}
            </Button>
          )
        ) : (
          // Hydration 期间显示占位符
          <div className={styles.userMenuPlaceholder} />
        )}
      </div>

      {/* 通知抽屉 */}
      <Drawer
        title={
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 8,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Bell style={{ color: '#3b82f6', width: 20, height: 20 }} />
              <span>{t('header.notificationCenter')}</span>
              {unreadCount > 0 && <Badge count={unreadCount} size="small" />}
            </div>
            {unreadCount > 0 && (
              <Button
                type="text"
                size="small"
                icon={<CheckCheck size={14} />}
                onClick={markAllAsRead}
                style={{ color: '#3b82f6' }}
              >
                全部标记为已读
              </Button>
            )}
          </div>
        }
        placement="right"
        size="large"
        style={{ maxWidth: 'calc(100vw - 64px)' }}
        open={notificationsOpen}
        onClose={() => setNotificationsOpen(false)}
        className={styles.notificationDrawer}
      >
        {notifications.length === 0 ? (
          <div
            className={styles.emptyNotifications}
            style={{ textAlign: 'center', padding: '48px 0' }}
          >
            <Bell style={{ width: 48, height: 48, marginBottom: '16px', opacity: 0.3 }} />
            <Text style={{ display: 'block', color: '#9ca3af' }}>
              {t('header.noNotifications')}
            </Text>
          </div>
        ) : (
          <div
            className="notifications-list"
            style={{ maxHeight: 'calc(100vh - 120px)', overflowY: 'auto' }}
          >
            {notifications.map(item => (
              <div
                key={item.id}
                className={styles.notificationItem}
                style={{
                  opacity: item.read ? 0.7 : 1,
                  padding: '12px',
                  borderBottom: '1px solid #f0f0f0',
                  cursor: 'pointer',
                  transition: 'background-color 0.2s',
                }}
                onClick={() => handleNotificationClick(item)}
                onMouseEnter={e =>
                  (e.currentTarget.style.backgroundColor = item.read ? '#f9fafb' : '#eff6ff')
                }
                onMouseLeave={e => (e.currentTarget.style.backgroundColor = 'transparent')}
              >
                <div style={{ display: 'flex', gap: 12, alignItems: 'flex-start' }}>
                  <div
                    className={styles.notificationAvatar}
                    style={{
                      backgroundColor:
                        item.priority === 'urgent'
                          ? '#ef4444'
                          : item.priority === 'high'
                            ? '#f59e0b'
                            : '#3b82f6',
                      width: 36,
                      height: 36,
                      borderRadius: '50%',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      color: '#fff',
                      fontWeight: 600,
                      flexShrink: 0,
                    }}
                  >
                    {item.type === 'ticket' ? 'T' : item.type === 'system' ? 'S' : 'SLA'}
                  </div>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div className={styles.notificationTitle}>
                      <Text
                        style={{
                          fontWeight: item.read ? '400' : '600',
                          color: item.read ? '#6b7280' : '#1f2937',
                          display: 'block',
                        }}
                      >
                        {item.title}
                      </Text>
                      <Text style={{ fontSize: '12px', color: '#9ca3af' }}>{item.time}</Text>
                    </div>
                    <Text
                      className={styles.notificationContent}
                      style={{ display: 'block', marginTop: 4, fontSize: '13px' }}
                    >
                      {item.content}
                    </Text>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </Drawer>

      <Modal
        title={t('header.searchResults')}
        open={searchModalVisible}
        onCancel={() => setSearchModalVisible(false)}
        footer={null}
        width={800}
      >
        {searchResults ? (
          <div>
            <h3>{t('tickets.title')}</h3>
            <div className="search-results">
              {searchResults.tickets.map(item => (
                <div key={item.id} style={{ padding: '8px 0', borderBottom: '1px solid #f0f0f0' }}>
                  <a href={`/tickets/${item.id}`}>{item.title}</a>
                </div>
              ))}
            </div>
            <h3>{t('incidents.title')}</h3>
            <div className="search-results">
              {searchResults.incidents.map(item => (
                <div key={item.id} style={{ padding: '8px 0', borderBottom: '1px solid #f0f0f0' }}>
                  <a href={`/incidents/${item.id}`}>{item.title}</a>
                </div>
              ))}
            </div>
            <h3>{t('problems.title')}</h3>
            <div className="search-results">
              {searchResults.problems.map(item => (
                <div key={item.id} style={{ padding: '8px 0', borderBottom: '1px solid #f0f0f0' }}>
                  <a href={`/problems/${item.id}`}>{item.title}</a>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <p>{t('header.noResults')}</p>
        )}
      </Modal>
    </AntHeader>
  );
};
