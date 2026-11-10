'use client';

import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Layout, Button, Tooltip, Badge, message, Breadcrumb } from 'antd';
import { PanelLeftClose, PanelLeftOpen, Bell, Bot, Home, Moon, Sun } from 'lucide-react';
import { useTheme } from '@/lib/design-system/theme';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore, useAuthStoreHydration } from '@/lib/store/auth-store';
import { AuthService } from '@/lib/services/auth-service';
import { DESIGN } from '@/design-system/tokens';
import {
  TicketNotificationApi,
  toTicketNotification,
  type TicketNotification,
  type UserNotification,
} from '@/lib/api/ticket-notification-api';
import type { GlobalSearchResponse } from '@/lib/api/global-search-api';
import { notificationWS } from '@/lib/services/notification-ws';
import { UserMenuDropdown } from './UserMenuDropdown';
import { NotificationDrawer } from './NotificationDrawer';
import { GlobalSearch, SearchInput } from './GlobalSearch';
import { buildBreadcrumb, useBuildBreadcrumb } from './breadcrumb-utils';
import styles from './Header.module.css';

const { Header: AntHeader } = Layout;

// 常量定义
const NOTIFICATION_REFRESH_INTERVAL_MS = 30000; // 30秒
const NOTIFICATION_PAGE_SIZE = 10;

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
  showBreadcrumb = false,
}) => {
  const router = useRouter();
  const pathname = usePathname();
  const { user, hasPermission, isAdmin } = useAuthStore();
  const { isDark, toggleTheme } = useTheme();
  useAuthStoreHydration();

  // 动态菜单驱动的面包屑：与 Sidebar 共享 useUserMenusQuery 缓存，
  // 未传入自定义 breadcrumb 时优先用 hook 版（带菜单 label），失败时回退到 segment 兜底
  const dynamicBreadcrumb = useBuildBreadcrumb(pathname || '');

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
  // badge 计数使用服务端权威值（/unread-count），而非当前页列表推导，
  // 避免只统计本页导致的计数偏差
  const [unreadCount, setUnreadCount] = useState(0);

  useEffect(() => {
    setIsClient(true);
  }, []);

  // 加载通知
  const loadNotifications = useCallback(async () => {
    if (!user?.id) return;
    setNotificationsLoading(true);
    try {
      const [response, unread] = await Promise.all([
        TicketNotificationApi.getUserNotifications({
          page: 1,
          pageSize: NOTIFICATION_PAGE_SIZE,
        }),
        TicketNotificationApi.getUnreadCount().catch(() => null),
      ]);
      setNotifications(response.items.map(toTicketNotification));
      if (unread && typeof unread.count === 'number') setUnreadCount(unread.count);
    } catch (error) {
      console.error('Failed to load notifications:', error);
    } finally {
      setNotificationsLoading(false);
    }
  }, [user?.id]);

  // 初始化通知和WebSocket
  // 凭证只在 httpOnly cookie 里，会话端点确认后（user.id 存在）即可直接发起请求
  useEffect(() => {
    if (user?.id) {
      loadNotifications();
      notificationWS.acquire(user.id).catch(() => {
        // WebSocket server not available, ignore silently
      });

      const unsubscribe = notificationWS.onNotification((notification: UserNotification) => {
        const normalized = toTicketNotification(notification);
        setNotifications(prev => [normalized, ...prev]);
        if (!notification.read) setUnreadCount(prev => prev + 1);
        message.info(normalized.content);
      });

      return () => {
        unsubscribe();
        notificationWS.release();
      };
    }
  }, [user?.id, loadNotifications]);

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
    AuthService.logout(); // calls backend to clear httpOnly cookies + clears store
    // Use hard navigation to ensure middleware re-checks auth state
    window.location.href = '/login';
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
      setUnreadCount(prev => Math.max(0, prev - 1));
    } catch (error) {
      console.error('Failed to mark notification as read:', error);
    }
  }, []);

  const markAllAsRead = useCallback(async () => {
    try {
      await TicketNotificationApi.markAllNotificationsRead();
      setNotifications(prev => prev.map(n => ({ ...n, status: 'read' as const })));
      setUnreadCount(0);
    } catch (error) {
      console.error('Failed to mark all notifications as read:', error);
    }
  }, []);

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
            aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            className={styles.collapseButton}
            style={{
              width: 36,
              height: 36,
              borderRadius: DESIGN.radius.md,
              flexShrink: 0,
            }}
          />
          {showBreadcrumb && (() => {
            const items = breadcrumb || (dynamicBreadcrumb.length > 1 ? dynamicBreadcrumb : buildBreadcrumb(pathname || ''));
            // Hide breadcrumb on root/dashboard (single item with no navigation value)
            if (items.length <= 1 && (pathname === '/' || pathname === '/dashboard')) return null;
            return (
              <div className={styles.breadcrumb} role="navigation" aria-label="Breadcrumb navigation">
                <Breadcrumb items={items} separator="/" />
              </div>
            );
          })()}
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

          {/* AI 助手入口：权限码与后端 permissions 表对齐（ai:read），admin 默认放行 */}
          {hasPermission('ai:read') || isAdmin() ? (
            <Tooltip title="AI Assistant">
              <Button
                type="text"
                className={styles.actionButton}
                onClick={() => router.push('/ai/chat')}
                aria-label="AI Assistant"
                title="AI Assistant"
              >
                <Bot size={18} />
              </Button>
            </Tooltip>
          ) : null}

          {/* 通知 */}
          <Tooltip title="Notification Center">
            <Badge count={unreadCount} size="small" offset={[-2, 2]}>
              <Button
                type="text"
                className={`${styles.actionButton} ${styles.notificationButton}${notificationsOpen ? ` ${styles.active}` : ''}`}
                onClick={() => setNotificationsOpen(true)}
                aria-label="Notification Center"
                title="Notification Center"
              >
                <Bell size={18} />
              </Button>
            </Badge>
          </Tooltip>

          {/* 主题切换 */}
          <Tooltip title={isDark ? 'Switch to light theme' : 'Switch to dark theme'}>
            <Button
              type="text"
              className={styles.actionButton}
              onClick={toggleTheme}
              aria-label={isDark ? 'Switch to light theme' : 'Switch to dark theme'}
              title={isDark ? 'Switch to light theme' : 'Switch to dark theme'}
            >
              {isDark ? <Sun size={18} /> : <Moon size={18} />}
            </Button>
          </Tooltip>


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
