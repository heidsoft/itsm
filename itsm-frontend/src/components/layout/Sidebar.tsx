'use client';

import React, { useState, useEffect } from 'react';
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
  Rocket,
  Monitor,
  Key,
  Workflow,
  Play,
  Settings,
  History,
  Users,
  Activity,
  Clock,
  ClipboardList,
  Bot,
} from 'lucide-react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore, useAuthStoreHydration } from '@/lib/store/auth-store';
import { LAYOUT_CONFIG } from '@/config/layout.config';
import styles from './Sidebar.module.css';
import { useI18n } from '@/lib/i18n';
import { getUserMenus, MenuItem as MenuItemType, MenuTreeResponse } from '@/lib/api/menu-api';

const { Sider } = Layout;

// 图标样式，统一大小
const iconStyle = { width: 16, height: 16 };

// 菜单项接口定义
interface MenuItem {
  key: string;
  icon: React.ReactNode;
  label: string | React.ReactNode;
  path?: string;
  permission?: string;
  description?: string;
  badge?: string;
  children?: MenuItem[];
}

// 菜单配置接口
interface MenuConfig {
  main: MenuItem[];
  admin: MenuItem[];
}

// 获取菜单配置 - ITIL v4 标准结构
const getMenuConfig = (t: (key: string, params?: Record<string, string | number>) => string): MenuConfig => ({
  main: [
    // ===== 服务运营 =====
    {
      key: '/dashboard',
      icon: <LayoutDashboard style={iconStyle} />,
      label: '服务台',
      path: '/dashboard',
      permission: 'dashboard:view',
      description: '服务台概览',
    },
    {
      key: '/service-requests',
      icon: <FileText style={iconStyle} />,
      label: '服务请求',
      path: '/service-requests',
      permission: 'ticket:read',
      description: '服务请求管理',
      children: [
        { key: '/service-requests', icon: <FileText style={iconStyle} />, label: '服务请求', path: '/service-requests', permission: 'ticket:read' },
        { key: '/tickets/templates', icon: <FileText style={iconStyle} />, label: '服务请求模板', path: '/tickets/templates', permission: 'ticket:write' },
        { key: '/tickets/analytics', icon: <BarChart3 style={iconStyle} />, label: '工单统计', path: '/tickets/analytics', permission: 'ticket:read' },
      ],
    },
    {
      key: '/my-requests',
      icon: <User style={iconStyle} />,
      label: '我的请求',
      path: '/my-requests',
      permission: 'ticket:read',
      description: '我的服务请求',
    },
    {
      key: '/incidents',
      icon: <AlertCircle style={iconStyle} />,
      label: '事件管理',
      path: '/incidents',
      permission: 'incident:read',
      description: '事件管理',
      children: [
        { key: '/incidents', icon: <AlertCircle style={iconStyle} />, label: '事件列表', path: '/incidents', permission: 'incident:read' },
        { key: '/incidents/major', icon: <AlertTriangle style={iconStyle} />, label: '重大事件', path: '/incidents/major', permission: 'incident:read' },
        { key: '/incidents/analytics', icon: <TrendingUp style={iconStyle} />, label: '事件分析', path: '/incidents/analytics', permission: 'incident:read' },
      ],
    },
    {
      key: '/problems',
      icon: <HelpCircle style={iconStyle} />,
      label: '问题管理',
      path: '/problems',
      permission: 'problem:read',
      description: '问题管理',
      children: [
        { key: '/problems', icon: <HelpCircle style={iconStyle} />, label: '问题列表', path: '/problems', permission: 'problem:read' },
        { key: '/problems/known-errors', icon: <AlertCircle style={iconStyle} />, label: '已知错误', path: '/problems/known-errors', permission: 'problem:read' },
        { key: '/problems/root-cause', icon: <Search style={iconStyle} />, label: '根因分析', path: '/problems/root-cause', permission: 'problem:read' },
      ],
    },
    {
      key: '/changes',
      icon: <BarChart3 style={iconStyle} />,
      label: '变更管理',
      path: '/changes',
      permission: 'change:read',
      description: '变更管理',
      children: [
        { key: '/changes', icon: <BarChart3 style={iconStyle} />, label: '变更列表', path: '/changes', permission: 'change:read' },
        { key: '/changes/standard', icon: <CheckCircle style={iconStyle} />, label: '标准变更', path: '/changes/standard', permission: 'change:read' },
        { key: '/changes/emergency', icon: <Zap style={iconStyle} />, label: '紧急变更', path: '/changes/emergency', permission: 'change:read' },
        { key: '/changes/approvals', icon: <GitMerge style={iconStyle} />, label: '变更审批', path: '/changes/approvals', permission: 'change:read' },
      ],
    },
    // ===== 服务保障 =====
    {
      key: '/knowledge',
      icon: <Book style={iconStyle} />,
      label: '知识库',
      path: '/knowledge',
      permission: 'knowledge:read',
      description: '知识库管理',
      children: [
        { key: '/knowledge', icon: <Book style={iconStyle} />, label: '知识库', path: '/knowledge', permission: 'knowledge:read' },
        { key: '/knowledge/articles', icon: <FileText style={iconStyle} />, label: '文章管理', path: '/knowledge/articles', permission: 'knowledge:write' },
        { key: '/knowledge/categories', icon: <Tag style={iconStyle} />, label: '文章分类', path: '/knowledge/categories', permission: 'knowledge:write' },
      ],
    },
    {
      key: '/service-catalog',
      icon: <BookOpen style={iconStyle} />,
      label: '服务目录',
      path: '/service-catalog',
      permission: 'service:read',
      description: '服务目录',
      children: [
        { key: '/service-catalog', icon: <BookOpen style={iconStyle} />, label: '服务目录', path: '/service-catalog', permission: 'service:read' },
        { key: '/admin/service-catalogs', icon: <List style={iconStyle} />, label: '服务项管理', path: '/admin/service-catalogs', permission: 'service:write' },
      ],
    },
    {
      key: '/cmdb',
      icon: <Database style={iconStyle} />,
      label: 'CMDB',
      path: '/cmdb',
      permission: 'cmdb:read',
      description: '配置管理数据库',
      children: [
        { key: '/cmdb', icon: <Server style={iconStyle} />, label: '配置项列表', path: '/cmdb/cis', permission: 'cmdb:read' },
        { key: '/admin/cmdb-types', icon: <Database style={iconStyle} />, label: 'CI类型管理', path: '/admin/cmdb-types', permission: 'cmdb:write' },
        { key: '/cmdb/cloud-resources', icon: <Cloud style={iconStyle} />, label: '云资源', path: '/cmdb/cloud-resources', permission: 'cmdb:read' },
        { key: '/cmdb/cloud-accounts', icon: <Key style={iconStyle} />, label: '云账号', path: '/cmdb/cloud-accounts', permission: 'cmdb:read' },
        { key: '/cmdb/cloud-services', icon: <Boxes style={iconStyle} />, label: '云服务', path: '/cmdb/cloud-services', permission: 'cmdb:read' },
        { key: '/cmdb/reconciliation', icon: <RefreshCw style={iconStyle} />, label: '同步对账', path: '/cmdb/reconciliation', permission: 'cmdb:write' },
      ],
    },
    // ===== 报告分析 =====
    {
      key: '/sla-dashboard',
      icon: <Calendar style={iconStyle} />,
      label: 'SLA管理',
      path: '/sla-dashboard',
      permission: 'sla:read',
      description: 'SLA监控与配置',
      children: [
        { key: '/sla-dashboard', icon: <Activity style={iconStyle} />, label: 'SLA监控', path: '/sla-dashboard', permission: 'sla:read' },
        { key: '/admin/sla-definitions', icon: <Calendar style={iconStyle} />, label: 'SLA配置', path: '/admin/sla-definitions', permission: 'sla:write' },
        { key: '/admin/escalation-rules', icon: <ArrowUpCircle style={iconStyle} />, label: '升级规则', path: '/admin/escalation-rules', permission: 'sla:write' },
      ],
    },
    {
      key: '/reports',
      icon: <TrendingUp style={iconStyle} />,
      label: '报表中心',
      path: '/reports',
      permission: 'report:read',
      description: '报表与分析',
      children: [
        { key: '/reports', icon: <TrendingUp style={iconStyle} />, label: '报表中心', path: '/reports', permission: 'report:read' },
        { key: '/reports/tickets', icon: <FileText style={iconStyle} />, label: '工单报表', path: '/reports/tickets', permission: 'report:read' },
        { key: '/reports/incidents', icon: <AlertCircle style={iconStyle} />, label: '事件报表', path: '/reports/incidents', permission: 'report:read' },
        { key: '/reports/problems', icon: <HelpCircle style={iconStyle} />, label: '问题报表', path: '/reports/problems', permission: 'report:read' },
        { key: '/reports/sla-performance', icon: <Calendar style={iconStyle} />, label: 'SLA报表', path: '/reports/sla-performance', permission: 'report:read' },
        { key: '/reports/cmdb-quality', icon: <Database style={iconStyle} />, label: 'CMDB质量', path: '/reports/cmdb-quality', permission: 'report:read' },
      ],
    },
    // ===== 自动化 =====
    {
      key: '/workflow',
      icon: <GitMerge style={iconStyle} />,
      label: '工作流',
      path: '/workflow',
      permission: 'workflow:read',
      description: '工作流自动化',
      children: [
        { key: '/workflow', icon: <List style={iconStyle} />, label: '工作流列表', path: '/workflow', permission: 'workflow:read' },
        { key: '/workflow/designer', icon: <Edit style={iconStyle} />, label: '流程设计器', path: '/workflow/designer', permission: 'workflow:write' },
        { key: '/workflow/instances', icon: <Play style={iconStyle} />, label: '流程实例', path: '/workflow/instances', permission: 'workflow:read' },
        { key: '/workflow/versions', icon: <History style={iconStyle} />, label: '版本管理', path: '/workflow/versions', permission: 'workflow:write' },
        { key: '/workflow/dashboard', icon: <Activity style={iconStyle} />, label: '监控仪表盘', path: '/workflow/dashboard', permission: 'workflow:read' },
        { key: '/workflow/audit', icon: <ClipboardList style={iconStyle} />, label: '审计日志', path: '/workflow/audit', permission: 'workflow:read' },
        { key: '/workflow/ticket-approval', icon: <Zap style={iconStyle} />, label: '工单自动化', path: '/workflow/ticket-approval', permission: 'workflow:write' },
        { key: '/workflow/automation', icon: <Settings style={iconStyle} />, label: '自动化规则', path: '/workflow/automation', permission: 'workflow:write' },
      ],
    },
    // ===== 工具 =====
    {
      key: '/assets',
      icon: <Monitor style={iconStyle} />,
      label: '资产管理',
      path: '/assets',
      permission: 'asset:read',
      description: 'IT资产管理',
      children: [
        { key: '/assets', icon: <Server style={iconStyle} />, label: '资产列表', path: '/assets', permission: 'asset:read' },
        { key: '/licenses', icon: <Key style={iconStyle} />, label: '许可证管理', path: '/licenses', permission: 'license:read' },
        { key: '/assets/categories', icon: <Tag style={iconStyle} />, label: '资产分类', path: '/assets/categories', permission: 'asset:write' },
      ],
    },
    {
      key: '/releases',
      icon: <Rocket style={iconStyle} />,
      label: '发布管理',
      path: '/releases',
      permission: 'release:read',
      description: '发布管理',
      children: [
        { key: '/releases', icon: <Calendar style={iconStyle} />, label: '发布计划', path: '/releases', permission: 'release:read' },
        { key: '/releases/new', icon: <Plus style={iconStyle} />, label: '新建发布', path: '/releases/new', permission: 'release:write' },
      ],
    },
    {
      key: '/ai/chat',
      icon: <Bot style={iconStyle} />,
      label: 'AI助手',
      path: '/ai/chat',
      permission: 'ai:view',
      description: 'AI 智能助手',
    },
  ],
  admin: [
    // ===== 客户管理 =====
    {
      key: '/msp',
      icon: <Building style={iconStyle} />,
      label: '客户管理',
      path: '/msp',
      permission: 'msp:read',
      description: '客户管理',
      children: [
        { key: '/msp', icon: <LayoutDashboard style={iconStyle} />, label: '客户仪表盘', path: '/msp', permission: 'msp:read' },
        { key: '/msp/management', icon: <Settings style={iconStyle} />, label: '分配管理', path: '/msp/management', permission: 'msp:manage' },
      ],
    },
    // ===== 发布管理 =====
    {
      key: '/releases-admin',
      icon: <Rocket style={iconStyle} />,
      label: '发布管理',
      path: '/releases',
      permission: 'release:read',
      description: '发布管理',
      children: [
        { key: '/releases', icon: <Calendar style={iconStyle} />, label: '发布计划', path: '/releases', permission: 'release:read' },
        { key: '/releases/new', icon: <Plus style={iconStyle} />, label: '新建发布', path: '/releases/new', permission: 'release:write' },
      ],
    },
    // ===== 系统管理 =====
    {
      key: '/admin',
      icon: <Settings style={iconStyle} />,
      label: '系统管理',
      path: '/admin',
      permission: 'admin:write',
      description: '系统管理',
      children: [
        { key: '/admin', icon: <LayoutDashboard style={iconStyle} />, label: '系统概览', path: '/admin', permission: 'admin:write' },
        { key: '/admin/users', icon: <Users style={iconStyle} />, label: '用户管理', path: '/admin/users', permission: 'user:read' },
        { key: '/admin/roles', icon: <Shield style={iconStyle} />, label: '角色管理', path: '/admin/roles', permission: 'role:read' },
        { key: '/admin/groups', icon: <Users style={iconStyle} />, label: '组管理', path: '/admin/groups', permission: 'group:read' },
        { key: '/admin/tenants', icon: <Building style={iconStyle} />, label: '租户管理', path: '/admin/tenants', permission: 'admin:write' },
        { key: '/enterprise/departments', icon: <Building style={iconStyle} />, label: '部门管理', path: '/enterprise/departments', permission: 'department:read' },
        { key: '/enterprise/teams', icon: <Users style={iconStyle} />, label: '团队管理', path: '/enterprise/teams', permission: 'team:read' },
        { key: '/admin/ticket-categories', icon: <Tag style={iconStyle} />, label: '工单分类', path: '/admin/ticket-categories', permission: 'ticket:write' },
        { key: '/admin/tickets/assignment-rules', icon: <GitBranch style={iconStyle} />, label: '工单分配规则', path: '/admin/tickets/assignment-rules', permission: 'ticket:write' },
        { key: '/admin/tickets/automation-rules', icon: <Zap style={iconStyle} />, label: '自动化规则', path: '/admin/tickets/automation-rules', permission: 'ticket:write' },
        { key: '/admin/approvals', icon: <GitMerge style={iconStyle} />, label: '审批管理', path: '/admin/approvals', permission: 'approval:read' },
        { key: '/admin/approval-chains', icon: <Link style={iconStyle} />, label: '审批链管理', path: '/admin/approval-chains', permission: 'admin:write' },
        { key: '/admin/permissions', icon: <Lock style={iconStyle} />, label: '权限管理', path: '/admin/permissions', permission: 'admin:write' },
        { key: '/admin/system-config', icon: <Settings style={iconStyle} />, label: '系统配置', path: '/admin/system-config', permission: 'system:write' },
      ],
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
  // 手动触发 auth store 的 hydration
  useAuthStoreHydration();
  const MENU_CONFIG = getMenuConfig(t);

  // 动态菜单状态
  const [dynamicMenus, setDynamicMenus] = useState<MenuTreeResponse | null>(null);
  const [menuLoading, setMenuLoading] = useState(false);

  // 加载用户菜单
  useEffect(() => {
    const loadMenus = async () => {
      if (!user) return;

      try {
        setMenuLoading(true);
        const menus = await getUserMenus();
        if (menus && (menus.main?.length > 0 || menus.admin?.length > 0)) {
          setDynamicMenus(menus);
        }
      } catch (error) {
        console.warn('Failed to load dynamic menus, using fallback:', error);
      } finally {
        setMenuLoading(false);
      }
    };

    loadMenus();
  }, [user]);

  // 将API菜单转换为Sidebar格式
  const convertApiMenuToSidebar = (menus: MenuItemType[]): MenuItem[] => {
    return menus.map(menu => {
      const icon = getIconByName(menu.icon);
      const item: MenuItem = {
        key: menu.path,
        icon,
        label: menu.name,
        path: menu.path,
        permission: menu.permission_code,
        description: menu.description,
        children: menu.children ? convertApiMenuToSidebar(menu.children) : undefined,
      };
      return item;
    });
  };

  // 根据图标名称获取图标组件
  const getIconByName = (iconName?: string) => {
    if (!iconName) return undefined;
    const iconMap: Record<string, React.ReactNode> = {
      LayoutDashboard: <LayoutDashboard style={iconStyle} />,
      FileText: <FileText style={iconStyle} />,
      AlertCircle: <AlertCircle style={iconStyle} />,
      HelpCircle: <HelpCircle style={iconStyle} />,
      BarChart3: <BarChart3 style={iconStyle} />,
      Database: <Database style={iconStyle} />,
      Book: <Book style={iconStyle} />,
      Calendar: <Calendar style={iconStyle} />,
      GitMerge: <GitMerge style={iconStyle} />,
      Shield: <Shield style={iconStyle} />,
      TrendingUp: <TrendingUp style={iconStyle} />,
      Rocket: <Rocket style={iconStyle} />,
      Monitor: <Monitor style={iconStyle} />,
      Key: <Key style={iconStyle} />,
      Workflow: <Workflow style={iconStyle} />,
      Play: <Play style={iconStyle} />,
      Settings: <Settings style={iconStyle} />,
      History: <History style={iconStyle} />,
      Users: <Users style={iconStyle} />,
      Activity: <Activity style={iconStyle} />,
      Clock: <Clock style={iconStyle} />,
      ClipboardList: <ClipboardList style={iconStyle} />,
      Bot: <Bot style={iconStyle} />,
    };
    return iconMap[iconName];
  };

  // 确定使用哪个菜单配置 - 优先使用数据库动态菜单
  const useDynamicMenu = !!dynamicMenus;
  const mainMenus = useDynamicMenu ? convertApiMenuToSidebar(dynamicMenus.main) : MENU_CONFIG.main;
  const adminMenus = useDynamicMenu ? convertApiMenuToSidebar(dynamicMenus.admin) : MENU_CONFIG.admin;

  const handleMenuClick = ({ key }: { key: string }) => {
    router.push(key);
  };

  const isAdmin = user?.role === 'admin' || user?.role === 'super_admin';

  // 渲染菜单项，支持徽章和描述
  const renderMenuItems = (items: MenuItem[]) => {
    return items.map(item => {
      // 如果有子菜单
      if (item.children) {
        return {
          key: item.key,
          icon: item.icon,
          label: (
            <div className={styles.menuItemLabel} title={typeof item.description === 'string' ? item.description : typeof item.label === 'string' ? item.label : undefined}>
              <span className="truncate">{item.label}</span>
            </div>
          ),
          children: item.children.map((child: MenuItem) => ({
            key: child.key,
            icon: child.icon,
            label: (
              <div className={styles.menuItemLabel}>
                <span className="truncate">{child.label}</span>
              </div>
            ),
            onClick: () => handleMenuClick({ key: child.key }),
          })),
        };
      }

      return {
        key: item.key,
        icon: item.icon,
        label: (
          <div className={styles.menuItemLabel} title={typeof item.description === 'string' ? item.description : typeof item.label === 'string' ? item.label : undefined}>
            <span className="truncate">{item.label}</span>
            {item.badge && (
              <Badge count={item.badge} size="small" className={styles.menuItemBadge} />
            )}
          </div>
        ),
        onClick: () => handleMenuClick({ key: item.key }),
        className: styles.menuItem,
      };
    });
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
          mode="inline"
          inlineIndent={24}
          selectedKeys={[pathname]}
          className={styles.customMenu}
          items={renderMenuItems(mainMenus)}
          theme="light"
        />
      </div>

      {/* 管理员菜单 */}
      {isAdmin && (
        <div className={styles.adminMenuContainer}>
          {!collapsed && <div className={styles.adminMenuHeader}>管理功能</div>}
          <div className={styles.adminMenu}>
            <Menu
              mode="inline"
              inlineIndent={24}
              selectedKeys={[pathname]}
              className={styles.customMenu}
              items={renderMenuItems(adminMenus)}
              theme="light"
            />
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
