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
  Tag,
  List,
  Divider,
  Space,
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
  Ticket,
  AlertTriangle,
  Calendar,
  Clock,
  ArrowRight,
  Zap,
  Mail,
  Phone,
  Building,
  LogIn,
} from 'lucide-react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore, useAuthStoreHydration } from '@/lib/store/auth-store';
import styles from './Header.module.css';
import { useI18n } from '@/lib/i18n';
import { globalSearchApi, GlobalSearchResult } from '@/lib/api/global-search-api';

const { Header: AntHeader } = Layout;
const { Text, Title } = Typography;

// 独特的设计系统
const DESIGN = {
  colors: {
    primary: '#0f172a',
    accent: '#3b82f6',
    success: '#10b981',
    warning: '#f59e0b',
    danger: '#ef4444',
    surface: '#ffffff',
    border: '#e2e8f0',
    text: '#1e293b',
    textMuted: '#64748b',
    bgSubtle: '#f8fafc',
  },
  shadows: {
    dropdown: '0 10px 40px -10px rgba(0,0,0,0.15)',
    glow: (color: string) => `0 0 20px ${color}20`,
  },
  radius: {
    sm: '8px',
    md: '12px',
    lg: '16px',
    full: '9999px',
  },
};

const { Header: AntDesignHeader } = Layout;

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
  useAuthStoreHydration();

  const [notificationsOpen, setNotificationsOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const [notifications, setNotifications] = useState([
    {
      id: 1,
      title: '新工单已分配给您',
      content: '工单 #1234 需要您的处理',
      time: '2分钟前',
      read: false,
      type: 'ticket',
      priority: 'high',
    },
    {
      id: 2,
      title: '系统维护通知',
      content: '系统将于今晚22:00进行维护',
      time: '1小时前',
      read: true,
      type: 'system',
      priority: 'medium',
    },
    {
      id: 3,
      title: 'SLA预警',
      content: '工单 #1234 即将超时，请及时处理',
      time: '3小时前',
      read: false,
      type: 'sla',
      priority: 'urgent',
    },
  ]);

  const [searchModalVisible, setSearchModalVisible] = useState(false);
  const [searchResults, setSearchResults] = useState<GlobalSearchResult | null>(null);
  const [isClient, setIsClient] = useState(false);

  useEffect(() => {
    setIsClient(true);
  }, []);

  const displayName = user?.name || user?.username || '';
  const userInitial = displayName.charAt(0).toUpperCase() || 'U';
  const roleText =
    user?.role === 'admin'
      ? '管理员'
      : user?.role === 'super_admin'
        ? '超级管理员'
        : '用户';

  const roleColor = user?.role === 'admin' || user?.role === 'super_admin' ? '#3b82f6' : '#64748b';

  const handleLogout = () => {
    logout();
    router.push('/login');
    message.success('已退出登录');
  };

  const handleSearch = async (value: string) => {
    if (value.trim()) {
      const results = await globalSearchApi.search(value);
      setSearchResults(results);
      setSearchValue('');
      setSearchModalVisible(true);
    }
  };

  const markAsRead = (id: number) => {
    setNotifications(prev => prev.map(n => (n.id === id ? { ...n, read: true } : n)));
  };

  const markAllAsRead = () => {
    setNotifications(prev => prev.map(n => ({ ...n, read: true })));
    message.success('已全部标记为已读');
  };

  const unreadCount = notifications.filter(n => !n.read).length;

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'urgent': return DESIGN.colors.danger;
      case 'high': return DESIGN.colors.warning;
      case 'medium': return DESIGN.colors.accent;
      default: return DESIGN.colors.textMuted;
    }
  };

  const getNotificationIcon = (type: string) => {
    switch (type) {
      case 'ticket': return <Ticket size={16} />;
      case 'sla': return <AlertTriangle size={16} />;
      default: return <Zap size={16} />;
    }
  };

  // 用户菜单项
  const userMenuItems = [
    {
      key: 'profile',
      label: (
        <div style={{ padding: '8px 0', minWidth: 160 }}>
          <div style={{ fontWeight: 600, marginBottom: 4 }}>{displayName}</div>
          <div style={{ fontSize: 12, color: DESIGN.colors.textMuted }}>{user?.email}</div>
        </div>
      ),
      disabled: true,
    },
    { type: 'divider' as const },
    {
      key: 'profile',
      label: (
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '8px 0' }}>
          <User size={16} />
          <span>个人中心</span>
        </div>
      ),
      onClick: () => router.push('/profile'),
    },
    {
      key: 'settings',
      label: (
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '8px 0' }}>
          <Settings size={16} />
          <span>设置</span>
        </div>
      ),
      onClick: () => router.push('/profile'),
    },
    { type: 'divider' as const },
    {
      key: 'logout',
      label: (
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, color: DESIGN.colors.danger }}>
          <LogOut size={16} />
          <span>退出登录</span>
        </div>
      ),
      onClick: handleLogout,
    },
  ];

  // 通知项
  const NotificationItem = ({ item }: { item: typeof notifications[0] }) => (
    <div
      onClick={() => markAsRead(item.id)}
      style={{
        padding: '16px',
        borderBottom: `1px solid ${DESIGN.colors.border}`,
        background: item.read ? 'transparent' : `${DESIGN.colors.accent}08`,
        cursor: 'pointer',
        transition: 'all 0.2s',
      }}
      className="notification-item"
    >
      <div style={{ display: 'flex', gap: 12 }}>
        {/* 图标 */}
        <div
          style={{
            width: 40,
            height: 40,
            borderRadius: DESIGN.radius.md,
            background: `${getPriorityColor(item.priority)}15`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: getPriorityColor(item.priority),
            flexShrink: 0,
          }}
        >
          {getNotificationIcon(item.type)}
        </div>

        {/* 内容 */}
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 4 }}>
            <Text strong style={{ fontSize: 14, color: item.read ? DESIGN.colors.textMuted : DESIGN.colors.text }}>
              {item.title}
            </Text>
            {!item.read && (
              <div
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  background: DESIGN.colors.accent,
                  flexShrink: 0,
                  marginLeft: 8,
                }}
              />
            )}
          </div>
          <Text style={{ fontSize: 13, color: DESIGN.colors.textMuted, display: 'block', marginBottom: 4 }}>
            {item.content}
          </Text>
          <Text style={{ fontSize: 12, color: DESIGN.colors.textMuted }}>
            {item.time}
          </Text>
        </div>
      </div>

      <style>{`
        .notification-item:hover {
          background: ${DESIGN.colors.bgSubtle} !important;
        }
      `}</style>
    </div>
  );

  return (
    <AntDesignHeader className={styles.header} style={{ background: DESIGN.colors.surface }}>
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
        <div style={{ position: 'relative' }}>
          <Input
            placeholder="搜索工单、事件..."
            prefix={<Search size={16} style={{ color: DESIGN.colors.textMuted }} />}
            value={searchValue}
            onChange={e => setSearchValue(e.target.value)}
            onPressEnter={() => handleSearch(searchValue)}
            style={{
              width: 200,
              height: 36,
              borderRadius: DESIGN.radius.full,
              border: `1px solid ${DESIGN.colors.border}`,
              background: DESIGN.colors.bgSubtle,
            }}
          />
          <kbd
            style={{
              position: 'absolute',
              right: 8,
              top: '50%',
              transform: 'translateY(-50%)',
              fontSize: 11,
              padding: '2px 6px',
              borderRadius: 4,
              background: DESIGN.colors.surface,
              border: `1px solid ${DESIGN.colors.border}`,
              color: DESIGN.colors.textMuted,
            }}
          >
            /
          </kbd>
        </div>

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

        {/* 用户菜单 */}
        {isClient ? (
          user ? (
            <Dropdown
              menu={{ items: userMenuItems }}
              placement="bottomRight"
              trigger={['click']}
              open={userMenuOpen}
              onOpenChange={setUserMenuOpen}
              overlayStyle={{ padding: 0 }}
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 10,
                  padding: '6px 12px 6px 6px',
                  borderRadius: DESIGN.radius.full,
                  background: DESIGN.colors.bgSubtle,
                  cursor: 'pointer',
                  transition: 'all 0.2s',
                  border: `1px solid ${userMenuOpen ? DESIGN.colors.accent : 'transparent'}`,
                }}
              >
                <Avatar
                  size={32}
                  style={{
                    background: `linear-gradient(135deg, ${DESIGN.colors.accent} 0%, #1d4ed8 100%)`,
                    fontSize: 14,
                    fontWeight: 600,
                  }}
                >
                  {userInitial}
                </Avatar>
                <div style={{ lineHeight: 1.3 }}>
                  <div style={{ fontSize: 13, fontWeight: 600, color: DESIGN.colors.text }}>
                    {displayName}
                  </div>
                  <Tag
                    color={roleColor}
                    style={{
                      fontSize: 10,
                      padding: '0 6px',
                      lineHeight: '16px',
                      margin: 0,
                      border: 'none',
                    }}
                  >
                    {roleText}
                  </Tag>
                </div>
                <ChevronDown
                  size={14}
                  style={{
                    color: DESIGN.colors.textMuted,
                    transition: 'transform 0.2s',
                    transform: userMenuOpen ? 'rotate(180deg)' : 'none',
                  }}
                />
              </div>
            </Dropdown>
          ) : (
            <Button
              type="primary"
              icon={<LogIn size={16} />}
              onClick={() => router.push('/login')}
              style={{
                borderRadius: DESIGN.radius.md,
                height: 36,
                background: `linear-gradient(135deg, ${DESIGN.colors.accent} 0%, #1d4ed8 100%)`,
                boxShadow: DESIGN.shadows.glow(DESIGN.colors.accent),
              }}
            >
              登录
            </Button>
          )
        ) : (
          <div style={{ width: 120, height: 36, background: DESIGN.colors.bgSubtle, borderRadius: DESIGN.radius.full }} />
        )}
      </div>

      {/* 通知抽屉 */}
      <Drawer
        title={
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
              <div
                style={{
                  width: 36,
                  height: 36,
                  borderRadius: DESIGN.radius.md,
                  background: `linear-gradient(135deg, ${DESIGN.colors.accent} 0%, #1d4ed8 100%)`,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: '#fff',
                }}
              >
                <Bell size={18} />
              </div>
              <div>
                <Title level={5} style={{ margin: 0, fontSize: 16 }}>通知中心</Title>
                <Text style={{ fontSize: 12, color: DESIGN.colors.textMuted }}>
                  {unreadCount > 0 ? `${unreadCount} 条未读` : '暂无未读'}
                </Text>
              </div>
            </div>
            {unreadCount > 0 && (
              <Button
                type="link"
                icon={<CheckCheck size={16} />}
                onClick={markAllAsRead}
                style={{ color: DESIGN.colors.accent }}
              >
                全部已读
              </Button>
            )}
          </div>
        }
        placement="right"
        size="large"
        open={notificationsOpen}
        onClose={() => setNotificationsOpen(false)}
        styles={{ body: { padding: 0 } }}
      >
        <div style={{ maxHeight: 'calc(100vh - 120px)', overflow: 'auto' }}>
          {notifications.length === 0 ? (
            <div style={{ padding: '60px 20px', textAlign: 'center' }}>
              <Bell size={48} style={{ color: DESIGN.colors.border, marginBottom: 16 }} />
              <Text style={{ color: DESIGN.colors.textMuted }}>暂无通知</Text>
            </div>
          ) : (
            notifications.map(item => <NotificationItem key={item.id} item={item} />)
          )}
        </div>

        {/* 底部链接 */}
        <div
          style={{
            padding: '16px',
            borderTop: `1px solid ${DESIGN.colors.border}`,
            textAlign: 'center',
          }}
        >
          <Button
            type="link"
            onClick={() => {
              setNotificationsOpen(false);
              router.push('/notifications');
            }}
            style={{ color: DESIGN.colors.accent }}
          >
            查看全部通知 <ArrowRight size={14} />
          </Button>
        </div>
      </Drawer>

      {/* 搜索结果弹窗 */}
      <Drawer
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <Search size={18} style={{ color: DESIGN.colors.accent }} />
            <span>搜索结果</span>
          </div>
        }
        placement="top"
        height={500}
        open={searchModalVisible}
        onClose={() => setSearchModalVisible(false)}
      >
        {searchResults ? (
          <div style={{ padding: '0 20px' }}>
            <List
              header={<strong>工单</strong>}
              dataSource={searchResults.tickets}
              renderItem={item => (
                <List.Item>
                  <List.Item.Meta
                    avatar={<Ticket size={16} style={{ color: DESIGN.colors.accent }} />}
                    title={<a onClick={() => { setSearchModalVisible(false); router.push(`/tickets/${item.id}`); }}>{item.title}</a>}
                    description={item.status}
                  />
                </List.Item>
              )}
            />
          </div>
        ) : (
          <div style={{ padding: 40, textAlign: 'center', color: DESIGN.colors.textMuted }}>
            输入关键词搜索...
          </div>
        )}
      </Drawer>
    </AntDesignHeader>
  );
};
