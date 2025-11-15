'use client';

import React from 'react';
import { Layout, Menu, theme, Badge } from 'antd';
import {
  LayoutDashboard,
  FileText,
  AlertTriangle,
  BookOpen,
  BarChart3,
  Database,
  HelpCircle,
  Calendar,
  Workflow,
  Shield,
  TrendingUp,
} from 'lucide-react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore } from '@/lib/store/auth-store';
import { LAYOUT_CONFIG } from '@/config/layout.config';

const { Sider } = Layout;

// 菜单配置
const MENU_CONFIG = {
  main: [
    {
      key: '/dashboard',
      icon: <LayoutDashboard size={18} />,
      label: '仪表盘',
      path: '/dashboard',
      permission: 'dashboard:view',
      description: '系统概览和关键指标',
    },
    {
      key: '/tickets',
      icon: <FileText size={18} />,
      label: '工单管理',
      path: '/tickets',
      permission: 'ticket:view',
      description: '管理和跟踪IT工单',
      badge: 'New',
    },
    {
      key: '/incidents',
      icon: <AlertTriangle size={18} />,
      label: '事件管理',
      path: '/incidents',
      permission: 'incident:view',
      description: '处理IT事件和故障',
    },
    {
      key: '/problems',
      icon: <HelpCircle size={18} />,
      label: '问题管理',
      path: '/problems',
      permission: 'problem:view',
      description: '分析根本原因和解决方案',
    },
    {
      key: '/changes',
      icon: <BarChart3 size={18} />,
      label: '变更管理',
      path: '/changes',
      permission: 'change:view',
      description: '管理IT变更和发布',
    },
    {
      key: '/cmdb',
      icon: <Database size={18} />,
      label: '配置管理',
      path: '/cmdb',
      permission: 'cmdb:view',
      description: 'IT资产和配置项管理',
    },
    {
      key: '/service-catalog',
      icon: <BookOpen size={18} />,
      label: '服务目录',
      path: '/service-catalog',
      permission: 'service:view',
      description: 'IT服务目录和请求',
    },
    {
      key: '/knowledge-base',
      icon: <HelpCircle size={18} />,
      label: '知识库',
      path: '/knowledge-base',
      permission: 'knowledge:view',
      description: '技术文档和解决方案',
    },
    {
      key: '/sla',
      icon: <Calendar size={18} />,
      label: 'SLA管理',
      path: '/sla',
      permission: 'sla:view',
      description: '服务级别协议管理',
    },
    {
      key: '/reports',
      icon: <TrendingUp size={18} />,
      label: '报表分析',
      path: '/reports',
      permission: 'report:view',
      description: '数据分析和报表',
    },
  ],
  admin: [
    {
      key: '/workflow',
      icon: <Workflow size={18} />,
      label: '工作流管理',
      path: '/workflow',
      permission: 'workflow:config',
      description: '配置业务流程和自动化',
    },
    {
      key: '/admin',
      icon: <Shield size={18} />,
      label: '系统管理',
      path: '/admin',
      permission: 'admin:view',
      description: '用户、权限和系统配置',
    },
  ],
};

