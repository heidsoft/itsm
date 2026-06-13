'use client';

import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Layout, Button, Tooltip, Badge, Dropdown, message, Breadcrumb } from 'antd';
import { PanelLeftClose, PanelLeftOpen, Bell, Globe, Home, Moon, Sun } from 'lucide-react';
import { useTheme } from '@/lib/design-system/theme';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore, useAuthStoreHydration } from '@/lib/store/auth-store';
import { DESIGN } from '@/design-system/tokens';
import { useI18n } from '@/lib/i18n';
import { TicketNotificationApi, type TicketNotification } from '@/lib/api/ticket-notification-api';
import type { GlobalSearchResponse } from '@/lib/api/global-search-api';
import { notificationWS } from '@/lib/services/notification-ws';
import { UserMenuDropdown } from './UserMenuDropdown';
import { NotificationDrawer } from './NotificationDrawer';
import { GlobalSearch, SearchInput } from './GlobalSearch';
import styles from './Header.module.css';

const { Header: AntHeader } = Layout;

// 常量定义
const NOTIFICATION_REFRESH_INTERVAL_MS = 30000; // 30秒
const NOTIFICATION_PAGE_SIZE = 10;

const normalizeNotification = (notification: TicketNotification): TicketNotification => ({
  ...notification,
  createdAt: notification.createdAt || notification.created_at,
  readAt: notification.readAt || notification.read_at,
  sentAt: notification.sentAt || notification.sent_at,
});

interface HeaderProps {
  collapsed: boolean;
  onCollapse: (collapsed: boolean) => void;
  title?: string;
  breadcrumb?: Array<{ title: string; href?: string }>;
  showBackButton?: boolean;
  extra?: React.ReactNode;
  showBreadcrumb?: boolean;
}

// 路径到面包屑的映射
const pathToBreadcrumb: Record<string, Array<{ title: string; href?: string }>> = {
  '/dashboard': [{ title: '首页', href: '/dashboard' }],
  '/tickets': [{ title: '首页', href: '/dashboard' }, { title: '工单管理' }],
  '/incidents': [{ title: '首页', href: '/dashboard' }, { title: '事件管理' }],
  '/problems': [{ title: '首页', href: '/dashboard' }, { title: '问题管理' }],
  '/changes': [{ title: '首页', href: '/dashboard' }, { title: '变更管理' }],
  '/knowledge': [{ title: '首页', href: '/dashboard' }, { title: '知识库' }],
  '/service-catalog': [{ title: '首页', href: '/dashboard' }, { title: '服务目录' }],
  '/profile': [{ title: '首页', href: '/dashboard' }, { title: '个人中心' }],
  '/notifications': [{ title: '首页', href: '/dashboard' }, { title: '通知中心' }],
};

