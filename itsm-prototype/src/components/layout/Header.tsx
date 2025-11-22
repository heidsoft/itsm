'use client';

import React, { useState } from 'react';
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
  List,
  message,
} from 'antd';
import {
  UserOutlined,
  SearchOutlined,
  LogoutOutlined,
  BellOutlined,
  DownOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore } from '@/lib/store/auth-store';

const { Header: AntHeader } = Layout;
const { Text } = Typography;

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
  const [notificationsOpen, setNotificationsOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [searchValue, setSearchValue] = useState('');

  const handleLogout = () => {
    logout();
    router.push('/login');
    message.success('已安全退出');
  };

  const handleSearch = (value: string) => {
    if (value.trim()) {
      // 这里可以实现全局搜索逻辑
      message.info(`搜索: ${value}`);
      setSearchValue('');
    }
  };

  const userMenuItems = [
    {
      key: 'profile',
      label: '个人资料',
      icon: <UserOutlined />,
      onClick: () => router.push('/profile'),
    },
    {
      key: 'settings',
      label: '设置',
      icon: <SettingOutlined />,
      onClick: () => router.push('/settings'),
    },
    {
      type: 'divider' as const,
    },
    {
      key: 'logout',
      label: '退出登录',
      icon: <LogoutOutlined />,
      onClick: handleLogout,
    },
  ];

  const notifications = [
    {
      id: 1,
      title: '新工单分配',
      content: '您有一个新的高优先级工单需要处理',
      time: '2分钟前',
      read: false,
      type: 'ticket',
      priority: 'high',
    },
    {
      id: 2,
      title: '系统维护通知',
      content: '系统将在今晚进行维护，预计停机2小时',
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
  ];

  const unreadCount = notifications.filter(n => !n.read).length;

  // 获取当前页面标题
  const getCurrentPageTitle = () => {
    if (breadcrumb && breadcrumb.length > 0) {
      return breadcrumb[breadcrumb.length - 1].title;
    }

    // 根据路径获取页面标题
    const pathTitles: Record<string, string> = {
      '/dashboard': '仪表盘',
      '/tickets': '工单管理',
      '/incidents': '事件管理',
      '/problems': '问题管理',
      '/changes': '变更管理',
      '/cmdb': '配置管理',
      '/service-catalog': '服务目录',
      '/knowledge-base': '知识库',
      '/sla': 'SLA管理',
      '/reports': '报表分析',
      '/workflow': '工作流管理',
      '/admin': '系统管理',
    };

    // 检查是否有匹配的路径前缀
    for (const [path, title] of Object.entries(pathTitles)) {
      if (pathname.startsWith(path)) {
        return title;
      }
    }

    return 'ITSM系统';
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
        dashboard: '仪表盘',
        tickets: '工单管理',
        incidents: '事件管理',
        problems: '问题管理',
        changes: '变更管理',
        cmdb: '配置管理',
        'service-catalog': '服务目录',
        'service-catalogs': '服务目录',
        'knowledge-base': '知识库',
        sla: 'SLA管理',
        reports: '报表分析',
        workflow: '工作流管理',
        admin: '系统管理',
        create: '创建',
        edit: '编辑',
        detail: '详情',
        templates: '模板',
        instances: '实例',
        versions: '版本',
        automation: '自动化',
        approval: '审批',
        users: '用户管理',
        roles: '角色管理',
        permissions: '权限管理',
        groups: '用户组',
        tenants: '租户管理',
        'escalation-rules': '升级规则',
        'approval-chains': '审批链',
        'service-catalogs': '服务目录',
        'sla-definitions': 'SLA定义',
        'system-config': '系统配置',
        'ticket-categories': '工单分类',
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
    <AntHeader
      style={{
        padding: '0 16px',
        background: '#fff',
        borderBottom: '1px solid #f0f0f0',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        height: 56,
        boxShadow: '0 1px 2px 0 rgba(0, 0, 0, 0.05)',
      }}
    >
      {/* 左侧区域 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
        {/* 折叠按钮 */}
        <Button
          type='text'
          icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
          onClick={() => onCollapse(!collapsed)}
          style={{
            fontSize: '16px',
            width: 36,
            height: 36,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            borderRadius: '6px',
          }}
        />

        {/* 当前页面标题 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Text
            style={{
              fontSize: '16px',
              fontWeight: '600',
              color: '#1f2937',
              marginRight: 8,
            }}
          >
            {getCurrentPageTitle()}
          </Text>

          {/* 智能面包屑导航 - 只在有多级路径时显示 */}
          {showBreadcrumb && smartBreadcrumb.length > 1 && (
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              {smartBreadcrumb.map((item, index) => (
                <React.Fragment key={index}>
                  {index > 0 && <Text style={{ color: '#9ca3af', margin: '0 4px' }}>/</Text>}
                  {item.href ? (
                    <Text
                      style={{
                        color: '#3b82f6',
                        cursor: 'pointer',
                        fontSize: '14px',
                        fontWeight: '500',
                        transition: 'all 0.2s',
                        padding: '4px 8px',
                        borderRadius: '6px',
                      }}
                      onClick={() => router.push(item.href!)}
                      onMouseEnter={e => {
                        e.currentTarget.style.backgroundColor = '#eff6ff';
                        e.currentTarget.style.color = '#1d4ed8';
                      }}
                      onMouseLeave={e => {
                        e.currentTarget.style.backgroundColor = 'transparent';
                        e.currentTarget.style.color = '#3b82f6';
                      }}
                    >
                      {item.title}
                    </Text>
                  ) : (
                    <Text
                      style={{
                        color: '#374151',
                        fontSize: '14px',
                        fontWeight: '600',
                        backgroundColor: '#f3f4f6',
                        padding: '4px 8px',
                        borderRadius: '6px',
                      }}
                    >
                      {item.title}
                    </Text>
                  )}
                </React.Fragment>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* 右侧区域 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        {/* 搜索框 */}
        <Input
          placeholder='搜索工单、知识库...'
          prefix={<SearchOutlined style={{ color: '#9ca3af' }} />}
          value={searchValue}
          onChange={e => setSearchValue(e.target.value)}
          onPressEnter={() => handleSearch(searchValue)}
          size="small"
          style={{
            width: 240,
            borderRadius: '6px',
            border: '1px solid #e5e7eb',
            transition: 'all 0.2s',
          }}
          onFocus={e => {
            e.target.style.borderColor = '#3b82f6';
            e.target.style.boxShadow = '0 0 0 2px rgba(59, 130, 246, 0.1)';
          }}
          onBlur={e => {
            e.target.style.borderColor = '#e5e7eb';
            e.target.style.boxShadow = 'none';
          }}
        />

        {/* 通知 */}
        <Tooltip title='通知中心'>
          <Badge count={unreadCount} size='small' offset={[-2, 2]}>
            <Button
              type='text'
              icon={<BellOutlined />}
              onClick={() => setNotificationsOpen(true)}
              style={{
                width: 36,
                height: 36,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                borderRadius: '6px',
                position: 'relative',
              }}
            />
          </Badge>
        </Tooltip>

        {/* 用户菜单 */}
        <Dropdown
          menu={{ items: userMenuItems }}
          placement='bottomRight'
          trigger={['click']}
          open={userMenuOpen}
          onOpenChange={setUserMenuOpen}
        >
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              cursor: 'pointer',
              padding: '6px 12px',
              borderRadius: '8px',
              transition: 'all 0.2s',
              border: '1px solid transparent',
            }}
            onMouseEnter={e => {
              e.currentTarget.style.backgroundColor = '#f3f4f6';
              e.currentTarget.style.borderColor = '#e5e7eb';
            }}
            onMouseLeave={e => {
              e.currentTarget.style.backgroundColor = 'transparent';
              e.currentTarget.style.borderColor = 'transparent';
            }}
          >
            <Avatar
              size={28}
              style={{
                background: 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)',
                color: '#fff',
                fontWeight: '600',
                fontSize: '12px',
              }}
            >
              {user?.name?.[0] || user?.username?.[0] || 'U'}
            </Avatar>
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'flex-start',
              }}
            >
              <Text
                style={{
                  fontSize: '13px',
                  fontWeight: '500',
                  color: '#1f2937',
                  lineHeight: '1.2',
                }}
              >
                {user?.name || user?.username}
              </Text>
              <Text style={{ fontSize: '11px', color: '#6b7280', lineHeight: '1.2' }}>
                {user?.role === 'admin' ? '管理员' : '用户'}
              </Text>
            </div>
            <DownOutlined style={{ color: '#9ca3af' }} />
          </div>
        </Dropdown>
      </div>

      {/* 通知抽屉 */}
      <Drawer
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <BellOutlined style={{ color: '#3b82f6', fontSize: 20 }} />
            <span>通知中心</span>
            {unreadCount > 0 && <Badge count={unreadCount} size='small' />}
          </div>
        }
        placement='right'
        width={400}
        open={notificationsOpen}
        onClose={() => setNotificationsOpen(false)}
        styles={{
          header: {
            borderBottom: '1px solid #f3f4f6',
            paddingBottom: '16px',
          },
        }}
      >
        <List
          dataSource={notifications}
          renderItem={item => (
            <List.Item
              style={{
                padding: '16px 0',
                borderBottom: '1px solid #f3f4f6',
                opacity: item.read ? 0.7 : 1,
                cursor: 'pointer',
                transition: 'all 0.2s',
              }}
              onMouseEnter={e => {
                e.currentTarget.style.backgroundColor = '#f9fafb';
              }}
              onMouseLeave={e => {
                e.currentTarget.style.backgroundColor = 'transparent';
              }}
            >
              <List.Item.Meta
                avatar={
                  <div
                    style={{
                      width: 32,
                      height: 32,
                      borderRadius: '50%',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      backgroundColor:
                        item.priority === 'urgent'
                          ? '#ef4444'
                          : item.priority === 'high'
                          ? '#f59e0b'
                          : '#3b82f6',
                      color: 'white',
                      fontSize: '12px',
                      fontWeight: '600',
                    }}
                  >
                    {item.type === 'ticket' ? 'T' : item.type === 'system' ? 'S' : 'SLA'}
                  </div>
                }
                title={
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                    }}
                  >
                    <Text
                      style={{
                        fontWeight: item.read ? '400' : '600',
                        color: item.read ? '#6b7280' : '#1f2937',
                      }}
                    >
                      {item.title}
                    </Text>
                    <Text style={{ fontSize: '12px', color: '#9ca3af' }}>{item.time}</Text>
                  </div>
                }
                description={
                  <Text
                    style={{
                      color: '#6b7280',
                      fontSize: '14px',
                      lineHeight: '1.5',
                    }}
                  >
                    {item.content}
                  </Text>
                }
              />
            </List.Item>
          )}
        />

        {notifications.length === 0 && (
          <div
            style={{
              textAlign: 'center',
              padding: '40px 20px',
              color: '#9ca3af',
            }}
          >
            <BellOutlined style={{ fontSize: 48, marginBottom: '16px', opacity: 0.5 }} />
            <Text>暂无通知</Text>
          </div>
        )}
      </Drawer>
    </AntHeader>
  );
};