interface SidebarProps {
  collapsed: boolean;
  onCollapse: (collapsed: boolean) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({ collapsed, onCollapse }) => {
  const { token } = theme.useToken();
  const router = useRouter();
  const pathname = usePathname();
  const { user } = useAuthStore();

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
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '2px 0',
          }}
        >
          <span style={{ fontSize: '13px', fontWeight: '500' }}>{item.label}</span>
          {item.badge && (
            <Badge
              count={item.badge}
              size='small'
              style={{
                backgroundColor: '#10b981',
                fontSize: '9px',
                lineHeight: '10px',
                fontWeight: '600',
              }}
            />
          )}
        </div>
      ),
      onClick: () => handleMenuClick({ key: item.key }),
      style: {
        margin: '2px 8px',
        borderRadius: '6px',
        padding: '6px 10px',
        height: 'auto',
      },
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
      style={{
        overflow: 'auto',
        height: '100vh',
        position: 'fixed',
        left: 0,
        top: 0,
        bottom: 0,
        background: 'linear-gradient(180deg, #ffffff 0%, #f8fafc 100%)',
        borderRight: `1px solid ${token.colorBorder}`,
        boxShadow: LAYOUT_CONFIG.shadows.md,
        zIndex: LAYOUT_CONFIG.zIndex.sider,
      }}
    >
      {/* Logo 区域 - 高度与 Header 一致 (64px) */}
      <div
        style={{
          height: LAYOUT_CONFIG.sider.logoAreaHeight,
          display: 'flex',
          alignItems: 'center',
          justifyContent: collapsed ? 'center' : 'flex-start',
          padding: collapsed 
            ? LAYOUT_CONFIG.sider.logoPaddingCollapsed 
            : LAYOUT_CONFIG.sider.logoPadding,
          borderBottom: `1px solid ${token.colorBorder}`,
          background: 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)',
          boxShadow: LAYOUT_CONFIG.shadows.sm,
        }}
      >
        <div
          style={{
            width: LAYOUT_CONFIG.sider.logoIconSize,
            height: LAYOUT_CONFIG.sider.logoIconSize,
            background: 'rgba(255, 255, 255, 0.2)',
            borderRadius: LAYOUT_CONFIG.borderRadius.lg,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: 'white',
            fontWeight: 'bold',
            fontSize: `${LAYOUT_CONFIG.header.fontSize}px`,
            backdropFilter: 'blur(10px)',
            border: '1px solid rgba(255, 255, 255, 0.3)',
          }}
        >
          IT
        </div>
        {!collapsed && (
          <div style={{ marginLeft: LAYOUT_CONFIG.spacing.md }}>
            <div
              style={{
                fontSize: `${LAYOUT_CONFIG.sider.logoTextSize}px`,
                fontWeight: '700',
                color: 'white',
                lineHeight: '1',
              }}
            >
              ITSM
            </div>
            <div
              style={{
                fontSize: '10px',
                color: 'rgba(255, 255, 255, 0.8)',
                textTransform: 'uppercase',
                letterSpacing: '1px',
                marginTop: '2px',
              }}
            >
              System
            </div>
          </div>
        )}
      </div>

      {/* 主菜单 */}
      <div style={{ 
        padding: LAYOUT_CONFIG.sider.menuPadding, 
        overflowY: 'auto', 
        maxHeight: `calc(100vh - ${LAYOUT_CONFIG.sider.logoAreaHeight + 120}px)` 
      }}>
        <Menu
          mode='inline'
          selectedKeys={[pathname]}
          style={{
            border: 'none',
            background: 'transparent',
          }}
          items={renderMenuItems(MENU_CONFIG.main)}
          theme='light'
          className='custom-menu'
        />
      </div>

      {/* 管理员菜单 */}
      {isAdmin && (
        <div style={{ marginTop: 'auto', paddingTop: '12px' }}>
          {!collapsed && (
            <div
              style={{
                padding: '6px 16px',
                fontSize: '10px',
                color: token.colorTextSecondary,
                fontWeight: '600',
                textTransform: 'uppercase',
                letterSpacing: '0.5px',
              }}
            >
              管理功能
            </div>
          )}
          <div
            style={{
              padding: '6px 8px',
              borderTop: `1px solid ${token.colorBorder}`,
            }}
          >
            <Menu
              mode='inline'
              selectedKeys={[pathname]}
              style={{
                border: 'none',
                background: 'transparent',
              }}
              items={renderMenuItems(MENU_CONFIG.admin)}
              theme='light'
            />
          </div>
        </div>
      )}

      {/* 底部用户信息 */}
      {!collapsed && (
        <div
          style={{
            padding: '12px 16px',
            borderTop: `1px solid ${token.colorBorder}`,
            background: 'rgba(0, 0, 0, 0.02)',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <div
              style={{
                width: 28,
                height: 28,
                borderRadius: '50%',
                background: 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: 'white',
                fontWeight: '600',
                fontSize: '12px',
              }}
            >
              {user?.name?.[0] || user?.username?.[0] || 'U'}
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div
                style={{
                  fontSize: '13px',
                  fontWeight: '600',
                  color: token.colorText,
                  marginBottom: '2px',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  lineHeight: '1.2',
                }}
              >
                {user?.name || user?.username}
              </div>
              <div
                style={{
                  fontSize: '11px',
                  color: token.colorTextSecondary,
                  textTransform: 'capitalize',
                  lineHeight: '1.2',
                }}
              >
                {user?.role || 'user'}
              </div>
            </div>
          </div>
        </div>
      )}
    </Sider>
  );
};