export const Header: React.FC<HeaderProps> = ({
  collapsed,
  onCollapse,
  breadcrumb,
  showBreadcrumb = false,
}) => {
  const router = useRouter();
  const pathname = usePathname();
  const { user, logout, token } = useAuthStore();
  const { isDark, toggleTheme } = useTheme();
  const { language, changeLanguage } = useI18n();
  useAuthStoreHydration();

  // UI 状态
  const [notificationsOpen, setNotificationsOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [searchModalVisible, setSearchModalVisible] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const [searchResults, setSearchResults] = useState<GlobalSearchResponse | null>(null);
  const [isClient, setIsClient] = useState(false);
  const suppressSearchOpenRef = useRef(false);

  // 通知状态
  const [notifications, setNotifications] = useState<TicketNotification[]>([]);
  const [notificationsLoading, setNotificationsLoading] = useState(false);

  useEffect(() => {
    setIsClient(true);
  }, []);

  // 加载通知
  const loadNotifications = useCallback(async () => {
    if (!user?.id) return;
    setNotificationsLoading(true);
    try {
      const response = await TicketNotificationApi.getUserNotifications({
        page: 1,
        page_size: NOTIFICATION_PAGE_SIZE,
      });
      setNotifications((response.notifications || []).map(normalizeNotification));
    } catch (error) {
      console.error('Failed to load notifications:', error);
    } finally {
      setNotificationsLoading(false);
    }
  }, [user?.id]);

  // 初始化通知和WebSocket
  useEffect(() => {
    if (user?.id && token) {
      loadNotifications();
      notificationWS.connect(user.id, token).catch(() => {
        // WebSocket server not available, ignore silently
      });

      const unsubscribe = notificationWS.onNotification((notification: TicketNotification) => {
        const normalized = normalizeNotification(notification);
        setNotifications(prev => [normalized, ...prev]);
        message.info(notification.content);
      });

      return () => {
        unsubscribe();
        notificationWS.disconnect();
      };
    }
  }, [user?.id, token, loadNotifications]);

  // 定期刷新通知
  useEffect(() => {
    if (!user?.id) return;
    const interval = setInterval(loadNotifications, NOTIFICATION_REFRESH_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [user?.id, loadNotifications]);

  // Ctrl+K 快捷键打开全局搜索
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        setSearchModalVisible(true);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  // 登出处理
  const handleLogout = () => {
    logout();
    router.push('/login');
  };

  // 搜索处理
  const handleSearch = async (value: string) => {
    const keyword = value.trim();
    if (keyword) {
      setSearchValue(keyword);
      setSearchResults(null);
      setSearchModalVisible(true);
    }
  };

  const handleOpenSearch = () => {
    if (suppressSearchOpenRef.current) return;
    setSearchModalVisible(true);
  };

  const handleCloseSearch = () => {
    suppressSearchOpenRef.current = true;
    setSearchModalVisible(false);

    window.setTimeout(() => {
      suppressSearchOpenRef.current = false;
    }, 1500);
  };

  // 标记已读
  const markAsRead = useCallback(async (id: number) => {
    try {
      await TicketNotificationApi.markNotificationRead(id);
      setNotifications(prev =>
        prev.map(n => (n.id === id ? { ...n, status: 'read' as const } : n))
      );
    } catch (error) {
      console.error('Failed to mark notification as read:', error);
    }
  }, []);

  const markAllAsRead = useCallback(async () => {
    try {
      await TicketNotificationApi.markAllNotificationsRead();
      setNotifications(prev => prev.map(n => ({ ...n, status: 'read' as const })));
    } catch (error) {
      console.error('Failed to mark all notifications as read:', error);
    }
  }, []);

  const unreadCount = notifications.filter(n => n.status !== 'read').length;

  // 语言切换菜单
  const languageItems = [
    {
      key: 'zh-CN',
      label: (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 14 }}>中</span>
          <span>中文</span>
          {language === 'zh-CN' && <span style={{ color: DESIGN.colors.accent }}>✓</span>}
        </div>
      ),
      onClick: () => changeLanguage('zh-CN'),
    },
    {
      key: 'en-US',
      label: (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 14 }}>En</span>
          <span>English</span>
          {language === 'en-US' && <span style={{ color: DESIGN.colors.accent }}>✓</span>}
        </div>
      ),
      onClick: () => changeLanguage('en-US'),
    },
  ];

  return (
    <AntHeader className={styles.header} style={{ background: DESIGN.colors.surface }}>
      {/* 主行：面包屑/收缩按钮 + 右侧工具 */}
      <div className={styles.mainRow}>
        {/* 左侧：收缩按钮 + 面包屑 */}
        <div className={styles.left}>
          <Button
            type="text"
            icon={collapsed ? <PanelLeftOpen size={18} /> : <PanelLeftClose size={18} />}
            onClick={() => onCollapse(!collapsed)}
            aria-label={collapsed ? '展开侧边栏' : '收起侧边栏'}
            title={collapsed ? '展开侧边栏' : '收起侧边栏'}
            className={styles.collapseButton}
            style={{
              width: 36,
              height: 36,
              borderRadius: DESIGN.radius.md,
              flexShrink: 0,
            }}
          />
          {showBreadcrumb && (
            <div className={styles.breadcrumb} role="navigation" aria-label="面包屑导航">
              <Breadcrumb
                items={
                  breadcrumb ||
                  pathToBreadcrumb[pathname] || [{ title: '首页', href: '/dashboard' }]
                }
                separator="/"
              />
            </div>
          )}
        </div>

        {/* 右侧 */}
        <div className={styles.right}>
          {/* 搜索 */}
          <SearchInput
            value={searchValue}
            onChange={value => {
              setSearchValue(value);
              setSearchResults(null);
            }}
            onSearch={handleSearch}
            onOpen={handleOpenSearch}
          />

          {/* 通知 */}
          <Tooltip title="通知中心">
            <Badge count={unreadCount} size="small" offset={[-2, 2]}>
              <Button
                type="text"
                className={styles.notificationButton}
                onClick={() => setNotificationsOpen(true)}
                aria-label="通知中心"
                title="通知中心"
                style={{
                  width: 36,
                  height: 36,
                  borderRadius: DESIGN.radius.md,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <Bell size={18} style={{ color: DESIGN.colors.textMuted }} />
              </Button>
            </Badge>
          </Tooltip>

          {/* 主题切换 */}
          <Tooltip title={isDark ? '切换到亮色' : '切换到暗色'}>
            <Button
              type="text"
              onClick={toggleTheme}
              aria-label={isDark ? '切换到亮色' : '切换到暗色'}
              title={isDark ? '切换到亮色' : '切换到暗色'}
              style={{
                width: 36,
                height: 36,
                borderRadius: DESIGN.radius.md,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              {isDark ? (
                <Sun size={18} style={{ color: DESIGN.colors.textMuted }} />
              ) : (
                <Moon size={18} style={{ color: DESIGN.colors.textMuted }} />
              )}
            </Button>
          </Tooltip>

          {/* 语言切换 */}
          <Dropdown menu={{ items: languageItems }} placement="bottomRight" trigger={['click']}>
            <Tooltip title={language === 'zh-CN' ? '切换语言' : 'Switch Language'}>
              <Button
                type="text"
                aria-label={language === 'zh-CN' ? '切换语言' : 'Switch Language'}
                title={language === 'zh-CN' ? '切换语言' : 'Switch Language'}
                style={{
                  width: 36,
                  height: 36,
                  borderRadius: DESIGN.radius.md,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <Globe size={18} style={{ color: DESIGN.colors.textMuted }} />
              </Button>
            </Tooltip>
          </Dropdown>

          {/* 用户菜单 */}
          {isClient && (
            <UserMenuDropdown
              open={userMenuOpen}
              onOpenChange={setUserMenuOpen}
              onLogout={handleLogout}
            />
          )}
        </div>
      </div>

      {/* 通知抽屉 */}
      <NotificationDrawer
        open={notificationsOpen}
        onClose={() => setNotificationsOpen(false)}
        notifications={notifications}
        unreadCount={unreadCount}
        onMarkAsRead={markAsRead}
        onMarkAllAsRead={markAllAsRead}
        onViewAll={() => {
          setNotificationsOpen(false);
          router.push('/notifications');
        }}
        loading={notificationsLoading}
      />

      {/* 全局搜索 */}
      <GlobalSearch
        open={searchModalVisible}
        onClose={handleCloseSearch}
        initialKeyword={searchValue}
        initialResults={searchResults}
      />
    </AntHeader>
  );
};
