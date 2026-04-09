'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Layout, Button, Tooltip, Badge, Dropdown, message } from 'antd';
import { PanelLeftClose, PanelLeftOpen, Bell, Globe } from 'lucide-react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore, useAuthStoreHydration } from '@/lib/store/auth-store';
import { DESIGN } from '@/design-system/tokens';
import { useI18n } from '@/lib/i18n';
import { TicketNotificationApi, type TicketNotification } from '@/lib/api/ticket-notification-api';
import { globalSearch, type GlobalSearchResponse } from '@/lib/api/global-search-api';
import { notificationWS } from '@/lib/services/notification-ws';
import { UserMenuDropdown } from './UserMenuDropdown';
import { NotificationDrawer } from './NotificationDrawer';
import { GlobalSearch, SearchInput } from './GlobalSearch';
import styles from './Header.module.css';

const { Header: AntHeader } = Layout;

interface HeaderProps {
  collapsed: boolean;
  onCollapse: (collapsed: boolean) => void;
  title?: string;
  breadcrumb?: Array<{ title: string; href?: string }>;
  showBackButton?: boolean;
  extra?: React.ReactNode;
  showBreadcrumb?: boolean;
}

export const Header: React.FC<HeaderProps> = ({ collapsed, onCollapse }) => {
  const router = useRouter();
  const pathname = usePathname();
  const { user, logout, token } = useAuthStore();
  const { language, changeLanguage } = useI18n();
  useAuthStoreHydration();

  // UI 状态
  const [notificationsOpen, setNotificationsOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [searchModalVisible, setSearchModalVisible] = useState(false);
  const [searchResults, setSearchResults] = useState<GlobalSearchResponse | null>(null);
  const [isClient, setIsClient] = useState(false);

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
        page_size: 10,
      });
      setNotifications(response.notifications || []);
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
      notificationWS.connect(user.id, token).catch(err => {
        console.error('Notification WebSocket connection failed:', err);
      });

      const unsubscribe = notificationWS.onNotification((notification: TicketNotification) => {
        setNotifications(prev => [notification, ...prev]);
        message.info(notification.content);
      });

      return () => {
        unsubscribe();
        notificationWS.disconnect();
      };
    }
  }, [user?.id, token, loadNotifications]);

  // 定期刷新通知（每30秒）
  useEffect(() => {
    if (!user?.id) return;
    const interval = setInterval(loadNotifications, 30000);
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
    if (value.trim()) {
      const results = await globalSearch(value);
      setSearchResults(results);
      setSearchModalVisible(true);
    }
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
          <span>🇨🇳</span>
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
          <span>🇺🇸</span>
          <span>English</span>
          {language === 'en-US' && <span style={{ color: DESIGN.colors.accent }}>✓</span>}
        </div>
      ),
      onClick: () => changeLanguage('en-US'),
    },
  ];

  return (
    <AntHeader className={styles.header} style={{ background: DESIGN.colors.surface }}>
      {/* 左侧 */}
      <div className={styles.left}>
        <Button
          type="text"
          icon={collapsed ? <PanelLeftOpen size={18} /> : <PanelLeftClose size={18} />}
          onClick={() => onCollapse(!collapsed)}
          className={styles.collapseButton}
          style={{
            width: 40,
            height: 40,
            borderRadius: DESIGN.radius.md,
          }}
        />
      </div>

      {/* 右侧 */}
      <div className={styles.right}>
        {/* 搜索 */}
        <SearchInput
          value=""
          onChange={() => {}}
          onSearch={handleSearch}
          onFocus={() => setSearchModalVisible(true)}
        />

        {/* 通知 */}
        <Tooltip title="通知中心">
          <Badge count={unreadCount} size="small" offset={[-2, 2]}>
            <Button
              type="text"
              onClick={() => setNotificationsOpen(true)}
              style={{
                width: 40,
                height: 40,
                borderRadius: DESIGN.radius.md,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <Bell size={20} style={{ color: DESIGN.colors.textMuted }} />
            </Button>
          </Badge>
        </Tooltip>

        {/* 语言切换 */}
        <Dropdown menu={{ items: languageItems }} placement="bottomRight" trigger={['click']}>
          <Tooltip title={language === 'zh-CN' ? '切换语言' : 'Switch Language'}>
            <Button
              type="text"
              style={{
                width: 40,
                height: 40,
                borderRadius: DESIGN.radius.md,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <Globe size={20} style={{ color: DESIGN.colors.textMuted }} />
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
      <GlobalSearch open={searchModalVisible} onClose={() => setSearchModalVisible(false)} />
    </AntHeader>
  );
};
